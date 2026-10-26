package round

import (
	"reflect"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// Players 1–2 are T (team state 2), 11–13 CT (team state 3).
func player(steam uint64) *fact.Player {
	side, team := fact.SideT, 2
	if steam > 10 {
		side, team = fact.SideCT, 3
	}
	return &fact.Player{Steam64: steam, Side: side, TeamStateID: &team, Alive: true}
}

func killAt(tick int, killer, victim *fact.Player) fact.Kill {
	ak := 7
	return fact.Kill{Snapshot: fact.Snapshot{Tick: tick}, Killer: killer, Victim: victim, Weapon: &ak}
}

func newRound() *Tracker {
	t := New()
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{1, 2, 11, 12, 13} {
		ms[id] = match.Member{Steam64: id}
	}
	t.Open(&match.Round{Num: 1, Members: ms, InitT: 2, InitCT: 3})
	return t
}

// The trade contract (D22, design note "Trades").
func TestTrades(t *testing.T) {
	cases := []struct {
		name  string
		kills []fact.Kill
		want  []Death // the last kill's avenged deaths
	}{
		{"320 ticks is in the window",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1320, player(12), player(1))},
			[]Death{{11, 1000}}},
		{"321 ticks is not",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1321, player(12), player(1))}, nil},
		{"same tick, in event order",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1000, player(12), player(1))},
			[]Death{{11, 1000}}},
		{"every avenged death, oldest first",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1100, player(1), player(12)),
				killAt(1200, player(13), player(1))},
			[]Death{{11, 1000}, {12, 1100}}},
		{"team kill trades nothing",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1100, player(2), player(1))}, nil},
		{"suicide trades nothing",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1100, player(1), player(1))}, nil},
		{"world death trades nothing",
			[]fact.Kill{killAt(1000, player(1), player(11)), killAt(1100, nil, player(1))}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, m := newRound(), match.New()
			var c Context
			for _, f := range tc.kills {
				c = Context{}
				if err := tr.kill(m, f, &c); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(c.Avenged, tc.want) {
				t.Errorf("avenged %v, want %v", c.Avenged, tc.want)
			}
		})
	}
}

// One kill avenging two deaths marks both, oldest first; a death is traded once, though bots share
// Steam64 0.
func TestTradedOnce(t *testing.T) {
	tr, m := newRound(), match.New()
	bot := player(1)
	bot.Steam64 = 0
	tr.members[0] = true
	for _, f := range []fact.Kill{killAt(1000, bot, player(11)), killAt(1100, bot, player(12))} {
		if err := tr.kill(m, f, &Context{}); err != nil {
			t.Fatal(err)
		}
	}
	// 12 is dead, but the round still counts kills by a member who died earlier in the tick.
	var c Context
	if err := tr.kill(m, killAt(1200, player(12), bot), &c); err != nil {
		t.Fatal(err)
	}
	if want := []Death{{11, 1000}, {12, 1100}}; !reflect.DeepEqual(c.Avenged, want) {
		t.Fatalf("avenged %v, want %v", c.Avenged, want)
	}
	c = Context{}
	if err := tr.kill(m, killAt(1250, player(11), bot), &c); err != nil {
		t.Fatal(err)
	}
	if c.Avenged != nil {
		t.Errorf("traded again: %v", c.Avenged)
	}
}
