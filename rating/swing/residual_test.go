package swing

import (
	"math"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

func TestApplyRoundResidualSaveAfterAdvantage(t *testing.T) {
	cfg := DefaultConfig()
	ledger := NewRoundSwingLedger()

	players := map[uint64]PlayerRoundContext{
		100: {Side: fact.SideT, Alive: true, PositiveSwing: 0.10, Kills: 2},
		101: {Side: fact.SideT, Alive: true, PositiveSwing: 0.08, Kills: 1},
		200: {Side: fact.SideCT, Alive: true, NegativeSwing: 0.05},
		201: {Side: fact.SideCT, Alive: true, NegativeSwing: 0.02},
	}

	allocs := ApplyRoundResidual(ledger, ResidualInput{
		Winner:           fact.SideT,
		CurrentWinProb:   0.82,
		Players:          players,
		MaxDamageInRound: 200,
	}, cfg)

	if !ledger.ResidualApplied {
		t.Fatal("expected residual to be applied")
	}

	residual := 1.0 - 0.82
	if residual > cfg.ResidualMax {
		residual = cfg.ResidualMax
	}

	positive := 0.0
	negative := 0.0
	for _, a := range allocs {
		if a.Amount > 0 {
			positive += a.Amount
		} else {
			negative += a.Amount
		}
	}

	if math.Abs(positive-residual) > 0.01 {
		t.Fatalf("positive residual = %f, want ~%f", positive, residual)
	}
	if math.Abs(negative+residual) > 0.01 {
		t.Fatalf("negative residual = %f, want ~-%f", negative, residual)
	}

	savePenalty := amountForPlayer(allocs, 200) + amountForPlayer(allocs, 201)
	if savePenalty >= 0 {
		t.Fatalf("expected saving CTs to receive negative residual, got %f", savePenalty)
	}
}

func TestApplyRoundResidualDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResidualEnabled = false
	ledger := NewRoundSwingLedger()

	allocs := ApplyRoundResidual(ledger, ResidualInput{
		Winner:         fact.SideT,
		CurrentWinProb: 0.82,
	}, cfg)

	if len(allocs) != 0 {
		t.Fatalf("expected no residual when disabled, got %d allocations", len(allocs))
	}
}
