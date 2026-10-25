package eco

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/rating"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

func TestTradeDetectorFindTradeableDeathConsumesOldest(t *testing.T) {
	td := NewTradeDetector()
	attackerID := uint64(99)
	td.recentKills[attackerID] = []recentKill{
		{VictimID: 1, VictimTeam: fact.SideCT, Tick: 10},
		{VictimID: 2, VictimTeam: fact.SideCT, Tick: 20},
	}

	attacker := &fact.Player{Side: fact.SideCT}
	victim := &fact.Player{Side: fact.SideT, Steam64: attackerID}

	first, ok := td.findTradeableDeath(attacker, victim, 50)
	if !ok || first.VictimID != 1 {
		t.Fatalf("first tradeable = %+v ok=%v, want victim 1", first, ok)
	}
	if len(td.recentKills[attackerID]) != 1 {
		t.Fatalf("expected 1 remaining kill, got %d", len(td.recentKills[attackerID]))
	}

	second, ok := td.findTradeableDeath(attacker, victim, 50)
	if !ok || second.VictimID != 2 {
		t.Fatalf("second tradeable = %+v ok=%v, want victim 2", second, ok)
	}

	_, ok = td.findTradeableDeath(attacker, victim, 50)
	if ok {
		t.Fatal("expected no further tradeable deaths (no double-credit)")
	}
}

func TestTradeDetectorOutOfWindowExcluded(t *testing.T) {
	td := NewTradeDetector()
	attackerID := uint64(5)
	td.recentKills[attackerID] = []recentKill{
		{VictimID: 1, VictimTeam: fact.SideCT, Tick: 0},
	}

	attacker := &fact.Player{Side: fact.SideCT}
	victim := &fact.Player{Side: fact.SideT, Steam64: attackerID}

	_, ok := td.findTradeableDeath(attacker, victim, rating.TradeWindowTicks+10)
	if ok {
		t.Fatal("expected out-of-window death to be excluded")
	}
}

func TestTradeDetectorRecordKillAppends(t *testing.T) {
	td := NewTradeDetector()
	attacker := &fact.Player{Steam64: 10, Side: fact.SideT}
	v1 := &fact.Player{Steam64: 1, Side: fact.SideCT}
	v2 := &fact.Player{Steam64: 2, Side: fact.SideCT}

	td.RecordKill(attacker, v1, 100)
	td.RecordKill(attacker, v2, 150)

	if got := len(td.recentKills[10]); got != 2 {
		t.Fatalf("expected 2 recent kills after two RecordKill calls, got %d", got)
	}
	if td.recentKills[10][0].VictimID != 1 || td.recentKills[10][1].VictimID != 2 {
		t.Fatalf("unexpected kill order: %+v", td.recentKills[10])
	}
}
