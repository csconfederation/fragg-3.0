package eco

import (
	"github.com/csconfederation/fragg-3.0/rating"
	"github.com/csconfederation/fragg-3.0/rating/probability"
	"github.com/csconfederation/fragg-3.0/rating/swing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
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
	tickRate         float64
	lastClockTick    int
	bombTimer        float64
}

const (
	roundTimeSeconds        = 115.0
	defaultBombTimerSeconds = 40.0
)

// NewSwingTrackerWithConfig creates a swing tracker with the given config.
func NewSwingTrackerWithConfig(cfg swing.Config) *SwingTracker {
	engine := probability.NewDefaultEngine()
	return &SwingTracker{
		engine:           engine,
		allocator:        swing.NewPoolAllocator(engine, cfg),
		damageTracker:    NewDamageTracker(),
		advantageTracker: NewAdvantageTracker(),
		cfg:              cfg,
		tickRate:         float64(rating.TickRate),
		bombTimer:        defaultBombTimerSeconds,
	}
}

// ResetRound clears state for a new round and starts the round clock at startTick.
func (st *SwingTracker) ResetRound(tAlive, ctAlive int, mapName string, startTick int) {
	st.roundState = probability.NewRoundState(tAlive, ctAlive, mapName)
	st.roundState.SetTimeRemaining(roundTimeSeconds)
	st.lastClockTick = startTick
	st.ledger = swing.NewRoundSwingLedger()
	st.damageTracker.Reset()
	st.advantageTracker.Reset()
}

// advanceClock subtracts elapsed demo time from TimeRemaining.
func (st *SwingTracker) advanceClock(currentTick int) {
	if st.roundState == nil || st.tickRate <= 0 {
		return
	}
	if st.lastClockTick > 0 && currentTick > st.lastClockTick {
		elapsed := float64(currentTick-st.lastClockTick) / st.tickRate
		st.roundState.SetTimeRemaining(st.roundState.TimeRemaining - elapsed)
	}
	st.lastClockTick = currentTick
}

// SetAlive sets the alive counts the probabilities read; engine/round counts them (#591). Clamped
// to 0-5: a miscount (an alive join, which doesn't add to the counts) mustn't reach the table
// fallback's division by the total, whose NaN breaks the payload's JSON.
func (st *SwingTracker) SetAlive(tAlive, ctAlive int) {
	if st.roundState != nil {
		st.roundState.TAlive, st.roundState.CTAlive = min(max(tAlive, 0), 5), min(max(ctAlive, 0), 5)
	}
}

// RecordDeath is a counted death that isn't an enemy kill (a team kill, suicide or world death):
// the victim's advantage slots and damage go, as in RecordKill.
func (st *SwingTracker) RecordDeath(victimID uint64, victimSide fact.Side) {
	if st.roundState == nil {
		return
	}
	st.advantageTracker.RecordDeath(victimID, victimSide)
	st.damageTracker.ClearVictimData(victimID)
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
	st.damageTracker.RecordDamage(attackerID, victimID, damage, timeInRound)
}

// RecordFlash records a flash for attribution tracking.
func (st *SwingTracker) RecordFlash(attackerID, victimID uint64, duration float64, tick int) {
	st.damageTracker.RecordFlash(attackerID, victimID, duration, tick)
}

// KillResult wraps swing allocations for a kill event.
type KillResult struct {
	Swing                   swing.KillSwingResult
	Allocations             []swing.PlayerSwingAllocation
	SurvivalBeneficiaries   []uint64
	SurvivalCreditPerPlayer float64
}

