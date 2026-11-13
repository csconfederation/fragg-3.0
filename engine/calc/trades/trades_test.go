package trades_test

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Players 1-2 T, 11-13 CT; 0 is the shared bot row. mate is a listed, alive teammate at x.
type mate struct {
	id uint64
	x  float64
}

func side(id uint64) fact.Side {
	if id > 10 {
		return fact.SideCT
	}
	return fact.SideT
}

func kill(tick int, killer, victim uint64, mates []mate, x round.Context) round.Event {
	w := 7
	k, v := &fact.Player{Steam64: killer, Side: side(killer), IsBot: killer == 0}, &fact.Player{Steam64: victim, Side: side(victim)}
	s := fact.Snapshot{Tick: tick}
	for _, m := range mates {
		s.Players = append(s.Players, fact.Player{Steam64: m.id, Side: side(m.id), Alive: true, Listed: true, Pos: [3]float64{m.x}})
	}
	return round.Event{Event: match.Event{Fact: fact.Kill{Snapshot: s, Killer: k, Victim: v, Weapon: &w}}, Context: x}
}

var enemy = round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, EcoKill: true}

func run(events ...round.Event) *row.Round {
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{0, 1, 2, 11, 12, 13} {
		ms[id] = match.Member{Steam64: id, IsBot: id == 0}
	}
	return calc.Run(&round.Round{OpenMembers: ms, Events: events, Commit: len(events)})
}

// A trade: the avenger gets the trade kill, a fast one, a denial and a save; the victim is saved,
// and an opening death that is traded counts. The first killer's own pending trade is cleared.
func TestTrade(t *testing.T) {
	open := enemy
	open.Before.Opening = true
	trade := enemy
	trade.Avenged = []round.Death{{Steam: 1, Tick: 1000}}
	out := run(
		kill(1000, 11, 1, []mate{{12, 100}}, open),
		kill(1064, 2, 11, nil, trade),
	)
	if got, want := out.Players[2].Trades, (row.Trades{Trades: 1, TradeKills: 1, FastTrades: 1, TradeDenials: 1, SavedTeammate: 1}); got != want {
		t.Errorf("avenger %+v, want %+v", got, want)
	}
	if got, want := out.Players[1].Trades, (row.Trades{Traded: 1, SavedByTeammate: 1, OpeningDeathsTraded: 1}); got != want {
		t.Errorf("traded %+v, want %+v", got, want)
	}
	if got := out.Players[11].Trades; got != (row.Trades{}) {
		t.Errorf("first killer %+v: a trade clears his pending denial", got)
	}
	trade.Avenged[0].Tick = 1064
	if got := run(kill(1064, 11, 1, nil, open), kill(1064, 2, 11, nil, trade)).Players[2].Trades.FastTrades; got != 1 {
		t.Errorf("same-tick trade is fast %d times, want 1 (D42)", got)
	}
}

// A kill whose victim had a teammate within 1200 units, alive, and went untraded for over the
// window is a denial: once per pass however many teammates were near, and not at the round's end.
func TestDenialByExpiry(t *testing.T) {
	near := []mate{{12, 100}, {13, 1199}, {2, 50}} // 2 is the killer's own side: not a teammate of the victim
	out := run(
		kill(1000, 1, 11, near, enemy),
		kill(1320, 2, 12, nil, enemy), // 320 ticks on: still in the window
		kill(1321, 2, 13, nil, enemy), // 321: expired
	)
	if got := out.Players[1].Trades.TradeDenials; got != 1 {
		t.Errorf("denials %d, want 1", got)
	}
	if got := out.Players[2].Trades.TradeDenials; got != 0 {
		t.Errorf("player 2 denials %d, want 0: his own kills had no teammate near", got)
	}
	if got := run(kill(1000, 1, 11, near, enemy)).Players[1].Trades.TradeDenials; got != 0 {
		t.Errorf("denials %d at the round's end, want 0: expiry is lazy", got)
	}
}

// The victim's own pending trade is dropped before the expiry runs: he is not denied for it.
func TestVictimIsCleared(t *testing.T) {
	out := run(kill(1000, 11, 1, []mate{{2, 10}}, enemy), kill(1400, 2, 11, nil, enemy))
	if got := out.Players[11].Trades.TradeDenials; got != 0 {
		t.Errorf("denials %d, want 0", got)
	}
}

// Teammates too far away, or dead, make no pending trade.
func TestNoOneNear(t *testing.T) {
	far := []mate{{12, 1200}} // not below the limit
	dead := fact.Player{Steam64: 13, Side: fact.SideCT, Alive: false, Listed: true}
	k := kill(1000, 1, 11, far, enemy)
	f := k.Fact.(fact.Kill)
	f.Players = append(f.Players, dead)
	k.Fact = f
	if got := run(k, kill(2000, 2, 12, nil, enemy)).Players[1].Trades.TradeDenials; got != 0 {
		t.Errorf("denials %d, want 0", got)
	}
}

// Bots share row 0 and eco has none: a bot's trade and denial stay with CSC's trades field only.
func TestBotKiller(t *testing.T) {
	out := run(kill(1000, 0, 1, []mate{{2, 10}}, enemy), kill(2000, 11, 2, nil, enemy))
	if got := out.Players[0].Trades; got != (row.Trades{}) {
		t.Errorf("bot %+v, want nothing", got)
	}
	trade := enemy
	trade.Avenged = []round.Death{{Steam: 1, Tick: 1000}}
	out = run(kill(1000, 11, 1, nil, enemy), kill(1010, 0, 11, nil, trade))
	if got := out.Players[0].Trades; got != (row.Trades{Trades: 1}) {
		t.Errorf("bot trade %+v, want trades only", got)
	}
}

// Eco's kill handler skips a team kill and a suicide, but runs the expiry for a kill without a killer.
func TestExpiryWithoutKiller(t *testing.T) {
	world := round.Context{Expected: true, Victim: true}
	e := kill(1400, 1, 12, nil, world)
	f := e.Fact.(fact.Kill)
	f.Killer = nil
	e.Fact = f
	teamKill := round.Context{Expected: true, Killer: true, Victim: true}
	if got := run(kill(1000, 1, 11, []mate{{12, 10}}, enemy), e).Players[1].Trades.TradeDenials; got != 1 {
		t.Errorf("world death: denials %d, want 1", got)
	}
	if got := run(kill(1000, 1, 11, []mate{{12, 10}}, enemy), kill(1400, 2, 1, nil, teamKill)).Players[1].Trades.TradeDenials; got != 0 {
		t.Errorf("team kill: denials %d, want 0", got)
	}
}

// A human who joins mid-round has an eco row from the join: an earlier trade was a scratch row's.
func TestLateJoiner(t *testing.T) {
	trade := enemy
	trade.Avenged = []round.Death{{Steam: 1, Tick: 1000}}
	join := round.Event{Event: match.Event{Fact: fact.TeamChange{}, Join: &match.Member{Steam64: 99}}}
	e := kill(1010, 99, 11, nil, trade)
	out := calc.Run(&round.Round{OpenMembers: map[uint64]match.Member{1: {Steam64: 1}, 11: {Steam64: 11}},
		Events: []round.Event{e, join, e}, Commit: 3})
	if got := out.Players[99].Trades; got.TradeKills != 1 || got.SavedTeammate != 1 {
		t.Errorf("joiner %+v, want only the trade after the join", got)
	}
}
