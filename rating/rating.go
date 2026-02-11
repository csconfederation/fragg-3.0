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
	"fmt"
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
		p.RatingBreakdown = model.RatingBreakdown{Baseline: RatingBaseline, FinalRating: 0}
		return 0
	}

	adr := float64(p.Damage) / rounds
	kast := p.KAST
	probSwingPerRound := p.ProbabilitySwingPerRound

	// ADR contribution - rewards consistent damage output
	var adrContrib float64
	adrMultiplier := ADRContribBelow
	adrNotes := "ADR below baseline"
	if adr >= BaselineADR {
		adrMultiplier = ADRContribAbove
		adrNotes = "ADR above baseline"
	}
	adrContrib = (adr - BaselineADR) * adrMultiplier

	// KAST contribution - rewards round involvement
	var kastContrib float64
	kastMultiplier := KASTContribBelow
	kastNotes := "KAST below baseline"
	if kast >= BaselineKAST {
		kastMultiplier = KASTContribAbove
		kastNotes = "KAST above baseline"
	}
	kastContrib = (kast - BaselineKAST) * kastMultiplier

	// Probability-based swing contribution (core metric)
	// Now includes both positive swing from kills AND negative swing from deaths
	probSwingContrib := probSwingPerRound * ProbSwingContribMultiplier

	rating := RatingBaseline + adrContrib + kastContrib + probSwingContrib
	clamped := math.Max(MinRating, math.Min(MaxRating, rating))

	p.RatingBreakdown = model.RatingBreakdown{
		Baseline: RatingBaseline,
		ADR: model.RatingComponent{
			Metric:       "adr",
			Value:        adr,
			Baseline:     BaselineADR,
			Multiplier:   adrMultiplier,
			Contribution: adrContrib,
			Notes:        adrNotes,
		},
		KAST: model.RatingComponent{
			Metric:       "kast",
			Value:        kast,
			Baseline:     BaselineKAST,
			Multiplier:   kastMultiplier,
			Contribution: kastContrib,
			Notes:        kastNotes,
		},
		ProbabilitySwing: model.RatingComponent{
			Metric:       "probability_swing_per_round",
			Value:        probSwingPerRound,
			Multiplier:   ProbSwingContribMultiplier,
			Contribution: probSwingContrib,
			Notes:        fmt.Sprintf("Total swing: %.4f", p.ProbabilitySwing),
		},
		UnclampedRating: rating,
		FinalRating:     clamped,
		Formula: fmt.Sprintf(
			"rating = %.2f + (%.2f-%.2f)*%.3f + (%.2f-%.2f)*%.2f + (%.4f*%.2f)",
			RatingBaseline,
			adr, BaselineADR, adrMultiplier,
			kast, BaselineKAST, kastMultiplier,
			probSwingPerRound, ProbSwingContribMultiplier,
		),
	}

	return clamped
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
