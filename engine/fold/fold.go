// Package fold folds a match's committed round rows into its whole-match, T and CT player rows and
// its team rows: one pass over the rounds, then the rates, the ratings and the projections
// (design note section 5). Every sum runs first to last round, every map in sorted order, so two
// runs are byte-identical.
package fold

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

const total = 4 // the Side of a whole-match row

// Match is a match's folded rows: each player's whole-match row and side rows, by Steam64, and each
// team's match row by clan.
type Match struct {
	Whole, T, CT map[uint64]*row.Player
	Teams        map[string]*row.Team
}

// tally is what eco's ratings count per row that the family sums don't: rounds by kill count, rounds
// survived and rounds with a death.
type tally struct {
	multi          [6]int // rounds with exactly k eco kills, k 1-5
	survived, died int
}

// scope is a row set a rating is computed over. A side's or buy's rows carry no rating inputs of
// their own, so these are kept beside them, by player.
type scope struct {
	rows           map[uint64]*row.Player
	teamsWinPoints map[uint64]float64
	normalizer     map[uint64]int
}

func newScope() *scope {
	return &scope{map[uint64]*row.Player{}, map[uint64]float64{}, map[uint64]int{}}
}

// Fold folds rounds, the committed rounds' rows in order. A round row's constants aren't applied yet.
func Fold(rounds []*row.Round) (*Match, error) {
	m := &Match{Whole: map[uint64]*row.Player{}, T: map[uint64]*row.Player{}, CT: map[uint64]*row.Player{},
		Teams: map[string]*row.Team{}}
	tallies := map[*row.Player]*tally{}
	var buys [numBuys]*scope
	for i := range buys {
		buys[i] = newScope()
	}
	for _, r := range rounds {
		for _, id := range slices.Sorted(maps.Keys(r.Players)) {
			if p := r.Players[id]; p.Side != int(fact.SideT) && p.Side != int(fact.SideCT) {
				return nil, &match.InvariantError{Tick: r.EndingTick, Reason: fmt.Sprintf("round %d: player %s has side %d", r.RoundNum, p.SteamID, p.Side)}
			}
		}
		m.accumulate(r, tallies, buys)
	}

	// F2: rates, for every set.
	for _, set := range []map[uint64]*row.Player{m.Whole, m.T, m.CT} {
		derive(set)
	}
	for _, s := range buys {
		derive(s.rows)
	}

	// F3: ratings. Pass 1 is each set's averages, pass 2 the ratings against them.
	all, t, ct := average(m.Whole), average(m.T), average(m.CT)
	for _, id := range slices.Sorted(maps.Keys(m.Whole)) {
		p := m.Whole[id]
		p.ImpactRating, p.Rating = cscRating(p, all, p.TeamsWinPoints, p.WinPointsNormalizer)
		if s := m.CT[id]; s != nil {
			s.ImpactRating, s.Rating = cscRating(s, ct, p.CtTeamsWinPoints, p.CtWinPointsNormalizer)
		}
		if s := m.T[id]; s != nil {
			s.ImpactRating, s.Rating = cscRating(s, t, p.TTeamsWinPoints, p.TWinPointsNormalizer)
		}
	}
	rateBuys(m, buys)
	for _, id := range slices.Sorted(maps.Keys(m.Whole)) {
		m.eco(id, tallies)
	}

	// F4: the whole row's T and CT columns, from the side rows.
	for _, id := range slices.Sorted(maps.Keys(m.Whole)) {
		project(m.Whole[id], m.T[id], m.CT[id])
	}

	// The team rows' totals from their players' whole-match rows, by each player's last clan.
	for _, id := range slices.Sorted(maps.Keys(m.Whole)) {
		p := m.Whole[id]
		if team := m.Teams[p.TeamClanName]; team != nil {
			team.Util += p.SmokeThrown + p.FlashThrown + p.NadesThrown + p.FiresThrown
			team.Ud += p.UtilDmg
			team.Ef += p.Ef
			team.Fass += p.FAss
			team.Traded += p.Traded
			team.Deaths += p.Deaths
		}
	}
	return m, nil
}

// accumulate adds a round to the team rows, the players' whole-match and side rows, and the buy
// scopes its teams' buys put each player in. A row's name and team are those of its latest round.
func (m *Match) accumulate(r *row.Round, tallies map[*row.Player]*tally, buys [numBuys]*scope) {
	teams := [2]*row.Team{r.Teams[fact.SideT], r.Teams[fact.SideCT]}
	for _, src := range teams {
		if src.Clan == "" {
			continue
		}
		if m.Teams[src.Clan] == nil {
			m.Teams[src.Clan] = &row.Team{}
		}
		sum(m.Teams[src.Clan], src)
	}
	for _, id := range slices.Sorted(maps.Keys(r.Players)) {
		p := r.Players[id]
		sides := m.CT
		if p.Side == int(fact.SideT) {
			sides = m.T
		}
		whole, side := identify(m.Whole, id, p, tallies), identify(sides, id, p, tallies)
		count(whole, p, tallies)
		count(side, p, tallies)
		whole.Side, side.Side = total, p.Side
		if p.LurkRounds != 0 {
			side.Wlp += p.WinPoints
			if p.Side == int(fact.SideT) { // the whole row counts lurking on T rounds only
				whole.Wlp += p.WinPoints
			}
		}

		// Inputs to the whole-match row's ratings that the side rows don't carry.
		own, opp := teams[0], teams[1]
		if p.TeamClanName == teams[1].Clan {
			own, opp = teams[1], teams[0]
		}
		whole.TeamsWinPoints += own.WinPoints
		whole.TTeamsWinPoints += own.TWinPoints
		whole.CtTeamsWinPoints += own.CtWinPoints
		initial := r.InitCTCount
		if p.Side == int(fact.SideT) {
			initial = r.InitTerroristCount
			whole.TWinPointsNormalizer += initial
		} else {
			whole.CtWinPointsNormalizer += initial
			whole.CtAWP += p.CtAWP
		}
		whole.WinPointsNormalizer += initial

		for _, i := range buyScopes(own.Buy, opp.Buy) {
			s := buys[i]
			row := identify(s.rows, id, p, tallies)
			count(row, p, tallies)
			s.teamsWinPoints[id] += own.WinPoints
			s.normalizer[id] += initial
		}
	}
}

