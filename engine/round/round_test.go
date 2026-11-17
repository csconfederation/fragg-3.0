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

func newRound() *Deriver {
	d := NewDeriver(nil)
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{1, 2, 11, 12, 13} {
		ms[id] = match.Member{Steam64: id}
	}
	d.Open(&match.Round{Num: 1, Members: ms, InitT: 2, InitCT: 3}, fact.Snapshot{}, match.Roster{})
	return d
}

// rosterOf is the roster of s's listed players, as a lifecycle without reconnects takes it.
func rosterOf(s fact.Snapshot) match.Roster {
	var r match.Roster
	for i, p := range s.Players {
		if p.Listed && p.Side == fact.SideT {
			r.T = append(r.T, i)
		} else if p.Listed && p.Side == fact.SideCT {
			r.CT = append(r.CT, i)
		}
	}
	return r
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
			tr := newRound()
			var c Context
			for _, f := range tc.kills {
				c = Context{}
				if err := tr.kill(f, match.Roster{}, &c); err != nil {
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
	tr := newRound()
	bot := player(1)
	bot.Steam64 = 0
	tr.members[0] = match.Member{}
	for _, f := range []fact.Kill{killAt(1000, bot, player(11)), killAt(1100, bot, player(12))} {
		if err := tr.kill(f, match.Roster{}, &Context{}); err != nil {
			t.Fatal(err)
		}
	}
	// 12 is dead, but the round still counts kills by a member who died earlier in the tick.
	var c Context
	if err := tr.kill(killAt(1200, player(12), bot), match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	if want := []Death{{11, 1000}, {12, 1100}}; !reflect.DeepEqual(c.Avenged, want) {
		t.Fatalf("avenged %v, want %v", c.Avenged, want)
	}
	c = Context{}
	if err := tr.kill(killAt(1250, player(11), bot), match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Avenged != nil {
		t.Errorf("traded again: %v", c.Avenged)
	}
}

// Bots share Steam64 0 but trade only their own kills (review52).
func TestBotsTradeTheirOwnKills(t *testing.T) {
	tr := newRound()
	tr.members[0] = match.Member{}
	a, b := player(1), player(2)
	a.Steam64, a.IsBot, a.UserID = 0, true, 5
	b.Steam64, b.IsBot, b.UserID = 0, true, 6
	for _, f := range []fact.Kill{killAt(1000, a, player(11)), killAt(1100, b, player(12))} {
		if err := tr.kill(f, match.Roster{}, &Context{}); err != nil {
			t.Fatal(err)
		}
	}
	var c Context
	if err := tr.kill(killAt(1200, player(13), a), match.Roster{}, &c); err != nil {
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
	k := killAt(1090, player(2), player(11))
	k.Assister = player(1)
	c = Context{}
	if err := tr.kill(k, match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	if !c.FlashAssisted || c.FlashAssister != 1 {
		t.Errorf("flash assist %v by %d", c.FlashAssisted, c.FlashAssister)
	}

	// D34: the latest flasher who made the kill got no assist on it.
	k = killAt(1090, player(1), player(11))
	k.Assister = player(2)
	c = Context{}
	if err := tr.kill(k, match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	if !c.Assist || c.FlashAssisted || c.FlashAssister != 0 {
		t.Errorf("own kill: assist %v, flash assist %v by %d", c.Assist, c.FlashAssisted, c.FlashAssister)
	}

	// A bot's flash on another bot's kill is not told apart from its own: kept, not suppressed.
	bot := func(uid int) *fact.Player { p := player(1); p.Steam64, p.IsBot, p.UserID = 0, true, uid; return p }
	tr.members[0] = match.Member{IsBot: true}
	tr.blind[11] = Blind{0, 2000}
	k = killAt(1090, bot(5), player(11))
	k.Assister = bot(6)
	c = Context{}
	if err := tr.kill(k, match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	if !c.FlashAssisted {
		t.Error("a bot's flash assist on another bot's kill was dropped")
	}
}

// A disconnect recounts the alive members; a player relisted alive mid-round without joining the
// round (a reconnect with no player-info match, S19 Combine mid7214) isn't one.
func TestDisconnectRecountsMembersOnly(t *testing.T) {
	tr := newRound()
	var s fact.Snapshot
	for _, id := range []uint64{1, 2, 3, 11, 12, 13} { // 3 is the relisted ghost
		p := *player(id)
		p.Listed = true
		p.Alive = id != 13 // 13 is the one disconnecting
		s.Players = append(s.Players, p)
	}
	var c Context
	if err := tr.apply(match.Event{Fact: fact.Disconnect{Snapshot: s, Steam64: 13}, Roster: rosterOf(s)}, &c); err != nil {
		t.Fatal(err)
	}
	if tr.s.TAlive != 2 || tr.s.CTAlive != 2 {
		t.Errorf("alive %dv%d, want 2v2", tr.s.TAlive, tr.s.CTAlive)
	}
}

// Bots share Steam64 0, so deaths go by actor: one bot's death leaves another bot alive at a
// disconnect recount (keyed by Steam64, every bot counted dead, and the counts went negative).
func TestBotDeathsByActor(t *testing.T) {
	tr := NewDeriver(nil)
	ms := map[uint64]match.Member{1: {Steam64: 1}, 0: {Steam64: 0, IsBot: true}, 11: {Steam64: 11}, 12: {Steam64: 12}, 13: {Steam64: 13}}
	tr.Open(&match.Round{Num: 1, Members: ms, InitT: 3, InitCT: 3}, fact.Snapshot{}, match.Roster{})
	bot := func(user int, alive bool) *fact.Player {
		p := player(0)
		p.IsBot, p.UserID, p.Alive = true, user, alive
		return p
	}
	var c Context
	if err := tr.kill(killAt(1000, player(11), bot(5, true)), match.Roster{}, &c); err != nil {
		t.Fatal(err)
	}
	var s fact.Snapshot
	for _, p := range []*fact.Player{player(1), bot(5, false), bot(6, true), player(11), player(12), player(13)} {
		p.Listed = true
		s.Players = append(s.Players, *p)
	}
	if err := tr.apply(match.Event{Fact: fact.Disconnect{Snapshot: s, Steam64: 99}, Roster: rosterOf(s)}, &c); err != nil {
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

func TestWinAt(t *testing.T) {
	end := func(side fact.Side, clan string, tick int) Event {
		return Event{Event: match.Event{Fact: fact.RoundEnd{Snapshot: fact.Snapshot{Tick: tick}, Winner: side}, Win: true, Winner: clan}}
	}
	plain := Event{Event: match.Event{Fact: fact.RoundEnd{}}} // a RoundEnd without a Win
	r := &Round{Events: []Event{plain, end(fact.SideT, "a", 10), {}, end(fact.SideCT, "b", 20), {}}, Wins: []int{3, 5}}
	for i, want := range []struct {
		ok   bool
		side fact.Side
		tick int
	}{{false, 0, 0}, {false, 0, 0}, {false, 0, 0}, {true, fact.SideT, 10}, {true, fact.SideT, 10}, {true, fact.SideCT, 20}} {
		w, ok := r.WinAt(i)
		if ok != want.ok || w.Side != want.side || w.Tick != want.tick {
			t.Errorf("WinAt(%d) = %+v, %v; want %+v", i, w, ok, want)
		}
	}
}

// D33: a player who joins alive mid-round is alive in the counts, so a clutch starts at the right
// size and the last death of a side leaves the right count.
func TestJoinerCountsAlive(t *testing.T) {
	tr := newRound() // 2v3
	join := match.Member{Steam64: 3, Side: fact.SideT, TeamID: 2}
	if err := tr.Event(match.Event{Fact: fact.TeamChange{}, Live: true, Join: &join}); err != nil {
		t.Fatal(err)
	}
	if tr.s.TAlive != 3 || tr.s.CTAlive != 3 {
		t.Errorf("alive %dv%d after a T joiner, want 3v3", tr.s.TAlive, tr.s.CTAlive)
	}
	if !tr.isMember(3) {
		t.Error("the joiner is not a member")
	}
}
