package fold

import (
	"maps"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
)

// The buy scopes a whole-match row is rated over (Core-planning#589, #590).
const (
	scopePistol = iota
	scopeEco
	scopeForce
	scopeFull
	scopeAntiEco
	scopes
)

// buyScope is a row set like a side's: the player's rounds in the scope, with the rating inputs the
// whole-match row carries for its side scopes.
type buyScope struct {
	rows           map[uint64]*cscstats.PlayerStats
	teamsWinPoints map[uint64]float64
	normalizer     map[uint64]int
}

// rateBuys rates every whole-match row over the rounds of each of its team's buys, as rate rates a
// side: the main CSC formula against the scope's own averages.
func rateBuys(g *cscstats.Game) {
	var bs [scopes]buyScope
	for i := range bs {
		bs[i] = buyScope{map[uint64]*cscstats.PlayerStats{}, map[uint64]float64{}, map[uint64]int{}}
	}
	for _, r := range g.Rounds {
		for _, steam := range slices.Sorted(maps.Keys(r.PlayerStats)) {
			p := r.PlayerStats[steam]
			team := r.TeamStats[p.TeamClanName]
			for _, i := range buyScopes(team.Buy, opponentBuy(r, p.TeamClanName)) {
				s := bs[i]
				add(row(s.rows, steam, p), p)
				s.teamsWinPoints[steam] += team.WinPoints
				if p.Side == sideT {
					s.normalizer[steam] += r.InitTerroristCount
				} else {
					s.normalizer[steam] += r.InitCTerroristCount
				}
			}
		}
	}
	for i, s := range bs {
		derive(s.rows, g.TickRate)
		a := average(s.rows)
		for _, steam := range slices.Sorted(maps.Keys(s.rows)) {
			row := s.rows[steam]
			_, rating := rating(row, a, s.teamsWinPoints[steam], s.normalizer[steam])
			w := g.TotalPlayerStats[steam]
			switch i {
			case scopePistol:
				w.PistolRounds, w.PistolRoundsWon, w.PistolRoundRating = row.Rounds, row.RF, rating
			case scopeEco:
				w.EcoBuyRounds, w.EcoBuyRoundsWon, w.EcoBuyRating = row.Rounds, row.RF, rating
			case scopeForce:
				w.ForceBuyRounds, w.ForceBuyRoundsWon, w.ForceBuyRating = row.Rounds, row.RF, rating
			case scopeFull:
				w.FullBuyRounds, w.FullBuyRoundsWon, w.FullBuyRating = row.Rounds, row.RF, rating
			case scopeAntiEco:
				w.AntiEcoRounds, w.AntiEcoRoundsWon, w.AntiEcoRating = row.Rounds, row.RF, rating
			}
		}
	}
}

// buyScopes are the scopes a round of the team's buy counts in.
func buyScopes(own, opponent cscstats.Buy) []int {
	switch own {
	case cscstats.BuyPistol:
		return []int{scopePistol}
	case cscstats.BuyEco:
		return []int{scopeEco}
	case cscstats.BuyForce:
		return []int{scopeForce}
	case cscstats.BuyFull:
		if opponent == cscstats.BuyEco {
			return []int{scopeFull, scopeAntiEco}
		}
		return []int{scopeFull}
	}
	return nil
}

func opponentBuy(r *cscstats.Round, clan string) cscstats.Buy {
	for name, t := range r.TeamStats {
		if name != clan {
			return t.Buy
		}
	}
	return cscstats.BuyUnknown
}
