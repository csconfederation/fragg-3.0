package positioning

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// T 1 stands alone at the origin; 2, 3 and 4 stand together 1,100 units away, within 200 of each other.
func players(alive ...bool) []fact.Player {
	pos := [][3]float64{{0, 0, 0}, {1100, 0, 0}, {1100, 100, 0}, {1100, 200, 0}}
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
	if p := out.Players[1]; p.LurkerBlips != 5 || p.LurkRounds != 1 {
		t.Errorf("the lone T: %+v", p)
	}
	if p := out.Players[2]; p.LurkerBlips != 0 || p.LurkRounds != 0 {
		t.Errorf("a T near another: %+v", p)
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

	// The lone T joins after two frames: no row before, so no blip before.
	join := round.Event{Event: match.Event{Fact: fact.TeamChange{}, Live: true, Join: &match.Member{Steam64: 1, Side: fact.SideT}}}
	late := []round.Event{frame(1300, all, ok), frame(1600, all, ok), join, frame(1900, all, ok), frame(2200, all, ok), frame(2500, all, ok)}
	r := &round.Round{StartTick: 100, OpenMembers: map[uint64]match.Member{}, Events: late, Commit: len(late)}
	lateOut := &row.Round{Players: map[uint64]*row.Player{}}
	for id := uint64(1); id <= 4; id++ {
		lateOut.Players[id] = &row.Player{}
		if id != 1 {
			r.OpenMembers[id] = match.Member{Steam64: id, Side: fact.SideT}
		}
	}
	Calc(r, lateOut)
	if p := lateOut.Players[1]; p.LurkerBlips != 3 {
		t.Errorf("a T who joined after two frames: %+v", p.Positioning)
	}

	// D38: two Ts alone and equally far from each other tie in every sample: nobody gets a blip.
	var tie []round.Event
	for _, tick := range []int{1300, 1600, 1900, 2200, 2500} {
		e := frame(tick, players(true, true, false, false), ok)
		e.Sample.Players[1].Pos = [3]float64{3000, 0, 0}
		tie = append(tie, e)
	}
	out = run(tie...)
	if a, b := out.Players[1], out.Players[2]; a.LurkerBlips != 0 || b.LurkerBlips != 0 {
		t.Errorf("a tie: %+v %+v", a.Positioning, b.Positioning)
	}
}

// D38: the farthest nearest-teammate distance needs to be over 1,000 units, and a close moment costs
// only its own sample: the lone T is back to a blip the next sample (today's rule strikes it out for the round).
func TestIsolationThreshold(t *testing.T) {
	all, ok := players(true, true, true, true), round.State{TAlive: 4}
	near := players(true, true, true, true)
	near[0].Pos = [3]float64{1100 - 999, 0, 0} // 999 from T 2: not isolated
	out := run(frame(1300, near, ok), frame(1600, all, ok), frame(1900, all, ok), frame(2200, all, ok), frame(2500, all, ok))
	if p := out.Players[1]; p.LurkerBlips != 4 || p.LurkRounds != 1 {
		t.Errorf("lone T after a close moment: %+v", p.Positioning)
	}
	atLimit := players(true, true, true, true)
	atLimit[0].Pos = [3]float64{1100 - 1000, 0, 0} // exactly 1,000 from T 2: not over
	if p := run(frame(1300, atLimit, ok)).Players[1]; p.LurkerBlips != 0 {
		t.Errorf("1,000 units gets a blip: %+v", p.Positioning)
	}
	// Dead mates don't count as near.
	if p := run(frame(1300, players(true, false, true, true), ok)).Players[1]; p.LurkerBlips != 1 {
		t.Errorf("with a dead mate: %+v", p.Positioning)
	}
}
