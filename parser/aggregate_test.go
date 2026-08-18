package parser

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating/swing"
)

func TestFoldSnapshotsKeepsLastReplay(t *testing.T) {
	d := &DemoParser{state: NewMatchStateWithConfig(swing.DefaultConfig())}

	first := &RoundSnapshot{
		RoundNumber: 12,
		Stats: map[uint64]*model.RoundStats{
			1: {Kills: 2, Damage: 100, PlayerSide: "T", EconImpact: 1.2, ProbabilitySwing: 0.10},
		},
	}
	second := &RoundSnapshot{
		RoundNumber: 12,
		Stats: map[uint64]*model.RoundStats{
			1: {Kills: 1, Damage: 40, PlayerSide: "T", EconImpact: 0.8, ProbabilitySwing: 0.03},
		},
	}
	_ = first

	// CSC removeInvalidRounds keeps the last RoundNum=12 playthrough.
	d.FoldSnapshots([]*RoundSnapshot{second})

	p := d.GetPlayers()[1]
	if p == nil {
		t.Fatal("expected folded player 1")
	}
	if p.Kills != 1 || p.Damage != 40 {
		t.Fatalf("kills=%d damage=%d, want last playthrough (1, 40); first playthrough leaked", p.Kills, p.Damage)
	}
	if p.RoundsPlayed != 1 {
		t.Fatalf("RoundsPlayed=%d, want 1", p.RoundsPlayed)
	}
	if d.GetRoundsCounted() != 1 {
		t.Fatalf("RoundsCounted=%d, want 1", d.GetRoundsCounted())
	}
}

func TestFoldSnapshotsDropsDuplicateThenCountsSurvivors(t *testing.T) {
	d := &DemoParser{state: NewMatchStateWithConfig(swing.DefaultConfig())}

	r11 := &RoundSnapshot{
		RoundNumber: 11,
		Stats: map[uint64]*model.RoundStats{
			1: {Kills: 1, Damage: 50, PlayerSide: "CT"},
		},
	}
	r12a := &RoundSnapshot{
		RoundNumber: 12,
		Stats: map[uint64]*model.RoundStats{
			1: {Kills: 5, Damage: 500, PlayerSide: "T"},
		},
	}
	r12b := &RoundSnapshot{
		RoundNumber: 12,
		Stats: map[uint64]*model.RoundStats{
			1: {Kills: 0, Damage: 10, PlayerSide: "T"},
		},
	}
	_ = r12a

	d.FoldSnapshots([]*RoundSnapshot{r11, r12b})
	p := d.GetPlayers()[1]
	if p.Kills != 1 || p.Damage != 60 {
		t.Fatalf("kills=%d damage=%d, want 1+0 kills and 50+10 damage (dropped r12a)", p.Kills, p.Damage)
	}
	if p.RoundsPlayed != 2 {
		t.Fatalf("RoundsPlayed=%d, want 2", p.RoundsPlayed)
	}
	if d.GetRoundsCounted() != 2 {
		t.Fatalf("RoundsCounted=%d, want 2", d.GetRoundsCounted())
	}
}
