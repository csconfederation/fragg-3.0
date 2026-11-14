// Package positioning is the positioning family: the lurk sampler's blips and the round's lurker.
package positioning

import (
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// isolated is the distance, in units (about 25 m), a T's nearest teammate must exceed for a blip.
const isolated = 1000

// Calc writes each player's row.Positioning. Every 4 s of a round after its first 18 s, with more
// than two Ts alive and the round undecided, the T farthest from its nearest teammate gets a blip
// (see sample); the T with more than 3 blips, alone at the most, is the round's lurker.
func Calc(r *round.Round, out *row.Round) {
	sides := map[uint64]fact.Side{} // the players with a row so far: one who joins mid-round has none before
	for id, m := range r.OpenMembers {
		sides[id] = m.Side
	}
	last := 0 // the sampler's clock
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil {
			sides[e.Join.Steam64] = e.Join.Side
		}
		f, ok := e.Fact.(fact.Frame)
		if !ok || e.Dropped || e.Sample == nil || last+4*fact.TickRate >= f.Tick {
			continue
		}
		last = f.Tick
		if e.After.TAlive <= 2 || e.After.Phase == round.Decided || f.Tick <= 18*fact.TickRate+r.StartTick {
			continue
		}
		sample(out, sides, e.Roster.Members(*e.Sample, fact.SideT))
	}

	var lurker uint64
	blips, tied := 0, false
	for id, p := range out.Players { // a tie is rejected below, so the order doesn't matter
		if sides[id] == fact.SideT && p.LurkerBlips > blips {
			blips, lurker = p.LurkerBlips, id
		}
	}
	for id, p := range out.Players {
		if sides[id] == fact.SideT && p.LurkerBlips == blips && id != lurker {
			tied = true
		}
	}
	if !tied && blips > 3 {
		out.Players[lurker].LurkRounds = 1
	}
}

// sample is one tick of the sampler over the Ts of the round's roster that have rows (D38): the living
// T whose nearest living teammate is farthest away gets a blip, if that is over isolated units and
// no other T is as far. A close moment costs only that sample.
func sample(out *row.Round, sides map[uint64]fact.Side, members []fact.Player) {
	var ts []fact.Player
	for _, t := range members {
		if _, ok := sides[t.Steam64]; ok && out.Players[t.Steam64] != nil && t.Alive {
			ts = append(ts, t)
		}
	}
	var lurker uint64
	far, tied := 0.0, false
	for _, t := range ts {
		near := math.Inf(1)
		for _, mate := range ts {
			if t.Steam64 != mate.Steam64 {
				near = min(near, distance(t.Pos, mate.Pos))
			}
		}
		switch {
		case near > far:
			far, lurker, tied = near, t.Steam64, false
		case near == far:
			tied = true
		}
	}
	if far > isolated && !tied {
		out.Players[lurker].LurkerBlips++
	}
}

// distance is r3.Vector.Distance (a.Sub(b).Norm()), with its arithmetic order.
func distance(a, b [3]float64) float64 {
	x, y, z := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(x*x + y*y + z*z)
}