// RecordKill records a kill event and returns swing allocations.
func (st *SwingTracker) RecordKill(
	killerID, victimID uint64,
	killerSide, victimSide fact.Side,
	killerEquip, victimEquip float64,
	isTradeKill bool,
	roundDecided bool,
	losingSideTeammates []uint64,
	currentTick int,
) KillResult {
	if st.roundState == nil {
		return KillResult{}
	}

	st.advanceClock(currentTick)

	killEvent := &swing.KillEvent{
		KillerID:            killerID,
		VictimID:            victimID,
		KillerSide:          killerSide,
		VictimSide:          victimSide,
		KillerEquip:         killerEquip,
		VictimEquip:         victimEquip,
		IsTradeKill:         isTradeKill,
		TotalDamageToVictim: st.damageTracker.GetTotalDamageToVictim(victimID),
		KillerDamageDealt:   st.damageTracker.GetKillerDamage(killerID, victimID),
		DamageContributors:  st.damageTracker.GetDamageContributors(victimID),
		FlashAssists:        st.damageTracker.GetFlashAssists(victimID, currentTick, st.tickRate),
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

	st.damageTracker.ClearVictimData(victimID)

	return KillResult{
		Swing:                   swingResult,
		Allocations:             allocations,
		SurvivalBeneficiaries:   survivalBeneficiaries,
		SurvivalCreditPerPlayer: survivalCredit,
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
	victimSide fact.Side,
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
	players map[uint64]swing.PlayerRoundContext,
	currentTick int,
) []swing.PlayerSwingAllocation {
	if st.roundState == nil {
		return nil
	}

	st.advanceClock(currentTick)

	input := swing.ObjectiveAllocationInput{
		PrimaryID:     planterID,
		PrimarySide:   fact.SideT,
		OpposingSide:  fact.SideCT,
		Players:       players,
		PrimaryShare:  st.cfg.PlantPlanterShare,
		PrimaryReason: swing.SwingReasonBombPlant,
		SupportReason: swing.SwingReasonPlantSupport,
	}
	event, _ := st.allocator.AllocateObjectiveEvent(
		st.roundState,
		st.engine.CalculateBombPlantSwing,
		func(s *probability.RoundState) {
			s.SetBombPlanted()
			s.SetTimeRemaining(st.bombTimer)
		},
		input,
	)
	st.ledger.RecordEvent(event)
	return append(event.PositiveAlloc, event.NegativeAlloc...)
}

// RecordBombDefuse records a bomb defuse and returns per-player swing allocations.
func (st *SwingTracker) RecordBombDefuse(
	defuserID uint64,
	players map[uint64]swing.PlayerRoundContext,
	currentTick int,
) []swing.PlayerSwingAllocation {
	if st.roundState == nil {
		return nil
	}

	st.advanceClock(currentTick)

	input := swing.ObjectiveAllocationInput{
		PrimaryID:     defuserID,
		PrimarySide:   fact.SideCT,
		OpposingSide:  fact.SideT,
		Players:       players,
		PrimaryShare:  st.cfg.DefuseDefuserShare,
		PrimaryReason: swing.SwingReasonBombDefuse,
		SupportReason: swing.SwingReasonDefuseSupport,
	}
	event, _ := st.allocator.AllocateObjectiveEvent(
		st.roundState,
		st.engine.CalculateBombDefuseSwing,
		func(s *probability.RoundState) { s.SetBombDefused() },
		input,
	)
	st.ledger.RecordEvent(event)
	return append(event.PositiveAlloc, event.NegativeAlloc...)
}

// ApplyRoundResidual applies end-of-round residual swing if enabled.
func (st *SwingTracker) ApplyRoundResidual(winner fact.Side, players map[uint64]swing.PlayerRoundContext, maxDamage int) []swing.PlayerSwingAllocation {
	if st.ledger == nil {
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
func (st *SwingTracker) GetCurrentWinProbability(side fact.Side) float64 {
	if st.roundState == nil {
		return 0.5
	}
	return st.engine.GetWinProbability(st.roundState, side)
}

// GetTimeToKill returns the time between first damage and kill for a specific kill.
func (st *SwingTracker) GetTimeToKill(killerID, victimID uint64, killTime float64) float64 {
	return st.damageTracker.GetTimeToKill(killerID, victimID, killTime)
}
