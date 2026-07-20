package probability

import (
	"fmt"
)

// ProbabilityTables holds all empirically-derived probability data.
type ProbabilityTables struct {
	// BaseWinProb maps state keys to T-side win probability.
	// Key format: "TvCT_bombStatus" (e.g., "5v4_none", "3v2_planted")
	BaseWinProb map[string]float64

	// DuelWinRates maps economy matchup keys to attacker win probability.
	// Key format: "attacker_vs_defender" (e.g., "awp_vs_pistol", "rifle_vs_smg")
	DuelWinRates map[string]float64

	// MapAdjustments maps map names to T-side win rate adjustments.
	// Value is the T-side win percentage (0.0-1.0) for that map.
	MapAdjustments map[string]float64
}

// NewProbabilityTables creates a new ProbabilityTables with empty maps.
func NewProbabilityTables() *ProbabilityTables {
	return &ProbabilityTables{
		BaseWinProb:    make(map[string]float64),
		DuelWinRates:   make(map[string]float64),
		MapAdjustments: make(map[string]float64),
	}
}

// GetBaseWinProbability returns the T-side win probability for a given state.
func (t *ProbabilityTables) GetBaseWinProbability(tAlive, ctAlive int, bombPlanted bool) float64 {
	bombStatus := "none"
	if bombPlanted {
		bombStatus = "planted"
	}
	key := stateKeyFromComponents(tAlive, ctAlive, bombStatus)

	if prob, ok := t.BaseWinProb[key]; ok {
		return prob
	}

	// Fallback: calculate based on player advantage
	return t.calculateFallbackProbability(tAlive, ctAlive, bombPlanted)
}

// calculateFallbackProbability provides a reasonable estimate when no empirical data exists.
func (t *ProbabilityTables) calculateFallbackProbability(tAlive, ctAlive int, bombPlanted bool) float64 {
	if tAlive == 0 {
		return 0.0
	}
	if ctAlive == 0 {
		return 1.0
	}

	// Base probability from player count ratio
	total := float64(tAlive + ctAlive)
	baseProb := float64(tAlive) / total

	// CT-side advantage in equal situations (CT wins ~52% of 5v5s)
	ctAdvantage := 0.04
	baseProb -= ctAdvantage * (float64(ctAlive) / 5.0)

	// Bomb planted heavily favors T
	if bombPlanted {
		baseProb += 0.25 * (1.0 - baseProb) // Move 25% closer to 1.0
	}

	return clamp(baseProb, 0.01, 0.99)
}

// GetDuelWinRate returns the probability that the attacker wins a duel.
func (t *ProbabilityTables) GetDuelWinRate(attackerEcon, defenderEcon EconomyCategory) float64 {
	key := fmt.Sprintf("%s_vs_%s", attackerEcon.String(), defenderEcon.String())

	if rate, ok := t.DuelWinRates[key]; ok {
		return rate
	}

	// Fallback: calculate based on economy difference
	return t.calculateFallbackDuelRate(attackerEcon, defenderEcon)
}

// calculateFallbackDuelRate provides a reasonable estimate for duel outcomes.
func (t *ProbabilityTables) calculateFallbackDuelRate(attackerEcon, defenderEcon EconomyCategory) float64 {
	diff := int(attackerEcon) - int(defenderEcon)

	// Each economy tier is worth ~7-8% advantage
	adjustment := float64(diff) * 0.07

	return clamp(0.50+adjustment, 0.20, 0.80)
}

// GetMapAdjustment returns the T-side win rate for a map (default 0.50).
func (t *ProbabilityTables) GetMapAdjustment(mapName string) float64 {
	if adj, ok := t.MapAdjustments[mapName]; ok {
		return adj
	}
	return 0.50 // Default balanced
}

// clamp restricts a value to the range [min, max].
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
