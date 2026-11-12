package swing

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Two Ts (1, 2) against two CTs (3, 4): T 1 hurts CT 3 for 100 at tick 640 and kills 3 at tick 768
// (2 s later); a bot kills CT 4, and T 1 wins.
func TestCalc(t *testing.T) {
	weapon := fact.EqAWP
	t1 := &fact.Player{Steam64: 1, Side: fact.SideT, EquipValue: 4000}
	c3 := &fact.Player{Steam64: 3, Side: fact.SideCT, EquipValue: 4000}
	c4 := &fact.Player{Steam64: 4, Side: fact.SideCT, EquipValue: 4000}
	bot := &fact.Player{Steam64: 0, IsBot: true, Side: fact.SideT, EquipValue: 4000}
	open := fact.Snapshot{Players: []fact.Player{{Steam64: 1, Side: fact.SideT, Alive: true}, {Steam64: 2, Side: fact.SideT, Alive: true},
		{Steam64: 3, Side: fact.SideCT, Alive: true}, {Steam64: 4, Side: fact.SideCT, Alive: true}}}
	hurt := round.Event{Event: match.Event{Fact: fact.Hurt{Snapshot: fact.Snapshot{Tick: 640}, Attacker: t1, Player: c3, HealthDamageTaken: 100}, Live: true},
		Context: round.Context{EcoDamage: true, Enemy: true, After: round.State{TAlive: 2, CTAlive: 2}}}
	kill := func(k, v *fact.Player, tick int, after round.State, alive ...fact.Player) round.Event {
		return round.Event{Event: match.Event{Fact: fact.Kill{Snapshot: fact.Snapshot{Tick: tick}, Killer: k, Victim: v, Weapon: &weapon}, Live: true},
			Context: round.Context{Expected: true, EcoKill: true, Victim: true, Killer: true, Enemy: true, After: after, Alive: alive}}
	}
	end := round.Event{Event: match.Event{Fact: fact.RoundEnd{Snapshot: fact.Snapshot{Tick: 1000, Players: []fact.Player{
		{Steam64: 1, Side: fact.SideT, Alive: true}, {Steam64: 2, Side: fact.SideT, Alive: true}}}, Winner: fact.SideT},
		Live: true, Win: true, Winner: "a", Roster: match.Roster{T: []int{0, 1}}}}
	r := &round.Round{Num: 2, Map: "de_dust2", InitT: 2, InitCT: 2, TClan: "a", CTClan: "b", Open: open,
		Roster: match.Roster{T: []int{0, 1}, CT: []int{2, 3}},
		OpenMembers: map[uint64]match.Member{1: {Steam64: 1, Side: fact.SideT}, 2: {Steam64: 2, Side: fact.SideT},
			3: {Steam64: 3, Side: fact.SideCT}, 4: {Steam64: 4, Side: fact.SideCT}},
		Events: []round.Event{hurt,
			kill(t1, c3, 768, round.State{TAlive: 2, CTAlive: 1}, fact.Player{Steam64: 1, Side: fact.SideT}, fact.Player{Steam64: 2, Side: fact.SideT}, fact.Player{Steam64: 4, Side: fact.SideCT}),
			kill(bot, c4, 900, round.State{TAlive: 2, CTAlive: 0}, fact.Player{Steam64: 1, Side: fact.SideT}, fact.Player{Steam64: 2, Side: fact.SideT}),
			end},
		Wins: []int{4}, Commit: 4}
	out := &row.Round{Players: map[uint64]*row.Player{}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	out.Players[0] = &row.Player{}
	Calc(r, out)

	k, v := out.Players[1].Swing, out.Players[3].Swing
	if k.TimeToKillSum != 2 || k.KillsWithTTK != 1 {
		t.Errorf("time to kill %+v", k)
	}
	if k.EcoProbabilitySwing <= 0 || v.EcoProbabilitySwing >= 0 || k.EcoAdjustedKills <= 0 {
		t.Errorf("killer %+v, victim %+v", k, v)
	}
	if b := out.Players[0].Swing; b.EcoProbabilitySwing != 0 || b.KillsWithTTK != 0 {
		t.Errorf("the bot's row: %+v", b)
	}
}
