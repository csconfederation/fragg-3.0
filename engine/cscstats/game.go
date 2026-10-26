package cscstats

import "github.com/csconfederation/fragg-3.0/engine/match"

// Game returns the committed rounds' CSC stats for engine/fold.
func (c *Collector) Game(res *match.Result) *Game {
	g := &Game{Rounds: make([]*round, 0, len(res.Rounds)), TickRate: tickRate}
	for _, r := range res.Rounds {
		g.Rounds = append(g.Rounds, c.rounds[r])
	}
	return g
}
