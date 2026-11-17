// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package rating implements the eco-rating calculation system.
// This file contains the core rating computation functions that combine
// various performance metrics into a final player rating.
package rating

import "math"

// exponentialAdjustment calculates an exponential adjustment capped at ±maxAdj.
// diff is the difference from baseline, positive means above baseline.
// k controls the curve steepness (higher = faster approach to cap).
func exponentialAdjustment(diff float64, maxAdj float64, k float64) float64 {
	sign := 1.0
	if diff < 0 {
		sign = -1.0
	}
	adj := sign * maxAdj * (1 - math.Exp(-k*math.Abs(diff)))
	return math.Max(-maxAdj, math.Min(maxAdj, adj))
}

// computeKPRDPRAdjustment calculates the combined KPR/DPR adjustment.
// Each is calculated independently with exponential scaling, range -0.2 to +0.2 total.
func computeKPRDPRAdjustment(kpr, dpr float64) float64 {
	kprAdj := exponentialAdjustment(kpr-BaselineKPR, 0.1, 5)
	dprAdj := exponentialAdjustment(BaselineDPR-dpr, 0.1, 5)
	return kprAdj + dprAdj
}

// computeContribution calculates a contribution based on value vs baseline with different multipliers.
func computeContribution(value, baseline, aboveMultiplier, belowMultiplier float64) float64 {
	if value >= baseline {
		return (value - baseline) * aboveMultiplier
	}
	return (value - baseline) * belowMultiplier
}

// ComputeSideRating calculates a rating for a specific side (T or CT).
// Pure probability-based rating:
// - ProbabilitySwing: Core metric measuring win probability impact
// - ADR: Rewards chip damage that didn't result in kills
// - KAST: Rewards round involvement
//
// Kills/deaths are captured entirely through swing to avoid double-counting.
func ComputeSideRating(rounds, kills, deaths, damage int, probabilitySwing, kast float64, kdprModifier bool) float64 {

	roundsF := float64(rounds)
	if roundsF == 0 {
		return 0
	}

	adr := float64(damage) / roundsF
	kastPct := kast / roundsF
	probSwingPerRound := probabilitySwing / roundsF

	var kprDprAdjustment float64
	if kdprModifier {
		kpr := float64(kills) / roundsF
		dpr := float64(deaths) / roundsF
		kprDprAdjustment = computeKPRDPRAdjustment(kpr, dpr)
	}

	adrContrib := computeContribution(adr, BaselineADR, ADRContribAbove, ADRContribBelow)
	kastContrib := computeContribution(kastPct, BaselineKAST, KASTContribAbove, KASTContribBelow)
	probSwingContrib := probSwingPerRound * ProbSwingContribMultiplier

	rating := RatingBaseline + adrContrib + kastContrib + probSwingContrib + kprDprAdjustment
	return math.Max(MinRating, math.Min(MaxRating, rating))
}
