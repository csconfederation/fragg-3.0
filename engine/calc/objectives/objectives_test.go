package objectives_test

import (
	"math"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc/objectives"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Players 1-2 T (clan A, team state 2), 11-12 CT (clan B, team state 3); 0 is the shared bot row, a T.
func side(id uint64) fact.Side {
	if id > 10 {
		return fact.SideCT
	}
	return fact.SideT
}

func kill(killer, victim uint64, killerEquip, victimEquip int, x round.Context) round.Event {
	w := 7
	k := &fact.Player{Steam64: killer, Side: side(killer), IsBot: killer == 0, EquipValue: killerEquip}
	v := &fact.Player{Steam64: victim, Side: side(victim), IsBot: victim == 0, EquipValue: victimEquip}
	return round.Event{Event: match.Event{Fact: fact.Kill{Killer: k, Victim: v, Weapon: &w}}, Context: x}
}

func st(phase round.Phase, opening bool) round.State {
	return round.State{Phase: phase, Opening: opening}
}

func ctx(before round.State, f round.Context) round.Context {
	f.Before = before
	return f
}

var enemy = round.Context{Expected: true, Killer: true, Victim: true, Enemy: true}

func winEvent(w fact.Side) round.Event {
	clan := map[fact.Side]string{fact.SideT: "A", fact.SideCT: "B"}[w]
	return round.Event{Event: match.Event{Fact: fact.RoundEnd{Winner: w}, Win: true, Winner: clan}}
}

// round is a round with these events and the Win at the end (index len(events)), or none.
func build(winner fact.Side, events ...round.Event) *round.Round {
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{0, 1, 2, 11, 12} {
		clan, team := "A", 2
		if id > 10 {
			clan, team = "B", 3
		}
		ms[id] = match.Member{Steam64: id, IsBot: id == 0, Clan: clan, TeamID: team, Side: side(id)}
	}
	r := &round.Round{OpenMembers: ms, TClan: "A", CTClan: "B", Events: events, Commit: len(events)}
	if winner != 0 {
		r.Events = append(r.Events, winEvent(winner))
		r.Commit++
		r.Wins = []int{len(r.Events)}
	}
	return r
}

func seed(r *round.Round) *row.Round {
	out := &row.Round{Players: map[uint64]*row.Player{}, Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	return out
}

func run(r *round.Round) *row.Round {
	out := seed(r)
	objectives.Calc(r, out)
	return out
}

// The kill value table (Q7, Q19) and its float order.
func TestKillValue(t *testing.T) {
	cases := []struct {
		name   string
		killer uint64
		x      round.Context
		want   float64
	}{
		{"T pre-plant", 1, ctx(st(round.PrePlant, false), enemy), 1.2},
		{"CT pre-plant", 11, ctx(st(round.PrePlant, false), enemy), 1},
		{"T post-plant", 1, ctx(st(round.PostPlant, false), enemy), 1},
		{"CT post-plant", 11, ctx(st(round.PostPlant, false), enemy), 1.2},
		{"T opening entry", 1, ctx(st(round.PrePlant, true), enemy), mul(1.2, add(1.0, 0.8))},
		{"T opening post-plant", 1, ctx(st(round.PostPlant, true), enemy), mul(1, add(1.0, 0.3))},
		{"CT opening", 11, ctx(st(round.PrePlant, true), enemy), mul(1, add(1.0, 0.5))},
		{"trade", 1, ctx(st(round.PrePlant, false), withAvenged(enemy)), mul(1.2, add(1.0, 0.3))},
		{"an opening kill is no trade", 1, ctx(st(round.PrePlant, true), withAvenged(enemy)), mul(1.2, add(1.0, 0.8))},
		{"team kill: no points", 1, ctx(st(round.PrePlant, false), round.Context{Expected: true, Killer: true, Victim: true}), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := run(build(0, kill(c.killer, 12, 1000, 1000, c.x)))
			if got := out.Players[c.killer].KillPoints; got != c.want {
				t.Errorf("%v, want %v", got, c.want)
			}
		})
	}
}

// Constant expressions are exact in Go; the calculator's are float64 operations, one by one.
func add(a, b float64) float64 { return a + b }
func sub(a, b float64) float64 { return a - b }
func mul(a, b float64) float64 { return a * b }

func withAvenged(x round.Context) round.Context {
	x.Avenged = []round.Death{{Steam: 2, Tick: 1}}
	return x
}

// After the Win the value follows the winner; before any Win it is 1 (Q20). The Win is in effect
// for the event it is placed before.
func TestAfterTheWin(t *testing.T) {
	decided := ctx(st(round.Decided, false), enemy)
	for _, c := range []struct {
		winner fact.Side
		killer uint64
		tMoney bool
		want   float64
	}{
		{fact.SideT, 1, false, 0.8}, {fact.SideT, 11, false, 0.6},
		{fact.SideCT, 1, false, 0.5}, {fact.SideCT, 11, true, 0.6}, {fact.SideCT, 11, false, 0.8},
	} {
		x := decided
		x.Before.TMoney = c.tMoney
		r := build(0, winEvent(c.winner), kill(c.killer, 12, 1000, 1000, x))
		r.Wins = []int{1} // placed right after its RoundEnd
		if got := run(r).Players[c.killer].KillPoints; got != c.want {
			t.Errorf("winner %v killer %d tMoney %v: %v, want %v", c.winner, c.killer, c.tMoney, got, c.want)
		}
		r.Wins = []int{2} // placed after the kill: no Win in effect yet
		if got := run(r).Players[c.killer].KillPoints; got != 1 {
			t.Errorf("before the Win: %v, want 1", got)
		}
	}
}

// ecoMod: the ratio's thresholds, an unarmed killer (+Inf) and nobody armed (NaN).
func TestEcoMod(t *testing.T) {
	cases := []struct {
		killer, victim int
		mod            float64
	}{
		{1000, 4001, add(1, 0.25)}, {1000, 4000, add(1, 0.14)}, {1000, 2001, add(1, 0.14)}, {1000, 2000, 1},
		{1000, 500, 1}, {1000, 499, sub(1, 0.14)}, {1000, 250, sub(1, 0.14)}, {1000, 249, sub(1, 0.25)},
		{0, 1000, add(1, 0.25)}, {0, 0, 1}, {1000, 0, sub(1, 0.25)},
	}
	for _, c := range cases {
		out := run(build(0, kill(1, 12, c.killer, c.victim, ctx(st(round.PrePlant, false), enemy))))
		if got, want := out.Players[1].KillPoints, mul(1.2, c.mod); got != want {
			t.Errorf("equipment %d against %d: %v, want %v", c.killer, c.victim, got, want)
		}
	}
}

// An assist takes 0.15 off an enemy kill and gives it to the assister; a flash assist adds 0.2 to
// the multiplier. A team kill's assist does neither.
func TestAssist(t *testing.T) {
	x := enemy
	x.Assist = true
	e := kill(1, 11, 1000, 1000, ctx(st(round.PrePlant, false), x))
	f := e.Fact.(fact.Kill)
	f.Assister = &fact.Player{Steam64: 2, Side: fact.SideT}
	e.Fact = f
	out := run(build(0, e))
	if got, want := out.Players[1].KillPoints, mul(sub(1.2, 0.15), 1.0); got != want {
		t.Errorf("killer %v, want %v", got, want)
	}
	if got := out.Players[2].ImpactPoints; got != 0.15 {
		t.Errorf("assister %v, want 0.15", got)
	}
	e.Context.FlashAssisted = true
	if got, want := run(build(0, e)).Players[1].KillPoints, mul(sub(1.2, 0.15), add(1.0, 0.2)); got != want {
		t.Errorf("flash assisted %v, want %v", got, want)
	}
	f.AssistedFlash = true
	e.Fact = f
	e.Context.FlashAssisted = false
	if got, want := run(build(0, e)).Players[1].KillPoints, mul(sub(1.2, 0.15), add(1.0, 0.2)); got != want {
		t.Errorf("marked flash assist %v, want %v", got, want)
	}
	team := ctx(st(round.PrePlant, false), round.Context{Expected: true, Killer: true, Victim: true, Assist: true})
	e = kill(1, 2, 1000, 1000, team)
	f = e.Fact.(fact.Kill)
	f.Assister = &fact.Player{Steam64: 11, Side: fact.SideCT}
	e.Fact = f
	if got := run(build(0, e)).Players[11].ImpactPoints; got != 0 {
		t.Errorf("a team kill's assister %v, want 0", got)
	}
}

func bomb(f fact.Fact, phase round.Phase) round.Event {
	return round.Event{Event: match.Event{Fact: f}, Context: round.Context{Before: round.State{Phase: phase}}}
}

func TestBomb(t *testing.T) {
	p := func(id uint64) *fact.Player { return &fact.Player{Steam64: id, Side: side(id)} }
	out := run(build(0,
		bomb(fact.BombPlanted{Snapshot: fact.Snapshot{Tick: 500}, Player: p(1)}, round.PrePlant),
		bomb(fact.BombExplode{}, round.PostPlant),
	))
	if out.Planter != 1 || out.BombStartTick != 500 || !out.EndDueToBombEvent || out.Defuser != 0 || out.Players[1].ImpactPoints != 0.5 {
		t.Errorf("explosion: %+v, planter points %v", out.Bomb, out.Players[1].ImpactPoints)
	}
	out = run(build(0,
		bomb(fact.BombPlanted{Snapshot: fact.Snapshot{Tick: 500}, Player: p(1)}, round.PrePlant),
		bomb(fact.BombDefused{Player: p(11)}, round.PostPlant),
		bomb(fact.BombPlanted{Snapshot: fact.Snapshot{Tick: 900}, Player: p(2)}, round.Decided), // decided: ignored
		bomb(fact.BombExplode{}, round.Decided),
	))
	if out.Planter != 1 || out.BombStartTick != 500 || out.Defuser != 11 || out.Players[11].ImpactPoints != 0.5 || out.Players[1].ImpactPoints != 0 {
		t.Errorf("defuse: %+v, points %v %v", out.Bomb, out.Players[11].ImpactPoints, out.Players[1].ImpactPoints)
	}
	// A bot planter is Steam64 0: no points on the explosion.
	out = run(build(0, bomb(fact.BombPlanted{Player: p(0)}, round.PrePlant), bomb(fact.BombExplode{}, round.PostPlant)))
	if out.Players[0].ImpactPoints != 0 || !out.EndDueToBombEvent {
		t.Errorf("bot planter: %+v %v", out.Bomb, out.Players[0].ImpactPoints)
	}
}

func TestExitFrags(t *testing.T) {
	x := ctx(st(round.Decided, false), enemy)
	out := run(build(0, kill(1, 11, 1000, 1000, x), kill(1, 12, 1000, 1000, ctx(st(round.PrePlant, false), enemy)), kill(0, 11, 1000, 1000, x)))
	if out.Players[1].ExitFrags != 1 || out.Players[0].ExitFrags != 0 {
		t.Errorf("exit frags %d, bot %d: want 1, 0 (eco has no bot row)", out.Players[1].ExitFrags, out.Players[0].ExitFrags)
	}
}

// finish runs Calc and Finish with the combat rows set.
func finish(r *round.Round, combat map[uint64]row.Combat) *row.Round {
	out := seed(r)
	for id, c := range combat {
		out.Players[id].Combat = c
	}
	objectives.Calc(r, out)
	objectives.Finish(r, out)
	return out
}

// Impact is the sum in its order: bonuses, clutch, kill points, damage/250, multi-kill. Win points, RF and
// RA follow the player's team state; mip is shared by a tie; the round win share splits the winners' damage.
func TestFinish(t *testing.T) {
	r := build(fact.SideT, kill(1, 11, 1000, 1000, ctx(st(round.PrePlant, false), enemy)), kill(1, 12, 1000, 1000, ctx(st(round.PrePlant, false), enemy)))
	r.End.TClutch = round.Clutch{Size: 2, Steam: 1}
	out := finish(r, map[uint64]row.Combat{1: {Damage: 130, Kills: 2}, 2: {Damage: 20}, 11: {Damage: 100}})
	k := mul(1.2, 1.0)    // each kill: pre-plant T
	want := add(0.0, 0.6) // the clutch bonus, 1v2
	want = add(want, k)   // kill points, kill by kill
	want = add(want, k)
	want = add(want, float64(130)/float64(250)) // damage
	want = add(want, 0.3)                       // two kills
	if got := out.Players[1].ImpactPoints; got != want {
		t.Errorf("impact %v, want %v", got, want)
	}
	p1, p2, p11 := out.Players[1], out.Players[2], out.Players[11]
	if p1.WinPoints != p1.ImpactPoints || p1.RF != 1 || p1.RA != 0 || p11.RF != 0 || p11.RA != 1 || p11.WinPoints != 0 {
		t.Errorf("win points and rounds: %+v / %+v", p1.Objectives, p11.Objectives)
	}
	if p1.Mip != 1 || p2.Mip != 0 {
		t.Errorf("mip %v %v", p1.Mip, p2.Mip)
	}
	if out.WinTeamDmg != 130+20 {
		t.Errorf("winning team damage %d", out.WinTeamDmg)
	}
	if got, want := p1.Rws, 100*(130.0/150.0); got != want {
		t.Errorf("rws %v, want %v", got, want)
	}
	tt, ct := out.Teams[fact.SideT], out.Teams[fact.SideCT]
	if tt.WinPoints != p1.WinPoints+p2.WinPoints || tt.TWinPoints != tt.WinPoints || tt.CtWinPoints != 0 ||
		tt.ImpactPoints != p1.ImpactPoints+p2.ImpactPoints || ct.WinPoints != 0 || ct.CtImpactPoints != ct.ImpactPoints {
		t.Errorf("teams %+v %+v", tt.TeamObjectives, ct.TeamObjectives)
	}
}

// Everyone with nothing shares mip (Q17); a zero-damage win has no round win share, but the
// planter keeps the bomb's 30 (D46).
func TestFinishEdges(t *testing.T) {
	r := build(fact.SideT)
	out := finish(r, nil)
	for id, p := range out.Players {
		if p.Mip != 1.0/5 {
			t.Errorf("player %d mip %v, want 0.2", id, p.Mip)
		}
	}
	r = build(fact.SideT,
		bomb(fact.BombPlanted{Player: &fact.Player{Steam64: 1, Side: fact.SideT}}, round.PrePlant),
		bomb(fact.BombExplode{}, round.PostPlant))
	out = finish(r, map[uint64]row.Combat{1: {Damage: 0}})
	if p := out.Players[1]; p.Rws != 30 {
		t.Errorf("zero-damage win, planter rws %v, want 30", p.Rws)
	}
	if p := out.Players[2]; p.Rws != 0 || math.IsNaN(p.Rws) {
		t.Errorf("zero-damage win, teammate rws %v, want 0", p.Rws)
	}
	out = finish(r, map[uint64]row.Combat{1: {Damage: 50}, 2: {Damage: 50}})
	if got, want := out.Players[1].Rws, 70*(50.0/100.0)+30; got != want {
		t.Errorf("planter rws %v, want %v", got, want)
	}
	if got, want := out.Players[2].Rws, 70*(50.0/100.0); got != want {
		t.Errorf("rws %v, want %v", got, want)
	}
	// Bots share row 0: a bot planter's row gets the +30, and more than five kills the five-kill bonus.
	r = build(fact.SideT, bomb(fact.BombPlanted{Player: &fact.Player{Steam64: 0, IsBot: true, Side: fact.SideT}}, round.PrePlant), bomb(fact.BombExplode{}, round.PostPlant))
	out = finish(r, map[uint64]row.Combat{0: {Damage: 10, Kills: 7}})
	if got, want := out.Players[0].Rws, 70*(10.0/10.0)+30; got != want {
		t.Errorf("bot planter rws %v, want %v", got, want)
	}
	if got, want := out.Players[0].ImpactPoints, 0.0+float64(10)/float64(250)+2; got != want {
		t.Errorf("bot impact %v, want %v", got, want)
	}
}

// A defuser or planter who joins later has no row at the event: no credit then.
func TestBombBeforeJoin(t *testing.T) {
	p := &fact.Player{Steam64: 5, Side: fact.SideCT}
	join := round.Event{Event: match.Event{Fact: fact.TeamChange{}, Join: &match.Member{Steam64: 5, Clan: "B"}}}
	r := build(0, bomb(fact.BombDefused{Player: p}, round.PostPlant), join)
	out := seed(r)
	out.Players[5] = &row.Player{} // calc.Run seeds the joiner's row
	objectives.Calc(r, out)
	if out.Players[5].ImpactPoints != 0 || out.Defuser != 5 {
		t.Errorf("defuse before the join: %v, defuser %d", out.Players[5].ImpactPoints, out.Defuser)
	}
	r = build(0, join, bomb(fact.BombDefused{Player: p}, round.PostPlant))
	out = seed(r)
	out.Players[5] = &row.Player{}
	objectives.Calc(r, out)
	if out.Players[5].ImpactPoints != 0.5 {
		t.Errorf("defuse after the join: %v", out.Players[5].ImpactPoints)
	}
}
