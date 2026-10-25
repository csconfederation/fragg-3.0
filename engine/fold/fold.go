// Package fold folds a match's committed rounds into its total, T and CT stats: the CSC round
// records once into three accumulators, the eco round records through eco's own fold, and both
// onto the payload's player rows (Core-planning#303). It replaces the oracle's end-of-match
// aggregation, its side re-aggregation and the eco merge.
package fold

import (
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
)

const (
	sideT  = 2
	sideCT = 3
	total  = 4 // the Side of a whole-match row
)

// Game folds g's committed rounds and the matching eco rounds into g's player and team stats.
func Game(g *cscstats.Game, e *eco.Collector, ecoRounds []*eco.Round) {
	g.TotalPlayerStats = map[uint64]*cscstats.PlayerStats{}
	g.TPlayerStats = map[uint64]*cscstats.PlayerStats{}
	g.CtPlayerStats = map[uint64]*cscstats.PlayerStats{}
	g.TotalTeamStats = map[string]*cscstats.TeamStats{}
	g.TotalWPAlog = []*cscstats.WPALog{}
	if len(g.Rounds) == 0 {
		return
	}
	g.EcoStatsOK = true // one model: eco covers every committed round and player
	for _, r := range g.Rounds {
		enrich(r)
	}
	accumulate(g)
	for _, side := range []map[uint64]*cscstats.PlayerStats{g.TotalPlayerStats, g.TPlayerStats, g.CtPlayerStats} {
		derive(side, g.TickRate)
	}
	rate(g)
	project(g)
	mergeEco(g, e.Players(ecoRounds))
}

// enrich computes a round's team win and impact points, its winning team's damage, and each
// winner's round win share; Stats stores the last two on the round rows.
func enrich(r *cscstats.Round) {
	for _, steam := range slices.Sorted(maps.Keys(r.PlayerStats)) {
		p := r.PlayerStats[steam]
		if p.RF == 1 {
			r.WinTeamDmg += p.Damage
		}
		team := r.TeamStats[p.TeamClanName]
		switch p.Side {
		case sideT:
			team.TWinPoints += p.WinPoints
			team.TImpactPoints += p.ImpactPoints
		case sideCT:
			team.CtWinPoints += p.WinPoints
			team.CtImpactPoints += p.ImpactPoints
		}
		team.WinPoints += p.WinPoints
		team.ImpactPoints += p.ImpactPoints
	}
	for _, p := range r.PlayerStats {
		if p.RF == 0 {
			continue
		}
		share := 100.0
		if r.EndDueToBombEvent {
			share = 70
		}
		p.Rws = share * (float64(p.Damage) / float64(r.WinTeamDmg))
		steam, _ := strconv.ParseUint(p.SteamID, 10, 64)
		if r.EndDueToBombEvent && (p.Side == sideT && r.Planter == steam || p.Side == sideCT && r.Defuser == steam) {
			p.Rws += 30
		}
		if math.IsNaN(p.Rws) {
			p.Rws = 0
		}
	}
}

// accumulate adds every round record to the player's whole-match row and to its side's row, last
// round first (so a player's name and team come from their last round), and sums the team totals.
func accumulate(g *cscstats.Game) {
	for i := len(g.Rounds) - 1; i >= 0; i-- {
		r := g.Rounds[i]
		g.TotalWPAlog = append(g.TotalWPAlog, r.WPAlog...)
		for _, name := range slices.Sorted(maps.Keys(r.TeamStats)) {
			src := r.TeamStats[name]
			if g.TotalTeamStats[name] == nil && name != "" {
				g.TotalTeamStats[name] = &cscstats.TeamStats{}
			}
			dst := g.TotalTeamStats[name]
			dst.Pistols += src.Pistols
			dst.PistolsW += src.PistolsW
			dst.FourVFiveS += src.FourVFiveS
			dst.FourVFiveW += src.FourVFiveW
			dst.FiveVFourS += src.FiveVFourS
			dst.FiveVFourW += src.FiveVFourW
			dst.Saves += src.Saves
			dst.Clutches += src.Clutches
			dst.CtR += src.CtR
			dst.CtRW += src.CtRW
			dst.TR += src.TR
			dst.TRW += src.TRW
		}
		for _, steam := range slices.Sorted(maps.Keys(r.PlayerStats)) {
			p := r.PlayerStats[steam]
			sides := g.CtPlayerStats
			if p.Side == sideT {
				sides = g.TPlayerStats
			}
			whole, side := row(g.TotalPlayerStats, steam, p), row(sides, steam, p)
			add(whole, p)
			add(side, p)
			whole.Side, side.Side = total, p.Side
			lurk(side, p)
			if p.Side == sideT { // the oracle's whole-match row counts lurking on T only
				lurk(whole, p)
			}

			// Inputs to the whole-match row's ratings that the side rows don't carry.
			team := r.TeamStats[p.TeamClanName]
			whole.TeamsWinPoints += team.WinPoints
			whole.TTeamsWinPoints += team.TWinPoints
			whole.CtTeamsWinPoints += team.CtWinPoints
			switch p.Side {
			case sideT:
				whole.WinPointsNormalizer += r.InitTerroristCount
				whole.TWinPointsNormalizer += r.InitTerroristCount
			case sideCT:
				whole.WinPointsNormalizer += r.InitCTerroristCount
				whole.CtWinPointsNormalizer += r.InitCTerroristCount
				whole.CtAWP += p.CtAWP
			}
		}
	}
	for _, steam := range slices.Sorted(maps.Keys(g.TotalPlayerStats)) {
		p := g.TotalPlayerStats[steam]
		team := g.TotalTeamStats[p.TeamClanName]
		team.Util += p.SmokeThrown + p.FlashThrown + p.NadesThrown + p.FiresThrown
		team.Ud += p.UtilDmg
		team.Ef += p.Ef
		team.Fass += p.FAss
		team.Traded += p.Traded
		team.Deaths += int(p.Deaths)
	}
}

