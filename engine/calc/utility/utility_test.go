package utility

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

func ev(f fact.Fact, x round.Context) round.Event {
	return round.Event{Event: match.Event{Fact: f, Live: true}, Context: x}
}

func TestCalc(t *testing.T) {
	a, b := &fact.Player{Steam64: 1}, &fact.Player{Steam64: 2}
	he, smoke, flash, molly, inc, decoy := fact.EqHE, fact.EqSmoke, fact.EqFlash, fact.EqMolotov, fact.EqIncendiary, fact.EqKnife
	dropped := ev(fact.GrenadeThrow{Thrower: a, Weapon: &he}, round.Context{})
	dropped.Dropped = true
	r := &round.Round{Events: []round.Event{
		ev(fact.Flashed{Attacker: a, Player: b, Duration: 1.5}, round.Context{Flash: true, Blind: round.Blind{Flasher: 1, Until: 99.5}}),
		ev(fact.Flashed{Attacker: a, Player: b, Duration: 3}, round.Context{}), // a blind that doesn't count
		ev(fact.Kill{Killer: a, Victim: b}, round.Context{FlashAssisted: true, FlashAssister: 2}),
		ev(fact.Kill{Killer: a, Victim: b}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: a, Weapon: &he}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: a, Weapon: &smoke}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: b, Weapon: &flash}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: b, Weapon: &molly}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: b, Weapon: &inc}, round.Context{}),
		ev(fact.GrenadeThrow{Thrower: b, Weapon: &decoy}, round.Context{}),                     // not counted
		ev(fact.GrenadeThrow{Thrower: nil, Weapon: &he}, round.Context{}),                      // unresolved thrower
		ev(fact.GrenadeThrow{Thrower: &fact.Player{Steam64: 9}, Weapon: &he}, round.Context{}), // no row
		dropped,
		ev(fact.GrenadeThrow{Thrower: a, Weapon: &he}, round.Context{}), // after the commit
	}, Commit: 13, OpenMembers: map[uint64]match.Member{1: {}, 2: {}}}
	r.Events = append([]round.Event{ev(fact.GrenadeThrow{Thrower: &fact.Player{Steam64: 7}, Weapon: &he}, round.Context{}), // before 7 joins
		{Event: match.Event{Fact: fact.TeamChange{}, Live: true, Join: &match.Member{Steam64: 7}}},
		ev(fact.GrenadeThrow{Thrower: &fact.Player{Steam64: 7}, Weapon: &he}, round.Context{})}, r.Events...)
	r.Commit += 3
	out := &row.Round{Players: map[uint64]*row.Player{1: {}, 2: {}, 7: {}}}
	Calc(r, out)
	want1 := row.Utility{Ef: 1, EnemyFlashTime: 1.5, NadesThrown: 1, SmokeThrown: 1}
	want2 := row.Utility{FAss: 1, MostRecentFlasher: 1, MostRecentFlashVal: 99.5, FlashThrown: 1, FiresThrown: 2}
	if got := out.Players[1].Utility; got != want1 {
		t.Errorf("player 1: %+v, want %+v", got, want1)
	}
	if got := out.Players[7].Utility; got != (row.Utility{NadesThrown: 1}) {
		t.Errorf("player 7, who joined after a throw: %+v", got)
	}
	if got := out.Players[2].Utility; got != want2 {
		t.Errorf("player 2: %+v, want %+v", got, want2)
	}
}
