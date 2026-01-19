package rating

import (
	"eco-rating/model"
	"math"
)

// ComputeFinalRating calculates a rating using a linear combination approach
// inspired by HLTV 3.0's use of round swing, economy-adjusted stats, and round context.
func ComputeFinalRating(p *model.PlayerStats) float64 {
	rounds := float64(p.RoundsPlayed)
	if rounds == 0 {
		return 0
	}

	// ==================== CORE STATS ====================
	kpr := float64(p.Kills) / rounds
	dpr := float64(p.Deaths) / rounds
	adr := float64(p.Damage) / rounds
	kast := p.KAST
	avgSwing := p.RoundSwing / rounds

	// HLTV-style Impact: opening kills + multi-kill rounds
	openingKillsPerRound := float64(p.OpeningKills) / rounds
	multiKillRoundsPerRound := float64(p.RoundsWithMultiKill) / rounds

	// ==================== HLTV 3.0 INSPIRED FORMULA ====================
	// Calibrated to produce ratings in the 0.25 to 2.30 range
	// Based on analysis: winners are ~1.1-2.3, losers are ~0.25-0.90

	// KPR contribution: 0.7 KPR = baseline
	// Asymmetric: high KPR rewarded more to boost top performers like TeSeS
	var kprContrib float64
	if kpr >= 0.7 {
		kprContrib = (kpr - 0.7) * 0.75
	} else {
		kprContrib = (kpr - 0.7) * 0.55
	}

	// DPR contribution: 0.7 DPR = baseline
	// Asymmetric: penalize high deaths more to differentiate bottom players
	var dprContrib float64
	if dpr <= 0.7 {
		// Low deaths: moderate reward
		dprContrib = (0.7 - dpr) * 0.15
	} else {
		// High deaths: stronger penalty to lower bottom players
		dprContrib = (0.7 - dpr) * 0.55
	}

	// ADR contribution: 75 ADR = baseline
	// Asymmetric: high ADR rewarded more, low ADR penalized less
	var adrContrib float64
	if adr >= 75.0 {
		adrContrib = (adr - 75.0) * 0.015
	} else {
		adrContrib = (adr - 75.0) * 0.004
	}

	// KAST contribution: 70% KAST = baseline
	// Asymmetric: low KAST penalized more to differentiate bottom players
	var kastContrib float64
	if kast >= 0.70 {
		kastContrib = (kast - 0.70) * 0.20
	} else {
		kastContrib = (kast - 0.70) * 0.35
	}

	// Round Swing contribution: context-aware performance metric
	// Asymmetric: positive swing rewarded more, negative swing penalized more
	var swingContrib float64
	if avgSwing >= 0 {
		swingContrib = avgSwing * 0.75
	} else {
		swingContrib = avgSwing * 1.00
	}

	// Impact contribution: opening kills and multi-kill rounds
	// Reduced coefficients to avoid over-boosting losers with opening kills
	impactContrib := openingKillsPerRound*0.3 + multiKillRoundsPerRound*0.15

	// Multi-kill bonus: exponential scaling for 3k, 4k, 5k
	multiKillBonus := float64(sumMulti(p.MultiKillsRaw)) / rounds
	multiContrib := multiKillBonus * 0.015

	// ==================== COMBINE ====================
	// Base rating of 1.0 (average performance)
	rating := 1.0 + kprContrib + dprContrib + adrContrib + kastContrib + swingContrib + impactContrib + multiContrib

	// Clamp to reasonable range
	return math.Max(MinRating, math.Min(MaxRating, rating))
}

func sumMulti(m [6]int) int {
	// Weight multi-kills with exponential scaling like HLTV 3.0
	// Higher kill counts are exponentially more valuable
	// Doubled weights for better HLTV alignment
	weights := [6]int{0, 0, 2, 6, 14, 30} // 0, 0, double=2, triple=6, quad=14, ace=30
	total := 0
	for i := 2; i <= 5; i++ {
		total += m[i] * weights[i]
	}
	return total
}

// ComputeSideRating calculates eco rating for a specific side (T or CT)
// Uses the same linear formula as ComputeFinalRating
func ComputeSideRating(rounds int, kills int, deaths int, damage int, ecoKillValue float64,
	roundSwing float64, kast float64, multiKills [6]int, clutchRounds int, clutchWins int) float64 {

	roundsF := float64(rounds)
	if roundsF == 0 {
		return 0
	}

	// Core stats
	kpr := float64(kills) / roundsF
	dpr := float64(deaths) / roundsF
	adr := float64(damage) / roundsF
	kastPct := kast / roundsF
	avgSwing := roundSwing / roundsF

	// Linear formula components (same as ComputeFinalRating)
	var kprContrib float64
	if kpr >= 0.7 {
		kprContrib = (kpr - 0.7) * 0.75
	} else {
		kprContrib = (kpr - 0.7) * 0.55
	}

	var dprContrib float64
	if dpr <= 0.7 {
		dprContrib = (0.7 - dpr) * 0.15
	} else {
		dprContrib = (0.7 - dpr) * 0.55
	}

	var adrContrib float64
	if adr >= 75.0 {
		adrContrib = (adr - 75.0) * 0.015
	} else {
		adrContrib = (adr - 75.0) * 0.004
	}

	var kastContrib float64
	if kastPct >= 0.70 {
		kastContrib = (kastPct - 0.70) * 0.20
	} else {
		kastContrib = (kastPct - 0.70) * 0.35
	}

	// Round Swing contribution: context-aware performance metric
	var swingContrib float64
	if avgSwing >= 0 {
		swingContrib = avgSwing * 0.75
	} else {
		swingContrib = avgSwing * 1.00
	}

	multiKillBonus := float64(sumMulti(multiKills)) / roundsF
	multiContrib := multiKillBonus * 0.015

	// Combine
	rating := 1.0 + kprContrib + dprContrib + adrContrib + kastContrib + swingContrib + multiContrib

	return math.Max(MinRating, math.Min(MaxRating, rating))
}
