package probability

import (
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestEnforceMonotonicityPeerReviewTransitions(t *testing.T) {
	tables := DefaultTables()

	// After enforcement, killing a CT must never lower T-win WP.
	cases := []struct {
		name              string
		tBefore, ctBefore int
		tAfter, ctAfter   int
		planted           bool
	}{
		{"planted 5v5→5v4", 5, 5, 5, 4, true},
		{"planted 4v5→4v4", 4, 5, 4, 4, true},
		{"planted 2v2→2v1", 2, 2, 2, 1, true},
		{"planted 1v2→1v1", 1, 2, 1, 1, true},
		{"none 1v2→1v1", 1, 2, 1, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tables.GetBaseWinProbability(tc.tBefore, tc.ctBefore, tc.planted, false)
			after := tables.GetBaseWinProbability(tc.tAfter, tc.ctAfter, tc.planted, false)
			if after < before {
				t.Fatalf("T-win decreased after CT death: %.3f → %.3f", before, after)
			}
		})
	}

	// CT kill path: 2v2→1v2 none must not raise T-win (hurting CT).
	t.Run("none 2v2→1v2 CT kill", func(t *testing.T) {
		before := tables.GetBaseWinProbability(2, 2, false, false)
		after := tables.GetBaseWinProbability(1, 2, false, false)
		if after > before {
			t.Fatalf("T-win rose after T death: %.3f → %.3f", before, after)
		}
	})

	// Planting must never lower T-win.
	t.Run("2v1 none→planted", func(t *testing.T) {
		none := tables.GetBaseWinProbability(2, 1, false, false)
		planted := tables.GetBaseWinProbability(2, 1, true, false)
		if planted < none {
			t.Fatalf("plant lowered T-win: none=%.3f planted=%.3f", none, planted)
		}
	})
}

func TestBombDefusedWinProbability(t *testing.T) {
	engine := NewDefaultEngine()
	state := NewRoundState(2, 3, "de_dust2")
	state.SetBombPlanted()
	state.SetBombDefused()

	tProb := engine.GetWinProbability(state, common.TeamTerrorists)
	ctProb := engine.GetWinProbability(state, common.TeamCounterTerrorists)
	if tProb > 0.02 {
		t.Fatalf("defused T WP = %f, want ~0", tProb)
	}
	if ctProb < 0.98 {
		t.Fatalf("defused CT WP = %f, want ~1", ctProb)
	}
}

func TestCalculateBombDefuseSwingNonZero(t *testing.T) {
	engine := NewDefaultEngine()
	state := NewRoundState(2, 3, "de_dust2")
	state.SetBombPlanted()

	delta := engine.CalculateBombDefuseSwing(state)
	if delta <= 0 {
		t.Fatalf("defuse swing = %f, want > 0", delta)
	}
}

func TestApplyTimeAdjustmentLowTime(t *testing.T) {
	engine := NewDefaultEngine()
	state := NewRoundState(2, 2, "de_dust2")
	state.SetBombPlanted()
	state.SetTimeRemaining(4.0)

	base := engine.tables.GetBaseWinProbability(2, 2, true, false)
	adjusted := engine.applyTimeAdjustment(base, state)
	if adjusted <= base {
		t.Fatalf("expected time boost at 4s remaining: base=%f adjusted=%f", base, adjusted)
	}
}
