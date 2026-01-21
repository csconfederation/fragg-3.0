// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package rating implements the eco-rating calculation system.
// This file contains the core rating computation functions that combine
// various performance metrics into a final player rating.
package rating

import (
	"eco-rating/model"
	"math"
)

// ComputeFinalRating calculates the overall eco-rating for a player.
// Pure probability-based rating (HLTV 3.0 style):
// - ProbabilitySwing: Core metric measuring win probability impact of all actions
// - ADR: Rewards chip damage that didn't result in kills
// - KAST: Rewards round involvement (kill/assist/survive/trade)
//
// Kills/deaths are captured entirely through ProbabilitySwing to avoid double-counting.
// Returns a value typically between 0.20 and 3.00.
func ComputeFinalRating(p *model.PlayerStats) float64 {
	rounds := float64(p.RoundsPlayed)
	if rounds == 0 {
		return 0
	}

	adr := float64(p.Damage) / rounds
	kast := p.KAST

	// ADR contribution - rewards consistent damage output
	var adrContrib float64
	if adr >= BaselineADR {
		adrContrib = (adr - BaselineADR) * ADRContribAbove
	} else {
		adrContrib = (adr - BaselineADR) * ADRContribBelow
	}

	// KAST contribution - rewards round involvement
	var kastContrib float64
	if kast >= BaselineKAST {
		kastContrib = (kast - BaselineKAST) * KASTContribAbove
	} else {
		kastContrib = (kast - BaselineKAST) * KASTContribBelow
	}

	// Probability-based swing contribution (core metric)
	// Now includes both positive swing from kills AND negative swing from deaths
	probSwingContrib := p.ProbabilitySwingPerRound * ProbSwingContribMultiplier

	rating := RatingBaseline + adrContrib + kastContrib + probSwingContrib

	return math.Max(MinRating, math.Min(MaxRating, rating))
}

// sumMulti calculates a weighted sum of multi-kill rounds.
// Higher kill counts receive exponentially higher weights.
func sumMulti(m [6]int) int {
	weights := [6]int{0, 0, 2, 6, 14, 30}
	total := 0
	for i := 2; i <= 5; i++ {
		total += m[i] * weights[i]
	}
	return total
}

// ComputeSideRating calculates a rating for a specific side (T or CT).
// Pure probability-based rating matching ComputeFinalRating:
// - ProbabilitySwing: Core metric measuring win probability impact
// - ADR: Rewards chip damage that didn't result in kills
// - KAST: Rewards round involvement
//
// Kills/deaths are captured entirely through swing to avoid double-counting.
func ComputeSideRating(rounds int, kills int, deaths int, damage int, ecoKillValue float64,
	probabilitySwing float64, kast float64, multiKills [6]int, clutchRounds int, clutchWins int) float64 {

	roundsF := float64(rounds)
	if roundsF == 0 {
		return 0
	}

	adr := float64(damage) / roundsF
	kastPct := kast / roundsF
	probSwingPerRound := probabilitySwing / roundsF

	// ADR contribution - rewards consistent damage output
	var adrContrib float64
	if adr >= BaselineADR {
		adrContrib = (adr - BaselineADR) * ADRContribAbove
	} else {
		adrContrib = (adr - BaselineADR) * ADRContribBelow
	}

	// KAST contribution - rewards round involvement
	var kastContrib float64
	if kastPct >= BaselineKAST {
		kastContrib = (kastPct - BaselineKAST) * KASTContribAbove
	} else {
		kastContrib = (kastPct - BaselineKAST) * KASTContribBelow
	}

	// Probability-based swing contribution (core metric)
	probSwingContrib := probSwingPerRound * ProbSwingContribMultiplier

	rating := RatingBaseline + adrContrib + kastContrib + probSwingContrib

	return math.Max(MinRating, math.Min(MaxRating, rating))
}
