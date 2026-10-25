package cscstats

import "github.com/csconfederation/fragg-3.0/engine/match"

// Game builds the oracle's payload from the match result with the committed rounds' CSC stats;
// engine/fold aggregates them.
func (c *Collector) Game(res *match.Result) *Game {
	g := &Game{
		Rounds:                   make([]*round, 0, len(res.Rounds)),
		PotentialRound:           c.potential,
		Teams:                    map[string]*team{},
		MapName:                  res.Map,
		TickRate:                 tickRate,
		RoundsToWin:              mr + 1,
		WinnerClanName:           res.WinnerClan,
		TotalRounds:              res.TotalRounds,
		ReconnectedPlayers:       res.Reconnected,
		ConnectedAfterRoundStart: map[uint64]bool{},
	}
	if res.Outcome == match.Ended || res.Outcome == match.Partial {
		g.Result = "Ended"
	}
	for clan, score := range res.Scores {
		g.Teams[clan] = &team{Name: clan, Score: score}
	}
	for _, r := range res.Rounds {
		g.Rounds = append(g.Rounds, c.rounds[r])
	}
	return g
}
