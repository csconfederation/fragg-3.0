package swing

import (
	"math"
	"testing"

	"github.com/csconfederation/fragg-3.0/rating/probability"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

func TestAllocateObjectiveEventPlantSharing(t *testing.T) {
	engine := probability.NewDefaultEngine()
	cfg := DefaultConfig()
	allocator := NewPoolAllocator(engine, cfg)

	state := probability.NewRoundState(3, 5, "de_dust2")
	players := map[uint64]PlayerRoundContext{
		100: {Side: fact.SideT, Kills: 1, Damage: 80, Alive: true},
		101: {Side: fact.SideT, Damage: 40, Alive: true},
		200: {Side: fact.SideCT, NegativeSwing: 0.05, Alive: true},
		201: {Side: fact.SideCT, NegativeSwing: 0.03, Alive: false},
	}

	input := ObjectiveAllocationInput{
		PrimaryID:     101,
		PrimarySide:   fact.SideT,
		OpposingSide:  fact.SideCT,
		Players:       players,
		PrimaryShare:  cfg.PlantPlanterShare,
		PrimaryReason: SwingReasonBombPlant,
		SupportReason: SwingReasonPlantSupport,
	}

	event, _ := allocator.AllocateObjectiveEvent(
		state,
		fact.SideT,
		engine.CalculateBombPlantSwing,
		func(s *probability.RoundState) { s.SetBombPlanted() },
		input,
	)

	if event.RawDelta <= 0 {
		t.Fatalf("expected positive plant delta, got %f", event.RawDelta)
	}

	positive := sumAllocations(event.PositiveAlloc)
	negative := sumAllocations(event.NegativeAlloc)
	if math.Abs(positive+negative) > cfg.ZeroSumTolerance {
		t.Fatalf("plant event not zero-sum: positive=%f negative=%f", positive, negative)
	}

	planterShare := positiveAmount(event.PositiveAlloc, 101)
	supportShare := positiveAmount(event.PositiveAlloc, 100)
	if planterShare <= 0 {
		t.Fatal("planter should receive positive plant swing")
	}
	if supportShare <= 0 {
		t.Fatal("entry player should receive plant support swing")
	}
}

func TestAllocateObjectiveEventDefuseSharing(t *testing.T) {
	engine := probability.NewDefaultEngine()
	cfg := DefaultConfig()
	allocator := NewPoolAllocator(engine, cfg)

	state := probability.NewRoundState(2, 3, "de_dust2")
	state.SetBombPlanted()

	players := map[uint64]PlayerRoundContext{
		300: {Side: fact.SideCT, Kills: 1, Alive: true},
		301: {Side: fact.SideCT, FlashAssists: 1, Alive: true},
		100: {Side: fact.SideT, NegativeSwing: 0.04, Alive: true},
	}

	input := ObjectiveAllocationInput{
		PrimaryID:     301,
		PrimarySide:   fact.SideCT,
		OpposingSide:  fact.SideT,
		Players:       players,
		PrimaryShare:  cfg.DefuseDefuserShare,
		PrimaryReason: SwingReasonBombDefuse,
		SupportReason: SwingReasonDefuseSupport,
	}

	event, amounts := allocator.AllocateObjectiveEvent(
		state,
		fact.SideCT,
		engine.CalculateBombDefuseSwing,
		func(s *probability.RoundState) { s.SetBombDefused() },
		input,
	)

	if !state.BombDefused {
		t.Fatal("expected BombDefused to be set after AllocateObjectiveEvent")
	}
	if event.RawDelta <= 0 {
		t.Fatalf("expected positive defuse delta via CalculateBombDefuseSwing, got %f", event.RawDelta)
	}
	if amounts[301] <= 0 {
		t.Fatal("defuser should receive positive swing")
	}

	positive := sumAllocations(event.PositiveAlloc)
	negative := sumAllocations(event.NegativeAlloc)
	if math.Abs(positive+negative) > cfg.ZeroSumTolerance {
		t.Fatalf("defuse event not zero-sum: positive=%f negative=%f", positive, negative)
	}
}

func TestAllocateObjectiveEventPlantAppliesStateEvenIfDeltaNonPositive(t *testing.T) {
	engine := probability.NewDefaultEngine()
	cfg := DefaultConfig()
	allocator := NewPoolAllocator(engine, cfg)

	// Any alive state: applyState must run regardless of delta sign.
	state := probability.NewRoundState(2, 1, "de_dust2")
	input := ObjectiveAllocationInput{
		PrimaryID:     1,
		PrimarySide:   fact.SideT,
		OpposingSide:  fact.SideCT,
		Players:       map[uint64]PlayerRoundContext{1: {Side: fact.SideT, Alive: true}},
		PrimaryShare:  cfg.PlantPlanterShare,
		PrimaryReason: SwingReasonBombPlant,
		SupportReason: SwingReasonPlantSupport,
	}

	_, _ = allocator.AllocateObjectiveEvent(
		state,
		fact.SideT,
		engine.CalculateBombPlantSwing,
		func(s *probability.RoundState) { s.SetBombPlanted() },
		input,
	)

	if !state.BombPlanted {
		t.Fatal("BombPlanted must be set even when plant delta is non-positive")
	}
}
