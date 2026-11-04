package cscstats

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/match"
	shared "github.com/csconfederation/fragg-3.0/engine/round"
)

// Bonus lookups outside the defined range skip the bonus instead of failing the match (the oracle
// indexes out of range): bots share one row, which can hold six kills, and a miscounted roster
// can leave a clutch of size -1.
func TestCommitBoundsBonuses(t *testing.T) {
	c := New()
	r := &match.Round{Num: 5}
	c.rounds[r] = &round{RoundNum: 5, WinnerENUM: 2, WinnerClanName: "A",
		PlayerStats: map[uint64]*playerStats{
			0: {SteamID: "0", IsBot: true, Side: 2, TeamENUM: 2, TeamClanName: "A", Kills: 6, DamageList: map[uint64]int{}},
			7: {SteamID: "7", Side: 2, TeamENUM: 2, TeamClanName: "A", Kills: 1, DamageList: map[uint64]int{}},
			9: {SteamID: "9", Side: 3, TeamENUM: 3, TeamClanName: "B", Deaths: 1, DamageList: map[uint64]int{}},
		},
		TeamStats: map[string]*teamStats{"A": {TR: 1}, "B": {CtR: 1}}}
	s := shared.State{TClutch: shared.Clutch{Size: -1, Steam: 7}}
	if err := c.Commit(nil, r, 1000, s); err != nil {
		t.Fatal(err)
	}
	pr := c.rounds[r]
	if bot := pr.PlayerStats[0]; bot.ImpactPoints < multiKillBonus[5] {
		t.Errorf("bot row impact %v, want at least the five-kill bonus %v", bot.ImpactPoints, multiKillBonus[5])
	}
	if p := pr.PlayerStats[7]; p.Cl_1+p.Cl_2+p.Cl_3+p.Cl_4+p.Cl_5 != 0 || pr.TeamStats["A"].Clutches != 0 {
		t.Errorf("clutch of size -1 recorded: %+v", p)
	}
}
