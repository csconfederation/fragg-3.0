package parser

import (
	"eco-rating/rating/probability"
	"eco-rating/rating/swing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// SwingTracker manages round state and swing calculation during parsing.
type SwingTracker struct {
	calculator       *swing.Calculator
	damageTracker    *DamageTracker
	advantageTracker *AdvantageTracker
	roundState       *probability.RoundState
	roundEvents      []swing.RoundEvent
	enabled          bool
}

// NewSwingTracker creates a new swing tracker.
func NewSwingTracker() *SwingTracker {
	return &SwingTracker{
		calculator:       swing.NewDefaultCalculator(),
		damageTracker:    NewDamageTracker(),
		advantageTracker: NewAdvantageTracker(),
		roundEvents:      make([]swing.RoundEvent, 0),
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

// ResetRound clears state for a new round.
func (st *SwingTracker) ResetRound(tAlive, ctAlive int, mapName string) {
	st.roundState = probability.NewRoundState(tAlive, ctAlive, mapName)
	st.roundEvents = make([]swing.RoundEvent, 0)
	st.damageTracker.Reset()
	st.advantageTracker.Reset()
}

// SetEconomy sets the economy categories for both teams.
func (st *SwingTracker) SetEconomy(tEcon, ctEcon probability.EconomyCategory) {
	if st.roundState != nil {
		st.roundState.TEconomy = tEcon
		st.roundState.CTEconomy = ctEcon
	}
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

// KillResult wraps the swing result with survival credit information.
type KillResult struct {
	Swing                   swing.KillSwingResult
	SurvivalBeneficiaries   []uint64 // Players who earn survival credit from this kill
	SurvivalCreditPerPlayer float64  // Amount of survival credit each beneficiary earns
	VictimPriorDamage       int      // Total damage victim took before the killing blow
}

// RecordKill records a kill event and returns economy-adjusted swing values
// along with survival credit information for advantage creators.
func (st *SwingTracker) RecordKill(
	killerID, victimID uint64,
	killerSide, victimSide common.Team,
	killerEquip, victimEquip float64,
	timeInRound float64,
	isTradeKill, isHeadshot bool,
) KillResult {
	if !st.enabled || st.roundState == nil {
		return KillResult{}
	}

	// Build kill event
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

	// Calculate economy-adjusted swing before updating state
	swingResult := st.calculator.CalculateKillSwingWithEconomy(st.roundState, killEvent)

	// Track man advantages: get survival beneficiaries BEFORE adding new slot
	survivalBeneficiaries := st.advantageTracker.RecordKill(killerID, killerSide)

	// Record the victim's death in the advantage tracker
	// (neutralizes one advantage slot on the victim's team)
	st.advantageTracker.RecordDeath(victimID, victimSide)

	// Calculate survival credit per beneficiary
	var survivalCredit float64
	if len(survivalBeneficiaries) > 0 {
		survivalCredit = swingResult.RawSwing * SurvivalCreditShare
	}

	// Add event to round events
	st.roundEvents = append(st.roundEvents, killEvent)

	// Update state
	st.roundState.RecordDeath(victimSide)

	// Capture victim's prior damage from OTHER players (excluding the killer's damage).
	// This represents damage taken before the killing blow, used for death penalty reduction.
	// We exclude the killer's damage because it includes the killing blow itself.
	victimPriorDamage := st.damageTracker.GetTotalDamageToVictim(victimID) - st.damageTracker.GetKillerDamage(killerID, victimID)
	if victimPriorDamage < 0 {
		victimPriorDamage = 0
	}

	// Clear victim's damage tracking data
	st.damageTracker.ClearVictimData(victimID)

	return KillResult{
		Swing:                   swingResult,
		SurvivalBeneficiaries:   survivalBeneficiaries,
		SurvivalCreditPerPlayer: survivalCredit,
		VictimPriorDamage:       victimPriorDamage,
	}
}

// GetDamageToPlayer returns the total damage dealt to a player this round.
// Used to estimate victim health at death time for death penalty reduction.
func (st *SwingTracker) GetDamageToPlayer(playerID uint64) int {
	if st.damageTracker == nil {
		return 0
	}
	return st.damageTracker.GetTotalDamageToVictim(playerID)
}

// RecordBombPlant records a bomb plant event.
func (st *SwingTracker) RecordBombPlant(planterID uint64, timeInRound float64) float64 {
	if !st.enabled || st.roundState == nil {
		return 0
	}

	// Calculate swing before updating state
	engine := st.calculator.GetProbabilityEngine()
	swingValue := engine.CalculateBombPlantSwing(st.roundState)

	// Add event
	plantEvent := &swing.BombPlantEvent{
		TimeInRound: timeInRound,
		PlanterID:   planterID,
	}
	st.roundEvents = append(st.roundEvents, plantEvent)

	// Update state
	st.roundState.SetBombPlanted()

	return swingValue
}

// RecordBombDefuse records a bomb defuse event.
func (st *SwingTracker) RecordBombDefuse(defuserID uint64, timeInRound float64) float64 {
	if !st.enabled || st.roundState == nil {
		return 0
	}

	// Calculate swing before updating state
	engine := st.calculator.GetProbabilityEngine()
	swingValue := engine.CalculateBombDefuseSwing(st.roundState)

	// Add event
	defuseEvent := &swing.BombDefuseEvent{
		TimeInRound: timeInRound,
		DefuserID:   defuserID,
	}
	st.roundEvents = append(st.roundEvents, defuseEvent)

	// Update state
	st.roundState.SetBombDefused()

	return swingValue
}

// RecordBombExplode records a bomb explosion.
func (st *SwingTracker) RecordBombExplode(timeInRound float64) {
	if !st.enabled || st.roundState == nil {
		return
	}

	explodeEvent := &swing.BombExplodeEvent{
		TimeInRound: timeInRound,
	}
	st.roundEvents = append(st.roundEvents, explodeEvent)
}

// GetCurrentState returns the current round state.
func (st *SwingTracker) GetCurrentState() *probability.RoundState {
	if st.roundState == nil {
		return nil
	}
	return st.roundState.Clone()
}

// GetCurrentWinProbability returns the current T-side win probability.
func (st *SwingTracker) GetCurrentWinProbability(side common.Team) float64 {
	if !st.enabled || st.roundState == nil {
		return 0.5
	}
	return st.calculator.GetProbabilityEngine().GetWinProbability(st.roundState, side)
}

// CalculateRoundSwings calculates final swing values for all players.
func (st *SwingTracker) CalculateRoundSwings(
	initialState *probability.RoundState,
	result *swing.RoundResult,
) map[uint64]float64 {
	if !st.enabled || initialState == nil {
		return make(map[uint64]float64)
	}

	return st.calculator.CalculateRoundSwing(st.roundEvents, initialState, result).PlayerSwings
}

// GetRoundEvents returns the events recorded this round.
func (st *SwingTracker) GetRoundEvents() []swing.RoundEvent {
	return st.roundEvents
}

// GetDamageTracker returns the damage tracker for direct access if needed.
func (st *SwingTracker) GetDamageTracker() *DamageTracker {
	return st.damageTracker
}

// GetTimeToKill returns the time between first damage and kill for a specific kill.
func (st *SwingTracker) GetTimeToKill(killerID, victimID uint64, killTime float64) float64 {
	return st.damageTracker.GetTimeToKill(killerID, victimID, killTime)
}

// GetCalculator returns the swing calculator.
func (st *SwingTracker) GetCalculator() *swing.Calculator {
	return st.calculator
}
