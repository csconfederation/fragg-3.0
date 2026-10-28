package probability

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
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

	tProb := engine.GetWinProbability(state, fact.SideT)
	ctProb := engine.GetWinProbability(state, fact.SideCT)
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

// The base table already carries the all-maps T rate, so a map at that rate (or an unknown map)
// leaves the base probability unchanged.
func TestMapFactorRelativeToBaseline(t *testing.T) {
	tables := NewProbabilityTables()
	tables.BaseWinProb["5v5_none"] = 0.48
	tables.MapBaseline = 0.48
	tables.MapAdjustments["de_even"] = 0.48
	tables.MapAdjustments["de_tsided"] = 0.528
	e := NewEngine(tables)
	for name, want := range map[string]float64{"de_even": 0.48, "de_unknown": 0.48, "de_tsided": 0.528} {
		got := e.GetWinProbability(&RoundState{TAlive: 5, CTAlive: 5, Map: name}, fact.SideT)
		if d := got - want; d > 1e-9 || d < -1e-9 {
			t.Errorf("%s: T win %.4f, want %.4f", name, got, want)
		}
	}
}

// The last kill never lowers the killer's side, even on a map whose factor is below 1 with a
// near-certain state before it (review72: 5v1 → 5v0 on nuke).
func TestDecidedRoundsAreTerminal(t *testing.T) {
	e := NewDefaultEngine()
	for _, tc := range []struct {
		name          string
		before, after RoundState
		side          fact.Side
	}{
		{"last CT", RoundState{TAlive: 5, CTAlive: 1, Map: "de_nuke"}, RoundState{TAlive: 5, CTAlive: 0, Map: "de_nuke"}, fact.SideT},
		{"last CT, planted", RoundState{TAlive: 5, CTAlive: 1, BombPlanted: true, Map: "de_nuke"}, RoundState{TAlive: 5, CTAlive: 0, BombPlanted: true, Map: "de_nuke"}, fact.SideT},
		{"last T", RoundState{TAlive: 1, CTAlive: 5, Map: "de_anubis"}, RoundState{TAlive: 0, CTAlive: 5, Map: "de_anubis"}, fact.SideCT},
	} {
		before, after := e.GetWinProbability(&tc.before, tc.side), e.GetWinProbability(&tc.after, tc.side)
		if after != 1 || after <= before {
			t.Errorf("%s: %.4f → %.4f", tc.name, before, after)
		}
	}
	// Ts all dead with the bomb planted is the defuse race, still measured.
	if p := e.GetWinProbability(&RoundState{TAlive: 0, CTAlive: 1, BombPlanted: true}, fact.SideT); p <= 0 || p >= 1 {
		t.Errorf("0v1 planted: %.4f", p)
	}
}