func row(rows map[uint64]*cscstats.PlayerStats, steam uint64, p *cscstats.PlayerStats) *cscstats.PlayerStats {
	if rows[steam] == nil {
		rows[steam] = &cscstats.PlayerStats{Name: p.Name, SteamID: p.SteamID, TeamClanName: p.TeamClanName}
	}
	return rows[steam]
}

// add adds one round record to a row.
func add(dst, p *cscstats.PlayerStats) {
	dst.Rounds++
	dst.Kills += p.Kills
	dst.Assists += p.Assists
	dst.Deaths += p.Deaths
	dst.Damage += p.Damage
	dst.TicksAlive += p.TicksAlive
	dst.DeathPlacement += p.DeathPlacement
	dst.Trades += p.Trades
	dst.Traded += p.Traded
	dst.Ok += p.Ok
	dst.Ol += p.Ol
	dst.KillPoints += p.KillPoints
	dst.Cl_1 += p.Cl_1
	dst.Cl_2 += p.Cl_2
	dst.Cl_3 += p.Cl_3
	dst.Cl_4 += p.Cl_4
	dst.Cl_5 += p.Cl_5
	dst.TwoK += p.TwoK
	dst.ThreeK += p.ThreeK
	dst.FourK += p.FourK
	dst.FiveK += p.FiveK
	dst.NadeDmg += p.NadeDmg
	dst.InfernoDmg += p.InfernoDmg
	dst.UtilDmg += p.UtilDmg
	dst.Ef += p.Ef
	dst.FAss += p.FAss
	dst.EnemyFlashTime += p.EnemyFlashTime
	dst.Hs += p.Hs
	dst.KastRounds += p.KastRounds
	dst.Saves += p.Saves
	dst.Entries += p.Entries
	dst.ImpactPoints += p.ImpactPoints
	dst.WinPoints += p.WinPoints
	dst.AwpKills += p.AwpKills
	dst.RF += p.RF
	dst.RA += p.RA
	dst.NadesThrown += p.NadesThrown
	dst.SmokeThrown += p.SmokeThrown
	dst.FlashThrown += p.FlashThrown
	dst.FiresThrown += p.FiresThrown
	dst.DamageTaken += p.DamageTaken
	dst.SuppDamage += p.SuppDamage
	dst.SuppRounds += p.SuppRounds
	dst.Rwk += p.Rwk
	dst.Mip += p.Mip
	dst.Eac += p.Eac
	dst.Rws += p.Rws // 0 unless the player's team won the round
	dst.IsBot = dst.IsBot || p.IsBot
}

// lurk adds a round's lurk stats to a row.
func lurk(dst, p *cscstats.PlayerStats) {
	dst.LurkRounds += p.LurkRounds
	if p.LurkRounds != 0 {
		dst.Wlp += p.WinPoints
	}
}

// derive computes a row set's per-round rates.
func derive(rows map[uint64]*cscstats.PlayerStats, tickRate int) {
	for _, steam := range slices.Sorted(maps.Keys(rows)) {
		p := rows[steam]
		p.Atd = p.TicksAlive / p.Rounds / tickRate
		p.DeathPlacement = p.DeathPlacement / float64(p.Deaths)
		p.Kast = p.KastRounds / float64(p.Rounds)
		p.KillPointAvg = p.KillPoints / float64(p.Kills)
		if p.Kills == 0 {
			p.KillPointAvg = 0
		}
		p.Iiwr = p.WinPoints / p.ImpactPoints
		p.Adr = float64(p.Damage) / float64(p.Rounds)
		p.DrDiff = p.Adr - (float64(p.DamageTaken) / float64(p.Rounds))
		p.Tr = float64(p.Traded) / float64(p.Deaths)
		p.KR = float64(p.Kills) / float64(p.Rounds)
		p.UtilThrown = p.SmokeThrown + p.FlashThrown + p.NadesThrown + p.FiresThrown
		p.Rws = p.Rws / float64(p.Rounds)
		if math.IsNaN(p.Rws) {
			p.Rws = 0
		}
		if p.ImpactPoints == 0 {
			p.Iiwr = 0
		}
		if p.Deaths == 0 {
			p.DeathPlacement = 0
			p.Tr = .50
		}
	}
}

