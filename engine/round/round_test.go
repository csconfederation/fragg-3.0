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

// Bots share Steam64 0 but trade only their own kills (review52).
func TestBotsTradeTheirOwnKills(t *testing.T) {
	tr, m := newRound(), match.New()
	tr.members[0] = true
	a, b := player(1), player(2)
	a.Steam64, a.IsBot, a.UserID = 0, true, 5
	b.Steam64, b.IsBot, b.UserID = 0, true, 6
	for _, f := range []fact.Kill{killAt(1000, a, player(11)), killAt(1100, b, player(12))} {
		if err := tr.kill(m, f, &Context{}); err != nil {
			t.Fatal(err)
		}
	}
	var c Context
	if err := tr.kill(m, killAt(1200, player(13), a), &c); err != nil {
		t.Fatal(err)
	}
	if want := []Death{{11, 1000}}; !reflect.DeepEqual(c.Avenged, want) {
		t.Errorf("avenged %v, want %v", c.Avenged, want)
	}
}

func flash(tick int, by, on *fact.Player, seconds float64) fact.Flashed {
	return fact.Flashed{Tick: tick, Attacker: by, Player: on, Duration: seconds}
}

// CSC's flash filter, its latest-flash state and its flash assists (#591).
func TestFlashes(t *testing.T) {
	tr := newRound()
	blinded := player(11)
	cases := []struct {
		name string
		f    fact.Flashed
		want bool
	}{
		{"enemy over 1 s", flash(1000, player(1), blinded, 1.5), true},
		{"1 s exactly", flash(1000, player(1), blinded, 1), false},
		{"teammate", flash(1000, player(12), blinded, 2), false},
		{"dead player", flash(1000, player(1), &fact.Player{Steam64: 11, Side: fact.SideCT}, 2), false},
	}
	for _, tc := range cases {
		var c Context
		tr.flashed(tc.f, &c)
		if c.Flash != tc.want {
			t.Errorf("%s: counted %v", tc.name, c.Flash)
		}
	}
	if want := (Blind{1, 1096}); tr.blind[11] != want {
		t.Fatalf("blind %v, want %v", tr.blind[11], want)
	}
	var c Context
	tr.flashed(flash(1010, player(2), blinded, 1.2), &c) // wears off sooner: the state stays
	if !c.Flash || c.Blind != (Blind{1, 1096}) {
		t.Errorf("shorter flash: counted %v, blind %v", c.Flash, c.Blind)
	}
	already := player(11)
	already.FlashDuration = 3
	c = Context{}
	if tr.flashed(flash(1020, player(1), already, 1.5), &c); c.Flash {
		t.Error("counted a flash on a player already blind for longer")
	}

	// A plain assist while the victim is blind is a flash assist for the flasher.
	m := match.New()
	k := killAt(1090, player(2), player(11))
	k.Assister = player(1)
	c = Context{}
	if err := tr.kill(m, k, &c); err != nil {
		t.Fatal(err)
	}
	if !c.FlashAssisted || c.FlashAssister != 1 {
		t.Errorf("flash assist %v by %d", c.FlashAssisted, c.FlashAssister)
	}
}

// A disconnect recounts the alive members; a player relisted alive mid-round without joining the
// round (a reconnect with no player-info match, S19 Combine mid7214) isn't one.
func TestDisconnectRecountsMembersOnly(t *testing.T) {
	tr, m := newRound(), match.New()
	var s fact.Snapshot
	for _, id := range []uint64{1, 2, 3, 11, 12, 13} { // 3 is the relisted ghost
		p := *player(id)
		p.Listed = true
		p.Alive = id != 13 // 13 is the one disconnecting
		s.Players = append(s.Players, p)
	}
	var c Context
	if err := tr.apply(m, fact.Disconnect{Snapshot: s, Steam64: 13}, &c); err != nil {
		t.Fatal(err)
	}
	if tr.s.TAlive != 2 || tr.s.CTAlive != 2 {
		t.Errorf("alive %dv%d, want 2v2", tr.s.TAlive, tr.s.CTAlive)
	}
}

// Bots share Steam64 0, so deaths go by actor: one bot's death leaves another bot alive at a
// disconnect recount (keyed by Steam64, every bot counted dead, and the counts went negative).
func TestBotDeathsByActor(t *testing.T) {
	tr, m := New(), match.New()
	ms := map[uint64]match.Member{1: {Steam64: 1}, 0: {Steam64: 0, IsBot: true}, 11: {Steam64: 11}, 12: {Steam64: 12}, 13: {Steam64: 13}}
	tr.Open(&match.Round{Num: 1, Members: ms, InitT: 3, InitCT: 3})
	bot := func(user int, alive bool) *fact.Player {
		p := player(0)
		p.IsBot, p.UserID, p.Alive = true, user, alive
		return p
	}
	var c Context
	if err := tr.kill(m, killAt(1000, player(11), bot(5, true)), &c); err != nil {
		t.Fatal(err)
	}
	var s fact.Snapshot
	for _, p := range []*fact.Player{player(1), bot(5, false), bot(6, true), player(11), player(12), player(13)} {
		p.Listed = true
		s.Players = append(s.Players, *p)
	}
	if err := tr.apply(m, fact.Disconnect{Snapshot: s, Steam64: 99}, &c); err != nil {
		t.Fatal(err)
	}
	if tr.s.TAlive != 2 {
		t.Errorf("T alive %d, want 2 (the human and the living bot)", tr.s.TAlive)
	}
}

// A flash counts only between round members (rows in every collector); the oracle dereferences
// nil for a player relisted without joining the round.
func TestFlashNeedsMembers(t *testing.T) {
	tr := newRound()
	for _, tc := range []struct {
		name    string
		by, on  uint64
		counted bool
	}{{"members", 1, 11, true}, {"non-member flasher", 3, 11, false}, {"non-member victim", 1, 14, false}} {
		var c Context
		tr.flashed(flash(1000, player(tc.by), player(tc.on), 3), &c)
		if c.Flash != tc.counted {
			t.Errorf("%s: flash counted %v, want %v", tc.name, c.Flash, tc.counted)
		}
	}
}
