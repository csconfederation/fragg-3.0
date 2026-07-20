package swing

import (
	"math"
	"testing"

	"github.com/ethsmith/eco-rating/rating/probability"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestAllocateObjectiveEventPlantSharing(t *testing.T) {
	engine := probability.NewDefaultEngine()
	cfg := DefaultConfig()
	allocator := NewPoolAllocator(engine, cfg)

	state := probability.NewRoundState(3, 5, "de_dust2")
	players := map[uint64]PlayerRoundContext{
		100: {SteamID: 100, Side: common.TeamTerrorists, Kills: 1, Damage: 80, Alive: true},
		101: {SteamID: 101, Side: common.TeamTerrorists, Damage: 40, Alive: true},
		200: {SteamID: 200, Side: common.TeamCounterTerrorists, NegativeSwing: 0.05, Alive: true},
		201: {SteamID: 201, Side: common.TeamCounterTerrorists, NegativeSwing: 0.03, Alive: false},
	}

	input := ObjectiveAllocationInput{
		PrimaryID:     101,
		PrimarySide:   common.TeamTerrorists,
		OpposingSide:  common.TeamCounterTerrorists,
		Players:       players,
		PrimaryShare:  cfg.PlantPlanterShare,
		PrimaryReason: SwingReasonBombPlant,
		SupportReason: SwingReasonPlantSupport,
	}

	event, _ := allocator.AllocateObjectiveEvent(
		state,
		common.TeamTerrorists,
		engine.CalculateBombPlantSwing,
		func(s *probability.RoundState) { s.SetBombPlanted() },
		input,
		SwingEventBombPlant,
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
	cfg := DefaultConfig()
	allocator := NewPoolAllocator(probability.NewDefaultEngine(), cfg)

	players := map[uint64]PlayerRoundContext{
		300: {SteamID: 300, Side: common.TeamCounterTerrorists, Kills: 1, Alive: true},
		301: {SteamID: 301, Side: common.TeamCounterTerrorists, FlashAssists: 1, Alive: true},
		100: {SteamID: 100, Side: common.TeamTerrorists, NegativeSwing: 0.04, Alive: true},
	}

	input := ObjectiveAllocationInput{
		PrimaryID:     301,
		PrimarySide:   common.TeamCounterTerrorists,
		OpposingSide:  common.TeamTerrorists,
		Players:       players,
		PrimaryShare:  cfg.DefuseDefuserShare,
		PrimaryReason: SwingReasonBombDefuse,
		SupportReason: SwingReasonDefuseSupport,
	}

	pool := 0.30
	positive := allocator.allocateObjectivePositive(input, pool)
	negative := allocator.allocateObjectiveNegative(input, pool)

	defuserShare := positiveAmount(positive, 301)
	supportShare := positiveAmount(positive, 300)
	if defuserShare <= 0 || supportShare <= 0 {
		t.Fatalf("expected defuser (%f) and support (%f) to receive credit", defuserShare, supportShare)
	}

	posTotal := sumAllocations(positive)
	negTotal := sumAllocations(negative)
	if math.Abs(posTotal-pool) > 0.001 {
		t.Fatalf("positive allocation = %f, want %f", posTotal, pool)
	}
	if math.Abs(negTotal+pool) > 0.001 {
		t.Fatalf("negative allocation = %f, want -%f", negTotal, pool)
	}
}
