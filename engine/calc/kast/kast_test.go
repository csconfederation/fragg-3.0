package kast_test

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Hand-built rounds: players 1-2 T, 11-13 CT; 0 is the shared bot row; 99 isn't a member.
func player(steam uint64) *fact.Player {
	side := fact.SideT
	if steam > 10 {
		side = fact.SideCT
	}
	return &fact.Player{Steam64: steam, Side: side, IsBot: steam == 0}
}

func kill(killer, victim uint64, x round.Context) round.Event {
	w := 7
	return round.Event{Event: match.Event{Fact: fact.Kill{Killer: player(killer), Victim: player(victim), Weapon: &w}}, Context: x}
}

func run(events ...round.Event) *row.Round {
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{0, 1, 2, 11, 12, 13} {
		ms[id] = match.Member{Steam64: id, IsBot: id == 0}
	}
	return calc.Run(&round.Round{OpenMembers: ms, Events: events, Commit: len(events)})
}

type got struct{ csc, eco float64 }

func TestKAST(t *testing.T) {
	enemy := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true}
	cases := []struct {
		name   string
		events func() []round.Event
		want   map[uint64]got
	}{
		{"nothing happens: everyone survives", func() []round.Event { return nil },
			map[uint64]got{0: {1, 0}, 1: {1, 1}, 11: {1, 1}}},
		{"a kill counts for the killer, the death does not", func() []round.Event { return []round.Event{kill(11, 1, enemy)} },
			map[uint64]got{11: {1, 1}, 1: {0, 0}, 12: {1, 1}}},
		{"an assist counts for the assister", func() []round.Event {
			e := kill(11, 1, round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, Assist: true})
			e.Fact = func() fact.Kill { k := e.Fact.(fact.Kill); k.Assister = player(12); return k }()
			return []round.Event{e, kill(13, 2, enemy), kill(1, 13, enemy)}
		}, map[uint64]got{12: {1, 1}, 13: {1, 1}, 1: {1, 1}, 2: {0, 0}}},
		{"a flash assist counts for the flasher (D35)", func() []round.Event {
			fa := enemy
			fa.FlashAssisted, fa.FlashAssister = true, 12
			return []round.Event{kill(1, 12, enemy), kill(11, 2, fa)}
		}, map[uint64]got{12: {1, 1}, 1: {1, 1}, 2: {0, 0}}},
		{"a traded death counts", func() []round.Event {
			x := enemy
			x.Avenged = []round.Death{{Steam: 1, Tick: 1}}
			return []round.Event{kill(11, 1, enemy), kill(2, 13, x)}
		}, map[uint64]got{1: {1, 1}, 13: {0, 0}, 2: {1, 1}}},
		{"a team kill counts for nobody; the victim died", func() []round.Event {
			return []round.Event{kill(12, 11, round.Context{Expected: true, Killer: true, Victim: true})}
		}, map[uint64]got{12: {1, 1}, 11: {0, 0}}},
		{"a kill of a non-member counts for nobody (D31)", func() []round.Event {
			k := kill(1, 99, round.Context{Expected: true})
			return []round.Event{kill(11, 1, enemy), k}
		}, map[uint64]got{1: {0, 0}, 11: {1, 1}}},
		{"D44: a player who disconnects alive did not survive", func() []round.Event {
			return []round.Event{{Event: match.Event{Fact: fact.Disconnect{Steam64: 1}, Live: true}}}
		}, map[uint64]got{1: {0, 0}, 2: {1, 1}}},
		{"D44: a kill before the disconnect still counts", func() []round.Event {
			return []round.Event{kill(1, 11, enemy), {Event: match.Event{Fact: fact.Disconnect{Steam64: 1}, Live: true}}}
		}, map[uint64]got{1: {1, 1}}},
		{"bots share a row: one bot death kills its KAST", func() []round.Event {
			return []round.Event{kill(11, 0, enemy)}
		}, map[uint64]got{0: {0, 0}, 11: {1, 1}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := run(c.events()...)
			for id, w := range c.want {
				if g := (got{out.Players[id].KastRounds, out.Players[id].EcoKastRounds}); g != w {
					t.Errorf("player %d: csc, eco = %v, want %v", id, g, w)
				}
			}
		})
	}
}

// D31: a kill of a non-member is nobody's, in CSC or eco.
func TestEcoGate(t *testing.T) {
	out := run(kill(1, 99, round.Context{Expected: true}), kill(11, 2, round.Context{Expected: true, Victim: true}))
	if p := out.Players[1]; p.KastRounds != 1 || p.EcoKastRounds != 1 {
		t.Errorf("killer survived either way: %+v", p.KAST)
	}
	out = run(kill(1, 99, round.Context{Expected: true}), kill(11, 1, round.Context{Expected: true, Killer: true, Victim: true, Enemy: true}))
	if p := out.Players[1]; p.KastRounds != 0 || p.EcoKastRounds != 0 {
		t.Errorf("killer died: no KAST either way, got %+v", p.KAST)
	}
}

// A human who joins mid-round has an eco row from the join: an earlier kill was a scratch row's.
func TestLateJoiner(t *testing.T) {
	x := round.Context{Expected: true}
	join := round.Event{Event: match.Event{Fact: fact.TeamChange{}, Join: &match.Member{Steam64: 99}}}
	died := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true}
	out := run(kill(99, 1, x), join, kill(11, 99, died))
	if got := out.Players[99].EcoKastRounds; got != 0 {
		t.Errorf("eco KAST %v, want 0: the kill came before the join and the joiner died", got)
	}
}
