package swing

import (
	"eco-rating/rating/probability"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// Calculator computes probability-based round swing for players.
type Calculator struct {
	probEngine *probability.Engine
	attrib     *Attributor
}

// NewCalculator creates a new swing calculator with the given probability engine.
func NewCalculator(engine *probability.Engine) *Calculator {
	return &Calculator{
		probEngine: engine,
		attrib:     NewAttributor(),
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

// CalculateRoundSwing computes swing for all players based on round events.
func (c *Calculator) CalculateRoundSwing(
	events []RoundEvent,
	initialState *probability.RoundState,
	result *RoundResult,
) *RoundSwingResult {
	playerSwing := make(map[uint64]float64)
	state := initialState.Clone()

	// Process each event in order
	for _, event := range events {
		switch e := event.(type) {
		case *KillEvent:
			c.processKill(playerSwing, state, e)
		case *BombPlantEvent:
			c.processBombPlant(playerSwing, state, e)
		case *BombDefuseEvent:
			c.processBombDefuse(playerSwing, state, e)
		case *BombExplodeEvent:
			c.processBombExplode(playerSwing, state)
		}
	}

	// Process round end (saves, final state)
	c.processRoundEnd(playerSwing, state, result)

	// Calculate totals
	res := &RoundSwingResult{
		PlayerSwings: playerSwing,
	}

	return res
}

// processKill handles a kill event and attributes swing to contributors.
func (c *Calculator) processKill(
	playerSwing map[uint64]float64,
	state *probability.RoundState,
	kill *KillEvent,
) {
	// Get probability before kill
	probBefore := c.probEngine.GetWinProbability(state, kill.KillerSide)

	// Update state
	state.RecordDeath(kill.VictimSide)

	// Get probability after kill
	probAfter := c.probEngine.GetWinProbability(state, kill.KillerSide)

	// Calculate raw delta (kills should always help the killer's side)
	rawDelta := probAfter - probBefore
	if rawDelta < 0 {
		rawDelta = 0
	}

	// Apply economy adjustment - harder kills are worth more
	duelWinRate := c.probEngine.GetDuelWinRate(kill.KillerEquip, kill.VictimEquip)
	ecoMultiplier := c.getEconomyMultiplier(duelWinRate)
	adjustedDelta := rawDelta * ecoMultiplier

	// Attribute credit to contributors
	c.attrib.AttributeKillCredit(playerSwing, kill, adjustedDelta)

	// Give negative swing to victim's team (implied by probability shift)
	// This is already captured in the probability - loser's side probability dropped
}

// getEconomyMultiplier returns a multiplier based on duel difficulty.
// Kills in unfavorable duels (low win rate) get bonus.
// Kills in favorable duels (high win rate) get penalty.
func (c *Calculator) getEconomyMultiplier(duelWinRate float64) float64 {
	// Baseline: 50% duel = 1.0 multiplier
	// 75% duel (easy) = ~0.67 multiplier
	// 25% duel (hard) = ~1.5 multiplier
	if duelWinRate <= 0.01 {
		return 2.0
	}
	return 0.50 / duelWinRate
}

// processBombPlant handles a bomb plant event.
func (c *Calculator) processBombPlant(
	playerSwing map[uint64]float64,
	state *probability.RoundState,
	plant *BombPlantEvent,
) {
	// Get probability before plant
	probBefore := c.probEngine.GetWinProbability(state, common.TeamTerrorists)

	// Update state
	state.SetBombPlanted()

	// Get probability after plant
	probAfter := c.probEngine.GetWinProbability(state, common.TeamTerrorists)

	// Calculate delta
	delta := probAfter - probBefore

	// Planter gets majority credit for plant but cap the swing contribution
	planterSwing := delta * PlantCreditShare
	if planterSwing > MaxPlantSwing {
		planterSwing = MaxPlantSwing
	} else if planterSwing < -MaxPlantSwing {
		planterSwing = -MaxPlantSwing
	}
	playerSwing[plant.PlanterID] += planterSwing
}

// processBombDefuse handles a bomb defuse event.
func (c *Calculator) processBombDefuse(
	playerSwing map[uint64]float64,
	state *probability.RoundState,
	defuse *BombDefuseEvent,
) {
	// Get probability before defuse
	probBefore := c.probEngine.GetWinProbability(state, common.TeamCounterTerrorists)

	// Update state
	state.SetBombDefused()

	// Get probability after defuse
	probAfter := c.probEngine.GetWinProbability(state, common.TeamCounterTerrorists)

	// Calculate delta
	delta := probAfter - probBefore

	// Defuser gets majority credit
	playerSwing[defuse.DefuserID] += delta * DefuseCreditShare
}

// processBombExplode handles bomb explosion (no individual credit, T team wins).
func (c *Calculator) processBombExplode(
	playerSwing map[uint64]float64,
	state *probability.RoundState,
) {
	// Bomb exploding is already factored into probability via BombPlanted state
	// and time remaining adjustments. No additional swing attribution needed.
}

// processRoundEnd handles end-of-round swing adjustments.
func (c *Calculator) processRoundEnd(
	playerSwing map[uint64]float64,
	state *probability.RoundState,
	result *RoundResult,
) {
	// Handle saves - when players survive a lost round
	if result != nil && len(result.Survivors) > 0 {
		if result.SurvivorSide != result.Winner {
			// Players saved on the losing team
			// Their "reward" is the saved weapon, not extra swing
			// Apply small penalty for not dying but losing
			for _, survivorID := range result.Survivors {
				playerSwing[survivorID] -= SavePenalty
			}
		}
	}
}

// KillSwingResult contains the economy-adjusted swing values for killer and victim.
type KillSwingResult struct {
	RawSwing          float64            // Raw probability delta from the kill
	KillerSwing       float64            // Economy-adjusted swing for the killer (after sharing with contributors)
	VictimSwing       float64            // Economy-adjusted penalty for the victim (worse for embarrassing deaths)
	EcoMultiplier     float64            // The economy multiplier applied
	ContributorSwings map[uint64]float64 // Swing credited to damage/flash assisters (playerID -> amount)
}

// CalculateSingleKillSwing computes the swing for a single kill event.
// Useful for real-time swing calculation during parsing.
// Returns the raw probability delta (no economy adjustment).
func (c *Calculator) CalculateSingleKillSwing(
	state *probability.RoundState,
	kill *KillEvent,
) float64 {
	stateBefore := state.Clone()

	// Get probability before
	probBefore := c.probEngine.GetWinProbability(stateBefore, kill.KillerSide)

	// Update state for after
	stateAfter := stateBefore.Clone()
	stateAfter.RecordDeath(kill.VictimSide)

	// Get probability after
	probAfter := c.probEngine.GetWinProbability(stateAfter, kill.KillerSide)

	// A kill should never reduce the killer's team win probability
	rawSwing := probAfter - probBefore
	if rawSwing < 0 {
		rawSwing = 0
	}
	return rawSwing
}

// CalculateKillSwingWithEconomy computes economy-adjusted swing for both killer and victim.
// - Killer gets bonus for hard kills (pistol vs rifle = 2x multiplier)
// - Victim gets extra penalty for embarrassing deaths (rifle dying to pistol = 2x penalty)
// - Damage contributors and flash assisters receive a share of the killer's swing
func (c *Calculator) CalculateKillSwingWithEconomy(
	state *probability.RoundState,
	kill *KillEvent,
) KillSwingResult {
	// Get raw probability swing
	rawSwing := c.CalculateSingleKillSwing(state, kill)

	// Get duel win rate from killer's perspective
	duelWinRate := c.probEngine.GetDuelWinRate(kill.KillerEquip, kill.VictimEquip)

	// Economy multiplier for killer (hard kills = bonus)
	killerEcoMult := c.getEconomyMultiplier(duelWinRate)

	// Economy multiplier for victim (embarrassing deaths = extra penalty)
	// If killer had low win rate, victim was favored - dying is embarrassing
	// Use victim's win rate / 0.50: high victim win rate → higher penalty
	victimWinRate := 1.0 - duelWinRate
	if victimWinRate < 0.01 {
		victimWinRate = 0.01
	}
	victimEcoMult := victimWinRate / 0.50
	if victimEcoMult > 2.0 {
		victimEcoMult = 2.0
	}

	// Total eco-adjusted swing to distribute among killer + contributors
	totalKillerSideSwing := rawSwing * killerEcoMult

	// Use the attributor to split credit among killer, damage contributors, and flash assisters
	playerSwings := make(map[uint64]float64)
	c.attrib.AttributeKillCredit(playerSwings, kill, totalKillerSideSwing)

	// Extract killer's share and contributor shares
	killerSwing := playerSwings[kill.KillerID]
	delete(playerSwings, kill.KillerID)

	// playerSwings now contains only contributor shares (assisters)
	var contributorSwings map[uint64]float64
	if len(playerSwings) > 0 {
		contributorSwings = playerSwings
	}

	return KillSwingResult{
		RawSwing:          rawSwing,
		KillerSwing:       killerSwing,
		VictimSwing:       rawSwing * victimEcoMult,
		EcoMultiplier:     killerEcoMult,
		ContributorSwings: contributorSwings,
	}
}

// GetProbabilityEngine returns the underlying probability engine.
func (c *Calculator) GetProbabilityEngine() *probability.Engine {
	return c.probEngine
}
