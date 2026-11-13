package survival

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

func kill(victim *fact.Player, tick int, after round.State, counted bool) round.Event {
	return round.Event{Event: match.Event{Fact: fact.Kill{Snapshot: fact.Snapshot{Tick: tick}, Victim: victim}, Live: true},
		Context: round.Context{After: after, Victim: counted}}
}

// T 1 and 2 against CT 3 and 4, the round starting at tick 1000: T 1 dies at 1640 (10 s in), CT 3 at 3200
// (34.3 s) and a bot at 1700; T wins at 4000. A Win's effect index follows its RoundEnd.
func TestCalc(t *testing.T) {
	t1, c3 := &fact.Player{Steam64: 1, Side: fact.SideT}, &fact.Player{Steam64: 3, Side: fact.SideCT}
	bot := &fact.Player{Steam64: 0, Side: fact.SideCT, IsBot: true}
	end := round.Event{Event: match.Event{Fact: fact.RoundEnd{Snapshot: fact.Snapshot{Tick: 4000}, Winner: fact.SideT}, Live: true,
		Win: true, Winner: "a"}}
	late := kill(c3, 4100, round.State{}, true) // after the commit
	r := &round.Round{StartTick: 1000, InitT: 2, InitCT: 3, TClan: "a", CTClan: "b",
		OpenMembers: map[uint64]match.Member{
			1: {Steam64: 1, Side: fact.SideT, TeamID: 2, Clan: "a"}, 2: {Steam64: 2, Side: fact.SideT, TeamID: 2, Clan: "a"},
			3: {Steam64: 3, Side: fact.SideCT, TeamID: 3, Clan: "b"}, 4: {Steam64: 4, Side: fact.SideCT, TeamID: 3, Clan: "b"},
			0: {Steam64: 0, IsBot: true, Side: fact.SideCT, TeamID: 3, Clan: "b"}},
		Events: []round.Event{
			kill(t1, 1640, round.State{TAlive: 1, CTAlive: 3}, true),
			kill(bot, 1700, round.State{TAlive: 1, CTAlive: 2}, true),
			kill(c3, 3200, round.State{TAlive: 1, CTAlive: 1}, true),
			kill(c3, 3300, round.State{}, false), // a death the round doesn't count
			end, late},
		Wins: []int{5}, Commit: 5}
	out := &row.Round{Players: map[uint64]*row.Player{}, Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	Calc(r, out)

	want := map[uint64]row.Survival{
		1: {DeathTick: 1640, DeathPlacement: 1, TicksAlive: 640, EarlyDeaths: 1, EcoDeathTime: 10, EcoTimeAlive: 10},
		2: {EcoTimeAlive: 46.875}, // a survivor on the winning side: the Win's time, no save
		3: {DeathTick: 3200, DeathPlacement: 2, TicksAlive: 2200, EcoDeathTime: 34.375, EcoTimeAlive: 34.375},
		4: {Saves: 1, EcoTimeAlive: 46.875},
		0: {DeathTick: 1700, DeathPlacement: 1, TicksAlive: 700}, // the bot's row: no eco values
	}
	for id, w := range want {
		if got := out.Players[id].Survival; got != w {
			t.Errorf("player %d: %+v, want %+v", id, got, w)
		}
	}
	if out.Teams[fact.SideCT].Saves != 1 || out.Teams[fact.SideT].Saves != 0 {
		t.Errorf("team saves %+v %+v", out.Teams[fact.SideT], out.Teams[fact.SideCT])
	}
}

// D44: a member who disconnects without dying is not a survivor, so a loser who leaves is not a save.
func TestDisconnectedIsNotASave(t *testing.T) {
	end := round.Event{Event: match.Event{Fact: fact.RoundEnd{Snapshot: fact.Snapshot{Tick: 4000}, Winner: fact.SideT}, Live: true,
		Win: true, Winner: "a"}}
	left := round.Event{Event: match.Event{Fact: fact.Disconnect{Steam64: 4}, Live: true}}
	r := &round.Round{StartTick: 1000, TClan: "a", CTClan: "b",
		OpenMembers: map[uint64]match.Member{
			2: {Steam64: 2, Side: fact.SideT, TeamID: 2, Clan: "a"},
			3: {Steam64: 3, Side: fact.SideCT, TeamID: 3, Clan: "b"}, 4: {Steam64: 4, Side: fact.SideCT, TeamID: 3, Clan: "b"}},
		Events: []round.Event{left, end}, Wins: []int{1}, Commit: 2}
	out := &row.Round{Players: map[uint64]*row.Player{}, Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	Calc(r, out)
	if p := out.Players[4]; p.Saves != 0 || p.LeftAlive != 1 {
		t.Errorf("disconnected loser %+v: want no save, LeftAlive 1", p.Survival)
	}
	if p := out.Players[3]; p.Saves != 1 || p.LeftAlive != 0 {
		t.Errorf("loser who stayed %+v: want a save", p.Survival)
	}
	if out.Teams[fact.SideCT].Saves != 1 {
		t.Errorf("team saves %d, want 1 (player 3 only)", out.Teams[fact.SideCT].Saves)
	}
}
