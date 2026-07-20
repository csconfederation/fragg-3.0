package parser

import (
	"github.com/ethsmith/eco-rating/rating/probability"
	"github.com/ethsmith/eco-rating/rating/swing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// SwingTracker manages round state and swing calculation during parsing.
type SwingTracker struct {
	engine           *probability.Engine
	allocator        *swing.PoolAllocator
	damageTracker    *DamageTracker
	advantageTracker *AdvantageTracker
	roundState       *probability.RoundState
	ledger           *swing.RoundSwingLedger
	cfg              swing.Config
	roundNumber      int
	enabled          bool
}

// NewSwingTrackerWithConfig creates a swing tracker with the given config.
func NewSwingTrackerWithConfig(cfg swing.Config) *SwingTracker {
	engine := probability.NewDefaultEngine()
	return &SwingTracker{
		engine:           engine,
		allocator:        swing.NewPoolAllocator(engine, cfg),
		damageTracker:    NewDamageTracker(),
		advantageTracker: NewAdvantageTracker(),
		cfg:              cfg,
		enabled:          true,
	}
}

// SetEnabled enables or disables swing tracking.
func (st *SwingTracker) SetEnabled(enabled bool) {
	st.enabled = enabled
}

// IsEnabled returns whether swing tracking is enabled.
func (st *SwingTracker) IsEnabled() bool {
	return st.enabled
}

// Config returns the current swing configuration.
func (st *SwingTracker) Config() swing.Config {
	return st.cfg
}

// ResetRound clears state for a new round.
func (st *SwingTracker) ResetRound(roundNumber, tAlive, ctAlive int, mapName string) {
	st.roundNumber = roundNumber
	st.roundState = probability.NewRoundState(tAlive, ctAlive, mapName)
	st.ledger = swing.NewRoundSwingLedger(roundNumber)
	st.damageTracker.Reset()
	st.advantageTracker.Reset()
}

// SetEconomyFromValues sets economy from equipment values.
func (st *SwingTracker) SetEconomyFromValues(tAvgEquip, ctAvgEquip float64) {
	if st.roundState != nil {
		st.roundState.TEconomy = probability.CategorizeEquipment(tAvgEquip)
		st.roundState.CTEconomy = probability.CategorizeEquipment(ctAvgEquip)
	}
}

// RecordDamage records damage dealt for attribution tracking.
func (st *SwingTracker) RecordDamage(attackerID, victimID uint64, damage int, timeInRound float64) {
	if !st.enabled {
		return
	}
	st.damageTracker.RecordDamage(attackerID, victimID, damage, timeInRound)
}

// RecordFlash records a flash for attribution tracking.
func (st *SwingTracker) RecordFlash(attackerID, victimID uint64, duration float64) {
	if !st.enabled {
		return
	}
	st.damageTracker.RecordFlash(attackerID, victimID, duration)
}

// KillResult wraps swing allocations for a kill event.
type KillResult struct {
	Swing                   swing.KillSwingResult
	Allocations             []swing.PlayerSwingAllocation
	SurvivalBeneficiaries   []uint64
	SurvivalCreditPerPlayer float64
	VictimPriorDamage       int
}

// RecordKill records a kill event and returns swing allocations.
func (st *SwingTracker) RecordKill(
	killerID, victimID uint64,
	killerSide, victimSide common.Team,
	killerEquip, victimEquip float64,
	timeInRound float64,
	isTradeKill, isHeadshot bool,
	roundDecided bool,
	losingSideTeammates []uint64,
) KillResult {
	if !st.enabled || st.roundState == nil {
		return KillResult{}
	}

	killEvent := &swing.KillEvent{
		TimeInRound:         timeInRound,
		KillerID:            killerID,
		VictimID:            victimID,
		KillerSide:          killerSide,
		VictimSide:          victimSide,
		KillerEquip:         killerEquip,
		VictimEquip:         victimEquip,
		IsTradeKill:         isTradeKill,
		IsHeadshot:          isHeadshot,
		TotalDamageToVictim: st.damageTracker.GetTotalDamageToVictim(victimID),
		KillerDamageDealt:   st.damageTracker.GetKillerDamage(killerID, victimID),
		DamageContributors:  st.damageTracker.GetDamageContributors(victimID),
		FlashAssists:        st.damageTracker.GetFlashAssists(victimID),
	}

	victimPriorDamage := st.damageTracker.GetTotalDamageToVictim(victimID) - st.damageTracker.GetKillerDamage(killerID, victimID)
	if victimPriorDamage < 0 {
		victimPriorDamage = 0
	}

	input := swing.KillAllocationInput{
		Kill:                killEvent,
		VictimPriorDamage:   victimPriorDamage,
		LosingSideTeammates: losingSideTeammates,
		ExitFrag:            roundDecided && st.cfg.ExitFrags == swing.ExitFragsZero,
		ExitFragScale:       st.exitFragScale(roundDecided),
	}

	event, swingResult := st.allocator.AllocateKillEvent(st.roundState, input)
	st.ledger.RecordEvent(event)
	allocations := append(event.PositiveAlloc, event.NegativeAlloc...)

	survivalBeneficiaries := st.advantageTracker.RecordKill(killerID, killerSide)
	st.advantageTracker.RecordDeath(victimID, victimSide)

	var survivalCredit float64
	if st.cfg.SurvivalCreditEnabled && len(survivalBeneficiaries) > 0 {
		survivalCredit = swingResult.RawSwing * SurvivalCreditShare
		if st.cfg.SurvivalCreditMaxShare > 0 {
			maxCredit := swingResult.RawSwing * st.cfg.SurvivalCreditMaxShare
			if survivalCredit > maxCredit {
				survivalCredit = maxCredit
			}
		}
	}

	st.roundState.RecordDeath(victimSide)
	st.damageTracker.ClearVictimData(victimID)

	return KillResult{
		Swing:                   swingResult,
		Allocations:             allocations,
		SurvivalBeneficiaries:   survivalBeneficiaries,
		SurvivalCreditPerPlayer: survivalCredit,
		VictimPriorDamage:       victimPriorDamage,
	}
}

func (st *SwingTracker) exitFragScale(roundDecided bool) float64 {
	if !roundDecided {
		return 1.0
	}
	if st.cfg.ExitFrags == swing.ExitFragsTiny {
		return 0.05
	}
	return 0
}

// ApplyTradeReallocation redistributes death penalty when a trade occurs.
func (st *SwingTracker) ApplyTradeReallocation(
	victimID uint64,
	victimSide common.Team,
	originalVictimPenalty float64,
	teammateIDs []uint64,
) []swing.PlayerSwingAllocation {
	if st.ledger == nil {
		return nil
	}
	allocs := st.allocator.AllocateTradeReallocation(victimID, victimSide, originalVictimPenalty, teammateIDs)
	if len(allocs) > 0 {
		st.ledger.ApplyAllocations(allocs)
	}
	return allocs
}

// RecordBombPlant records a bomb plant and returns per-player swing allocations.
func (st *SwingTracker) RecordBombPlant(
	planterID uint64,
	timeInRound float64,
	players map[uint64]swing.PlayerRoundContext,
) []swing.PlayerSwingAllocation {
	if !st.enabled || st.roundState == nil {
		return nil
	}

	input := swing.ObjectiveAllocationInput{
		PrimaryID:     planterID,
		PrimarySide:   common.TeamTerrorists,
		OpposingSide:  common.TeamCounterTerrorists,
		TimeInRound:   timeInRound,
		Players:       players,
		PrimaryShare:  st.cfg.PlantPlanterShare,
		PrimaryReason: swing.SwingReasonBombPlant,
		SupportReason: swing.SwingReasonPlantSupport,
	}
	event, _ := st.allocator.AllocateObjectiveEvent(
		st.roundState,
		common.TeamTerrorists,
		st.engine.CalculateBombPlantSwing,
		func(s *probability.RoundState) { s.SetBombPlanted() },
		input,
		swing.SwingEventBombPlant,
	)
	st.ledger.RecordEvent(event)
	return append(event.PositiveAlloc, event.NegativeAlloc...)
}

// RecordBombDefuse records a bomb defuse and returns per-player swing allocations.
func (st *SwingTracker) RecordBombDefuse(
	defuserID uint64,
	timeInRound float64,
	players map[uint64]swing.PlayerRoundContext,
) []swing.PlayerSwingAllocation {
	if !st.enabled || st.roundState == nil {
		return nil
	}

	input := swing.ObjectiveAllocationInput{
		PrimaryID:     defuserID,
		PrimarySide:   common.TeamCounterTerrorists,
		OpposingSide:  common.TeamTerrorists,
		TimeInRound:   timeInRound,
		Players:       players,
		PrimaryShare:  st.cfg.DefuseDefuserShare,
		PrimaryReason: swing.SwingReasonBombDefuse,
		SupportReason: swing.SwingReasonDefuseSupport,
	}
	event, _ := st.allocator.AllocateObjectiveEvent(
		st.roundState,
		common.TeamCounterTerrorists,
		st.engine.CalculateBombDefuseSwing,
		func(s *probability.RoundState) { s.SetBombDefused() },
		input,
		swing.SwingEventBombDefuse,
	)
	st.ledger.RecordEvent(event)
	return append(event.PositiveAlloc, event.NegativeAlloc...)
}

// ApplyRoundResidual applies end-of-round residual swing if enabled.
func (st *SwingTracker) ApplyRoundResidual(winner common.Team, players map[uint64]swing.PlayerRoundContext, maxDamage int) []swing.PlayerSwingAllocation {
	if !st.enabled || st.ledger == nil {
		return nil
	}
	currentProb := st.GetCurrentWinProbability(winner)
	return swing.ApplyRoundResidual(st.ledger, swing.ResidualInput{
		Winner:           winner,
		CurrentWinProb:   currentProb,
		Players:          players,
		MaxDamageInRound: maxDamage,
	}, st.cfg)
}

// FinalizeRound validates the ledger and returns debug warnings.
func (st *SwingTracker) FinalizeRound() []string {
	if st.ledger == nil {
		return nil
	}
	return swing.ValidateRoundSwingLedger(st.ledger, st.cfg.ZeroSumTolerance)
}

// GetCurrentWinProbability returns the current win probability for a side.
func (st *SwingTracker) GetCurrentWinProbability(side common.Team) float64 {
	if !st.enabled || st.roundState == nil {
		return 0.5
	}
	return st.engine.GetWinProbability(st.roundState, side)
}

// GetTimeToKill returns the time between first damage and kill for a specific kill.
func (st *SwingTracker) GetTimeToKill(killerID, victimID uint64, killTime float64) float64 {
	return st.damageTracker.GetTimeToKill(killerID, victimID, killTime)
}
