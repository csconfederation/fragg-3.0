package swing

import (
	"github.com/ethsmith/eco-rating/rating/probability"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// Calculator computes probability-based round swing for players.
type Calculator struct {
	probEngine *probability.Engine
	cfg        Config
	allocator  *PoolAllocator
}

// NewCalculator creates a new swing calculator with the given probability engine.
func NewCalculator(engine *probability.Engine) *Calculator {
	cfg := DefaultConfig()
	return &Calculator{
		probEngine: engine,
		cfg:        cfg,
		allocator:  NewPoolAllocator(engine, cfg),
	}
}

// NewCalculatorWithConfig creates a calculator with custom swing config.
func NewCalculatorWithConfig(engine *probability.Engine, cfg Config) *Calculator {
	return &Calculator{
		probEngine: engine,
		cfg:        cfg,
		allocator:  NewPoolAllocator(engine, cfg),
	}
}

// NewDefaultCalculator creates a swing calculator with default probability tables.
func NewDefaultCalculator() *Calculator {
	return NewCalculator(probability.NewDefaultEngine())
}

// RoundSwingResult contains the swing values for all players in a round.
type RoundSwingResult struct {
	PlayerSwings map[uint64]float64
	TotalTSwing  float64
	TotalCTSwing float64
}

// KillSwingResult contains swing metadata for a single kill event.
type KillSwingResult struct {
	RawSwing          float64
	KillerSwing       float64
	VictimSwing       float64
	EcoMultiplier     float64
	ContributorSwings map[uint64]float64
}

// CalculateRoundSwing computes swing for all players based on round events.
func (c *Calculator) CalculateRoundSwing(
	events []RoundEvent,
	initialState *probability.RoundState,
	result *RoundResult,
) *RoundSwingResult {
	ledger := NewRoundSwingLedger(0)
	state := initialState.Clone()
	players := make(map[uint64]PlayerRoundContext)

	for _, event := range events {
		switch e := event.(type) {
		case *KillEvent:
			input := KillAllocationInput{Kill: e}
			ledgerEvent, _ := c.allocator.AllocateKillEvent(state, input)
			ledger.RecordEvent(ledgerEvent)
			state.RecordDeath(e.VictimSide)
			updatePlayerContextFromKill(players, e, ledgerEvent)
		case *BombPlantEvent:
			input := ObjectiveAllocationInput{
				PrimaryID:     e.PlanterID,
				PrimarySide:   common.TeamTerrorists,
				OpposingSide:  common.TeamCounterTerrorists,
				Players:       players,
				PrimaryShare:  c.cfg.PlantPlanterShare,
				PrimaryReason: SwingReasonBombPlant,
				SupportReason: SwingReasonPlantSupport,
			}
			ledgerEvent, _ := c.allocator.AllocateObjectiveEvent(
				state,
				common.TeamTerrorists,
				c.probEngine.CalculateBombPlantSwing,
				func(s *probability.RoundState) { s.SetBombPlanted() },
				input,
				SwingEventBombPlant,
			)
			ledger.RecordEvent(ledgerEvent)
		case *BombDefuseEvent:
			input := ObjectiveAllocationInput{
				PrimaryID:     e.DefuserID,
				PrimarySide:   common.TeamCounterTerrorists,
				OpposingSide:  common.TeamTerrorists,
				Players:       players,
				PrimaryShare:  c.cfg.DefuseDefuserShare,
				PrimaryReason: SwingReasonBombDefuse,
				SupportReason: SwingReasonDefuseSupport,
			}
			ledgerEvent, _ := c.allocator.AllocateObjectiveEvent(
				state,
				common.TeamCounterTerrorists,
				c.probEngine.CalculateBombDefuseSwing,
				func(s *probability.RoundState) { s.SetBombDefused() },
				input,
				SwingEventBombDefuse,
			)
			ledger.RecordEvent(ledgerEvent)
		case *BombExplodeEvent:
			_ = e
		}
	}

	if result != nil && c.cfg.ResidualEnabled {
		currentProb := c.probEngine.GetWinProbability(state, result.Winner)
		ApplyRoundResidual(ledger, ResidualInput{
			Winner:         result.Winner,
			CurrentWinProb: currentProb,
			Players:        players,
		}, c.cfg)
	}

	ValidateRoundSwingLedger(ledger, c.cfg.ZeroSumTolerance)
	return &RoundSwingResult{PlayerSwings: ledger.PlayerTotals()}
}

func updatePlayerContextFromKill(players map[uint64]PlayerRoundContext, kill *KillEvent, event SwingLedgerEvent) {
	for _, alloc := range event.PositiveAlloc {
		ctx := players[alloc.SteamID]
		ctx.SteamID = alloc.SteamID
		ctx.Side = alloc.Side
		if alloc.Amount > 0 {
			ctx.PositiveSwing += alloc.Amount
		}
		ctx.Kills++
		players[alloc.SteamID] = ctx
	}
	for _, alloc := range event.NegativeAlloc {
		ctx := players[alloc.SteamID]
		ctx.SteamID = alloc.SteamID
		ctx.Side = alloc.Side
		if alloc.Amount < 0 {
			ctx.NegativeSwing += -alloc.Amount
		}
		players[alloc.SteamID] = ctx
	}
	if kill != nil {
		killerCtx := players[kill.KillerID]
		killerCtx.SteamID = kill.KillerID
		killerCtx.Side = kill.KillerSide
		killerCtx.Kills++
		killerCtx.Damage += kill.KillerDamageDealt
		players[kill.KillerID] = killerCtx
	}
}

// GetProbabilityEngine returns the underlying probability engine.
func (c *Calculator) GetProbabilityEngine() *probability.Engine {
	return c.probEngine
}
