package probability

import "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"

// Engine calculates win probabilities based on game state.
type Engine struct {
	tables *ProbabilityTables
}

// NewEngine creates a new probability engine with the given tables.
func NewEngine(tables *ProbabilityTables) *Engine {
	return &Engine{tables: tables}
}

// NewDefaultEngine creates a probability engine with default tables.
func NewDefaultEngine() *Engine {
	return NewEngine(DefaultTables())
}

// GetWinProbability returns the probability that the specified side wins the round.
func (e *Engine) GetWinProbability(state *RoundState, side common.Team) float64 {
	// Get base probability (T-side win rate)
	tWinProb := e.getBaseProbability(state)

	// Apply economy adjustment
	tWinProb = e.applyEconomyAdjustment(tWinProb, state)

	// Apply map adjustment
	tWinProb = e.applyMapAdjustment(tWinProb, state.Map)

	// Apply time adjustment for bomb planted scenarios
	tWinProb = e.applyTimeAdjustment(tWinProb, state)

	// Clamp to valid range
	tWinProb = clamp(tWinProb, 0.01, 0.99)

	if side == common.TeamTerrorists {
		return tWinProb
	}
	return 1.0 - tWinProb
}

// getBaseProbability returns the T-side win probability from the lookup tables.
func (e *Engine) getBaseProbability(state *RoundState) float64 {
	return e.tables.GetBaseWinProbability(state.TAlive, state.CTAlive, state.BombPlanted)
}

// applyEconomyAdjustment modifies probability based on economy differential.
func (e *Engine) applyEconomyAdjustment(baseProb float64, state *RoundState) float64 {
	// Calculate economy differential
	tEconLevel := int(state.TEconomy)
	ctEconLevel := int(state.CTEconomy)
	econDiff := tEconLevel - ctEconLevel

	// Economy adjustment multiplier
	// Each tier difference is worth about 3-5% advantage
	adjustments := map[int]float64{
		4:  1.20, // T has AWP, CT has Starter
		3:  1.15, // T has Rifle, CT has Starter
		2:  1.10, // Two tier advantage
		1:  1.05, // One tier advantage
		0:  1.00, // Equal economy
		-1: 0.95, // One tier disadvantage
		-2: 0.90, // Two tier disadvantage
		-3: 0.85, // T has Starter, CT has Rifle
		-4: 0.80, // T has Starter, CT has AWP
	}

	multiplier := 1.0
	if adj, ok := adjustments[clampInt(econDiff, -4, 4)]; ok {
		multiplier = adj
	}

	return baseProb * multiplier
}

// applyMapAdjustment modifies probability based on map T/CT balance.
func (e *Engine) applyMapAdjustment(baseProb float64, mapName string) float64 {
	mapTWinRate := e.tables.GetMapAdjustment(mapName)

	// Adjust relative to the default 50% baseline
	// If map is T-sided (e.g., 0.52), boost T probability slightly
	mapFactor := mapTWinRate / 0.50

	return baseProb * mapFactor
}

// applyTimeAdjustment modifies probability based on time remaining (bomb scenarios).
func (e *Engine) applyTimeAdjustment(baseProb float64, state *RoundState) float64 {
	if !state.BombPlanted {
		return baseProb
	}

	// Bomb timer is 40 seconds, defuse is 5s (with kit) or 10s (without)
	// As time runs down, T advantage increases

	if state.TimeRemaining <= 5.0 {
		// Very low time - T heavily favored unless CT is already defusing
		return baseProb * 1.15
	} else if state.TimeRemaining <= 10.0 {
		// Low time - T advantage increases
		return baseProb * 1.08
	} else if state.TimeRemaining <= 20.0 {
		// Medium time - slight T advantage
		return baseProb * 1.03
	}

	return baseProb
}

// GetDuelWinRate returns the probability that the attacker wins a duel.
func (e *Engine) GetDuelWinRate(attackerEquip, victimEquip float64) float64 {
	attackerCat := CategorizeEquipment(attackerEquip)
	victimCat := CategorizeEquipment(victimEquip)

	return e.tables.GetDuelWinRate(attackerCat, victimCat)
}

// GetDuelWinRateByCategory returns the duel win rate using economy categories.
func (e *Engine) GetDuelWinRateByCategory(attackerCat, victimCat EconomyCategory) float64 {
	return e.tables.GetDuelWinRate(attackerCat, victimCat)
}

// CalculateKillSwing calculates the probability swing from a kill.
func (e *Engine) CalculateKillSwing(stateBefore, stateAfter *RoundState, killerSide common.Team) float64 {
	probBefore := e.GetWinProbability(stateBefore, killerSide)
	probAfter := e.GetWinProbability(stateAfter, killerSide)

	return probAfter - probBefore
}

// CalculateBombPlantSwing calculates the probability swing from a bomb plant.
func (e *Engine) CalculateBombPlantSwing(stateBefore *RoundState) float64 {
	stateAfter := stateBefore.Clone()
	stateAfter.SetBombPlanted()

	probBefore := e.GetWinProbability(stateBefore, common.TeamTerrorists)
	probAfter := e.GetWinProbability(stateAfter, common.TeamTerrorists)

	return probAfter - probBefore
}

// CalculateBombDefuseSwing calculates the probability swing from a bomb defuse.
func (e *Engine) CalculateBombDefuseSwing(stateBefore *RoundState) float64 {
	stateAfter := stateBefore.Clone()
	stateAfter.SetBombDefused()

	probBefore := e.GetWinProbability(stateBefore, common.TeamCounterTerrorists)
	probAfter := e.GetWinProbability(stateAfter, common.TeamCounterTerrorists)

	return probAfter - probBefore
}

// GetEconomyAdjustedKillValue returns a multiplier for kill value based on economy.
// Kills against better-equipped opponents are worth more (>1.0).
// Kills against worse-equipped opponents are worth less (<1.0).
func (e *Engine) GetEconomyAdjustedKillValue(killerEquip, victimEquip float64) float64 {
	duelWinRate := e.GetDuelWinRate(killerEquip, victimEquip)

	// If you're expected to win the duel (high win rate), the kill is worth less.
	// If you're expected to lose (low win rate), the kill is worth more.
	// Base value at 50% win rate = 1.0
	// At 75% win rate (easy kill) = 0.67
	// At 25% win rate (hard kill) = 1.50

	if duelWinRate <= 0.01 {
		return 2.0 // Cap at 2x for extreme underdog
	}

	return 0.50 / duelWinRate
}

// clampInt restricts an integer value to the range [min, max].
func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
