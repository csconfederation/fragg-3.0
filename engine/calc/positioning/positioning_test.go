package positioning

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// T 1 stands alone at the origin; 2, 3 and 4 stand together 1,000 units away, within 500 of each other.
func players(alive ...bool) []fact.Player {
	pos := [][3]float64{{0, 0, 0}, {1000, 0, 0}, {1000, 100, 0}, {1000, 200, 0}}
	var ps []fact.Player
	for i, p := range pos {
		ps = append(ps, fact.Player{Steam64: uint64(i + 1), Alive: alive[i], Pos: p, Listed: true})
	}
	return ps
}

func frame(tick int, ps []fact.Player, after round.State) round.Event {
	return round.Event{
		Event:   match.Event{Fact: fact.Frame{Tick: tick}, Live: true, Sample: &fact.Snapshot{Tick: tick, Players: ps}, Roster: match.Roster{T: []int{0, 1, 2, 3}}},
		Context: round.Context{After: after}}
}

func run(events ...round.Event) *row.Round {
	r := &round.Round{StartTick: 100, OpenMembers: map[uint64]match.Member{}, Events: events, Commit: len(events)}
	out := &row.Round{Players: map[uint64]*row.Player{}}
	for id := uint64(1); id <= 4; id++ {
		r.OpenMembers[id] = match.Member{Steam64: id, Side: fact.SideT}
		out.Players[id] = &row.Player{}
	}
	Calc(r, out)
	return out
}

func TestLurker(t *testing.T) {
	all, ok := players(true, true, true, true), round.State{TAlive: 4}
	// Frames 300 ticks apart: the first two are inside the first 18 s (1,252 ticks) and don't count.
	var five []round.Event
	for _, tick := range []int{500, 800, 1300, 1600, 1900, 2200, 2500} {
		five = append(five, frame(tick, all, ok))
	}
	out := run(five...)
	if p := out.Players[1]; p.LurkerBlips != 5 || p.LurkRounds != 1 || p.DistanceToTeammates <= 0 {
		t.Errorf("the lone T: %+v", p)
	}
	if p := out.Players[2]; p.LurkerBlips != 0 || p.LurkRounds != 0 || p.DistanceToTeammates >= 0 {
		t.Errorf("a T within 500 of another: %+v", p)
	}

	// Frames closer than 4 s apart are skipped; so are frames with two Ts alive or after the round is decided.
	var skipped []round.Event
	for _, tick := range []int{1300, 1400, 1500, 1556} { // only the first is sampled, the others are within 256 ticks of it
		skipped = append(skipped, frame(tick, all, ok))
	}
	skipped = append(skipped, frame(2000, all, round.State{TAlive: 2}), frame(2300, all, round.State{TAlive: 4, Phase: round.Decided}))
	if p := run(skipped...).Players[1]; p.LurkerBlips != 1 || p.LurkRounds != 0 {
		t.Errorf("skipped frames: %+v", p)
	}

	// Two Ts alone and equally far from each other get a blip in turn, the first in the roster order
	// winning the tie: 4 blips apiece, so no lurker.
	var tie []round.Event
	for i, tick := range []int{1300, 1600, 1900, 2200, 2500, 2800, 3100, 3400} {
		e := frame(tick, players(true, true, false, false), ok)
		e.Sample.Players[1].Pos = [3]float64{3000, 0, 0}
		if i%2 == 1 {
			e.Roster = match.Roster{T: []int{1, 0, 2, 3}}
		}
		tie = append(tie, e)
	}
	out = run(tie...)
	if a, b := out.Players[1], out.Players[2]; a.LurkerBlips != 4 || b.LurkerBlips != 4 || a.LurkRounds+b.LurkRounds != 0 {
		t.Errorf("a tie: %+v %+v", a.Positioning, b.Positioning)
	}
}
