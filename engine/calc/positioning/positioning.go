// Package positioning is the positioning family: the lurk sampler's blips and the round's lurker.
package positioning

import (
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc writes each player's row.Positioning. Every 4 s of a round after its first 18 s, with more
// than two Ts alive and the round undecided, the T whose distance sum to the others is smallest
// and positive gets a blip; the T with more than 3 blips, alone at the most, is the round's lurker.
func Calc(r *round.Round, out *row.Round) {
	sides := map[uint64]fact.Side{}
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
		sample(out, e.Roster.Members(*e.Sample, fact.SideT))
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

// sample is one tick of the sampler over the Ts of the round's roster that have rows.
func sample(out *row.Round, members []fact.Player) {
	var ts []fact.Player
	for _, t := range members {
		if out.Players[t.Steam64] != nil {
			ts = append(ts, t)
		}
	}
	for _, t := range ts {
		if !t.Alive {
			continue
		}
		for _, mate := range ts {
			if t.Steam64 == mate.Steam64 || !mate.Alive {
				continue
			}
			p := out.Players[t.Steam64]
			dist := int(distance(t.Pos, mate.Pos))
			if dist < 500 {
				p.DistanceToTeammates = -999999
			}
			p.DistanceToTeammates += dist
		}
	}
	var lurker uint64
	best := 999999
	for _, t := range ts {
		if t.Alive {
			if d := out.Players[t.Steam64].DistanceToTeammates; d < best && d > 0 {
				best, lurker = d, t.Steam64
			}
		}
	}
	if lurker != 0 {
		out.Players[lurker].LurkerBlips++
	}
}

// distance is r3.Vector.Distance (a.Sub(b).Norm()), with its arithmetic order.
func distance(a, b [3]float64) float64 {
	x, y, z := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(x*x + y*y + z*z)
}
