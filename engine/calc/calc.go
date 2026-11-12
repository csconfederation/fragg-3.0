// Package calc holds the stat calculators that replace the collectors (Core-planning#681). Each
// reads one committed round, as the Deriver handed it over, and writes only its own family of the
// round's row model. The payload doesn't call them until the fold switches (PR 17); cmd/calcdiff
// compares them with the collectors meanwhile.
package calc

import (
	"github.com/csconfederation/fragg-3.0/engine/calc/combat"
	"github.com/csconfederation/fragg-3.0/engine/calc/economy"
	"github.com/csconfederation/fragg-3.0/engine/calc/positioning"
	"github.com/csconfederation/fragg-3.0/engine/calc/survival"
	"github.com/csconfederation/fragg-3.0/engine/calc/swing"
	"github.com/csconfederation/fragg-3.0/engine/calc/utility"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc is one family's calculator. It keeps nothing between calls, reads only constants, reads
// r.Events[:r.Commit] without the Dropped ones, and applies each Win at its r.Wins index. It
// can't fail: the Deriver made every check.
type Calc func(r *round.Round, out *row.Round)

// All run in this order: the tier 1 calculators, which read only r, then tier 2, which also read
// tier 1's results in out (design note section 3). The families land one per PR.
var All = []Calc{
	combat.Calc,
	utility.Calc,
	survival.Calc,
	economy.Calc,
	positioning.Calc,
	swing.Calc,
}

// Run is r's rows: one empty player row per member, then every calculator.
func Run(r *round.Round) *row.Round {
	out := &row.Round{Players: map[uint64]*row.Player{},
		Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil {
			out.Players[e.Join.Steam64] = &row.Player{}
		}
	}
	for _, c := range All {
		c(r, out)
	}
	return out
}
