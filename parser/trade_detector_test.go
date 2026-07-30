package parser

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/rating"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestTradeDetectorFindTradeableDeathConsumesOldest(t *testing.T) {
	td := NewTradeDetector()
	attackerID := uint64(99)
	td.recentKills[attackerID] = []recentKill{
		{VictimID: 1, VictimTeam: common.TeamCounterTerrorists, Tick: 10},
		{VictimID: 2, VictimTeam: common.TeamCounterTerrorists, Tick: 20},
	}

	attacker := &common.Player{Team: common.TeamCounterTerrorists}
	victim := &common.Player{Team: common.TeamTerrorists, SteamID64: attackerID}

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
		{VictimID: 1, VictimTeam: common.TeamCounterTerrorists, Tick: 0},
	}

	attacker := &common.Player{Team: common.TeamCounterTerrorists}
	victim := &common.Player{Team: common.TeamTerrorists, SteamID64: attackerID}

	_, ok := td.findTradeableDeath(attacker, victim, rating.TradeWindowTicks+10)
	if ok {
		t.Fatal("expected out-of-window death to be excluded")
	}
}

func TestTradeDetectorRecordKillAppends(t *testing.T) {
	td := NewTradeDetector()
	attacker := &common.Player{SteamID64: 10, Team: common.TeamTerrorists}
	v1 := &common.Player{SteamID64: 1, Team: common.TeamCounterTerrorists}
	v2 := &common.Player{SteamID64: 2, Team: common.TeamCounterTerrorists}

	td.RecordKill(attacker, v1, 100)
	td.RecordKill(attacker, v2, 150)

	if got := len(td.recentKills[10]); got != 2 {
		t.Fatalf("expected 2 recent kills after two RecordKill calls, got %d", got)
	}
	if td.recentKills[10][0].VictimID != 1 || td.recentKills[10][1].VictimID != 2 {
		t.Fatalf("unexpected kill order: %+v", td.recentKills[10])
	}
}