// identify is the row for id in set, made on first use, with p's name and team: later rounds
// overwrite them.
func identify(set map[uint64]*row.Player, id uint64, p *row.Player, tallies map[*row.Player]*tally) *row.Player {
	dst := set[id]
	if dst == nil {
		dst = &row.Player{}
		set[id] = dst
		tallies[dst] = &tally{}
	}
	dst.Name, dst.SteamID, dst.TeamClanName = p.Name, p.SteamID, p.TeamClanName
	return dst
}

// count adds one round row to dst: its family sums, the round, the bot flag, and its tally.
func count(dst, p *row.Player, tallies map[*row.Player]*tally) {
	sum(dst, p)
	dst.Rounds++
	dst.IsBot = dst.IsBot || p.IsBot
	t := tallies[dst]
	if k := p.EcoKills; k >= 1 && k <= 5 {
		t.multi[k]++
	}
	switch {
	case p.Deaths > 0:
		t.died++
	case p.LeftAlive == 0: // D44: a disconnect alive is neither a death nor a survival
		t.survived++
	}
}

// derive computes a row set's per-round rates.
func derive(rows map[uint64]*row.Player) {
	for _, id := range slices.Sorted(maps.Keys(rows)) {
		p := rows[id]
		// Average time to death (D20, D47): over the rounds the player died in, in seconds with its
		// fraction; survivors' rounds have no time to death (ticksAlive is set at death).
		if p.Deaths > 0 {
			p.Atd = float64(p.TicksAlive) / float64(p.Deaths) / fact.TickRate
		} else {
			p.Atd = 0
		}
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

// averages are a row set's match-wide per-round averages over its human rows: the shared bot row
// isn't a player to rate against (D40).
type averages struct{ impact, kills, deaths, kast, adr float64 }

func average(rows map[uint64]*row.Player) averages {
	var a averages
	n := 0
	for _, id := range slices.Sorted(maps.Keys(rows)) {
		p := rows[id]
		if p.IsBot {
			continue
		}
		n += p.Rounds
		a.impact += p.ImpactPoints
		a.kills += float64(p.Kills)
		a.deaths += float64(p.Deaths)
		a.kast += p.KastRounds
		a.adr += float64(p.Damage)
	}
	if n == 0 { // only bots: nothing to rate against, and a NaN would break the payload's JSON
		return a
	}
	a.impact /= float64(n)
	a.kills /= float64(n)
	a.deaths /= float64(n)
	a.kast /= float64(n)
	a.adr /= float64(n)
	return a
}

func cscRating(p *row.Player, a averages, teamsWinPoints float64, normalizer int) (impact, rating float64) {
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
func project(p, t, ct *row.Player) {
	if t != nil {
		p.TDamage, p.TImpactPoints, p.TWinPoints = t.Damage, t.ImpactPoints, t.WinPoints
		p.TOK, p.TOL, p.TKills, p.TDeaths = t.Ok, t.Ol, t.Kills, t.Deaths
		p.TKAST, p.TKASTRounds, p.TADR, p.TADP = t.Kast, t.KastRounds, t.Adr, t.DeathPlacement
		p.TRounds, p.TRF = t.Rounds, t.RF
		p.TRating, p.TImpactRating = t.Rating, t.ImpactRating
		p.EcoTProbabilitySwing, p.HLTVTRating, p.EcoTEcoRating = t.EcoProbabilitySwing, t.HLTVRating, t.SwingRating
	}
	if ct != nil {
		p.CtDamage, p.CtImpactPoints, p.CtWinPoints = ct.Damage, ct.ImpactPoints, ct.WinPoints
		p.CtOK, p.CtOL, p.CtKills, p.CtDeaths = ct.Ok, ct.Ol, ct.Kills, ct.Deaths
		p.CtKAST, p.CtKASTRounds, p.CtADR, p.CtADP = ct.Kast, ct.KastRounds, ct.Adr, ct.DeathPlacement
		p.CtRounds = ct.Rounds
		p.CtRating, p.CtImpactRating = ct.Rating, ct.ImpactRating
		p.EcoCTProbabilitySwing, p.HLTVCtRating, p.EcoCTEcoRating = ct.EcoProbabilitySwing, ct.HLTVRating, ct.SwingRating
	}
}