// averages are a row set's match-wide per-round averages.
type averages struct{ impact, kills, deaths, kast, adr float64 }

func average(rows map[uint64]*cscstats.PlayerStats) averages {
	var a averages
	n := 0
	for _, steam := range slices.Sorted(maps.Keys(rows)) {
		p := rows[steam]
		n += p.Rounds
		a.impact += p.ImpactPoints
		a.kills += float64(p.Kills)
		a.deaths += float64(p.Deaths)
		a.kast += p.KastRounds
		a.adr += float64(p.Damage)
	}
	a.impact /= float64(n)
	a.kills /= float64(n)
	a.deaths /= float64(n)
	a.kast /= float64(n)
	a.adr /= float64(n)
	return a
}

// rate computes the CSC rating of every row: one formula per scope, against that scope's averages.
func rate(g *cscstats.Game) {
	all, t, ct := average(g.TotalPlayerStats), average(g.TPlayerStats), average(g.CtPlayerStats)
	for _, steam := range slices.Sorted(maps.Keys(g.TotalPlayerStats)) {
		p := g.TotalPlayerStats[steam]
		p.ImpactRating, p.Rating = rating(p, all, p.TeamsWinPoints, p.WinPointsNormalizer)
		if s := g.CtPlayerStats[steam]; s != nil {
			s.ImpactRating, s.Rating = rating(s, ct, p.CtTeamsWinPoints, p.CtWinPointsNormalizer)
		}
		if s := g.TPlayerStats[steam]; s != nil {
			s.ImpactRating, s.Rating = rating(s, t, p.TTeamsWinPoints, p.TWinPointsNormalizer)
		}
	}
}

func rating(p *cscstats.PlayerStats, a averages, teamsWinPoints float64, normalizer int) (impact, rating float64) {
	openingFactor := (float64(p.Ok-p.Ol) / 13.0) + 1 //move from 13 to (rounds / 5)
	ipr := p.ImpactPoints / float64(p.Rounds)
	wpr := p.WinPoints / float64(p.Rounds)
	impact = (0.1 * openingFactor) + (0.6 * ratio(ipr, a.impact))
	if teamsWinPoints != 0 {
		impact += 0.3 * (wpr / (teamsWinPoints / float64(normalizer)))
	}
	dr := float64(p.Deaths) / float64(p.Rounds)
	death := 0.07 * (a.deaths / dr)
	if p.Deaths == 0 || death > 0.21 {
		death = 0.21
	}
	rating = (0.3 * impact) + (0.35 * ratio(p.KR, a.kills)) + death + (0.08 * ratio(p.Kast, a.kast)) + (0.2 * ratio(p.Adr, a.adr))
	return impact, rating
}

// ratio is a player's value relative to the match average; with no match-wide value (a lobby
// with no kills, say) it contributes 0 instead of a NaN (Core-planning#302, corpus finding 1).
func ratio(v, avg float64) float64 {
	if avg == 0 {
		return 0
	}
	return v / avg
}

// project copies each side row's stats onto the whole-match row's T and CT columns.
func project(g *cscstats.Game) {
	for steam, p := range g.TotalPlayerStats {
		if s := g.TPlayerStats[steam]; s != nil {
			p.TDamage, p.TImpactPoints, p.TWinPoints = s.Damage, s.ImpactPoints, s.WinPoints
			p.TOK, p.TOL, p.TKills, p.TDeaths = s.Ok, s.Ol, s.Kills, s.Deaths
			p.TKAST, p.TKASTRounds, p.TADR, p.TADP = s.Kast, s.KastRounds, s.Adr, s.DeathPlacement
			p.TRounds, p.TRF = s.Rounds, s.RF
			p.TRating, p.TImpactRating = s.Rating, s.ImpactRating
		}
		if s := g.CtPlayerStats[steam]; s != nil {
			p.CtDamage, p.CtImpactPoints, p.CtWinPoints = s.Damage, s.ImpactPoints, s.WinPoints
			p.CtOK, p.CtOL, p.CtKills, p.CtDeaths = s.Ok, s.Ol, s.Kills, s.Deaths
			p.CtKAST, p.CtKASTRounds, p.CtADR, p.CtADP = s.Kast, s.KastRounds, s.Adr, s.DeathPlacement
			p.CtRounds = s.Rounds
			p.CtRating, p.CtImpactRating = s.Rating, s.ImpactRating
		}
	}
}
