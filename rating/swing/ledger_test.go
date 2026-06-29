package swing

import (
	"math"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestValidateRoundSwingLedgerZeroSum(t *testing.T) {
	ledger := NewRoundSwingLedger(1)
	ledger.PlayerSwing[1] = 0.08
	ledger.PlayerSwing[2] = -0.05
	ledger.PlayerSwing[3] = -0.03

	warnings := ValidateRoundSwingLedger(ledger, 0.005)
	if len(warnings) != 0 {
		t.Fatalf("expected zero-sum ledger, got warnings: %v", warnings)
	}
}

func TestValidateRoundSwingLedgerDetectsImbalance(t *testing.T) {
	ledger := NewRoundSwingLedger(1)
	ledger.PlayerSwing[1] = 0.10
	ledger.PlayerSwing[2] = -0.05

	warnings := ValidateRoundSwingLedger(ledger, 0.005)
	if len(warnings) == 0 {
		t.Fatal("expected zero-sum warning for imbalanced ledger")
	}
}

func TestRecordEventUpdatesTotals(t *testing.T) {
	ledger := NewRoundSwingLedger(1)
	event := SwingLedgerEvent{
		PositiveAlloc: []PlayerSwingAllocation{{
			SteamID: 1, Side: common.TeamTerrorists, Amount: 0.08, Reason: SwingReasonKillFinalHit,
		}},
		NegativeAlloc: []PlayerSwingAllocation{{
			SteamID: 2, Side: common.TeamCounterTerrorists, Amount: -0.08, Reason: SwingReasonVictimDeath,
		}},
	}
	ledger.RecordEvent(event)

	if math.Abs(ledger.PlayerSwing[1]-0.08) > 1e-9 {
		t.Fatalf("killer swing = %f, want 0.08", ledger.PlayerSwing[1])
	}
	if math.Abs(ledger.PlayerSwing[2]+0.08) > 1e-9 {
		t.Fatalf("victim swing = %f, want -0.08", ledger.PlayerSwing[2])
	}
}
