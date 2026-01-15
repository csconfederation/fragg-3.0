package rating

import (
	"eco-rating/model"
	"math"
)

// ComputeFinalRating calculates an HLTV 3.0-style rating
// Based on HLTV's 60-40 output/cost balance with recent weight adjustments
// Enhanced with opening duel, trade efficiency, and utility components
func ComputeFinalRating(p *model.PlayerStats) float64 {
	rounds := float64(p.RoundsPlayed)
	if rounds == 0 {
		return 0
	}

	// === Component 1: Kill Rating (28%) ===
	// Eco-adjusted kills per round - primary component per HLTV updates
	ecoKPR := p.EcoKillValue / rounds
	// Enhanced scaling for exceptional fraggers
	killRatio := ecoKPR / BaselineKPR
	var killRating float64
	if killRatio >= 1.5 {
		// Exceptional fraggers: very strong boost
		killRating = 1.0 + (killRatio-1.0)*1.3
	} else if killRatio >= 1.2 {
		// Good fraggers: moderate boost
		killRating = 1.0 + (killRatio-1.0)*0.7
	} else if killRatio >= 0.8 {
		// Average performance: normal scaling
		killRating = math.Pow(killRatio, 0.9)
	} else {
		// Below average: stronger penalty
		killRating = math.Pow(killRatio, 1.1)
	}

	// === Component 2: Death Rating (16%) ===
	// Balanced death penalty - reward low deaths, penalize high deaths
	dpr := float64(p.Deaths) / rounds
	deathRatio := dpr / BaselineDPR
	var deathRating float64
	if deathRatio <= 0.5 {
		// Exceptionally low deaths: very strong reward
		deathRating = 2.0 - (deathRatio * 0.2)
	} else if deathRatio <= 0.8 {
		// Very low deaths: strong reward
		deathRating = 1.7 - (deathRatio * 0.4)
	} else if deathRatio <= 1.0 {
		// Below baseline: moderate reward
		deathRating = 1.4 - (deathRatio * 0.3)
	} else if deathRatio <= 1.3 {
		// Above baseline: moderate penalty
		deathRating = 1.0 / math.Pow(deathRatio, 1.0)
	} else {
		// High deaths: stronger penalty
		deathRating = 1.0 / math.Pow(deathRatio, 1.2)
	}
	deathRating = math.Max(0.3, math.Min(1.9, deathRating))

	// === Component 3: ADR Rating (18%) ===
	// Eco-adjusted damage per round - reward high damage dealers
	adr := float64(p.Damage) / rounds
	adrRatio := adr / BaselineADR
	var adrRating float64
	if adrRatio >= 1.4 {
		// Exceptional damage: very strong boost
		adrRating = 0.8 + (adrRatio * 0.6)
	} else if adrRatio >= 1.0 {
		// Above baseline: strong scaling for high damage
		adrRating = 0.7 + (adrRatio * 0.5)
	} else if adrRatio >= 0.8 {
		// Below baseline: stronger penalty
		adrRating = 0.4 + (adrRatio * 0.6)
	} else {
		// Low damage: very strong penalty
		adrRating = 0.3 + (adrRatio * 0.5)
	}

	// === Component 4: Round Swing Rating (10%) ===
	// Advanced round swing system
	avgSwing := p.RoundSwing / rounds
	var swingRating float64
	if avgSwing >= 0.05 {
		// High positive swing: moderate reward
		swingRating = 1.0 + (avgSwing/0.15)*0.4
	} else if avgSwing >= 0 {
		// Low positive swing: small reward
		swingRating = 1.0 + (avgSwing/0.10)*0.2
	} else {
		// Negative swing: penalty
		swingRating = 1.0 + (avgSwing/0.10)*0.3
	}
	swingRating = math.Max(0.6, math.Min(1.4, swingRating))

	// === Component 5: Multi-Kill Rating (10%) ===
	// Explosive moments - penalize if overall performance is poor
	multiKillBonus := float64(sumMulti(p.MultiKills)) / rounds
	multiKillRating := math.Min(math.Pow(multiKillBonus/BaselineMultiKill, 0.8), 2.0)

	// Sliding scale: multi-kill bonus proportional to overall performance
	overallPerformance := (ecoKPR/BaselineKPR + (adr / BaselineADR) + p.KAST/BaselineKAST) / 3.0
	if multiKillRating > 1.0 {
		penaltyFactor := math.Pow(math.Min(1.0, overallPerformance), 2)
		multiKillRating = 1.0 + (multiKillRating-1.0)*penaltyFactor
	}

	// === Component 6: KAST Rating (6%) ===
	// Consistency metric with penalties for low KAST
	kastRatio := p.KAST / BaselineKAST
	var kastRating float64
	if kastRatio >= 1.2 {
		// Very high KAST: diminishing returns
		kastRating = 1.0 + (kastRatio-1.0)*0.6
	} else if kastRatio >= 0.9 {
		// Good KAST: normal scaling
		kastRating = kastRatio
	} else {
		// Low KAST: stronger penalty
		kastRating = math.Pow(kastRatio, 1.2)
	}

	// === Component 7: Opening Duel Rating (6%) ===
	// Measures entry impact - success rate and round conversion
	openingRating := 1.0
	if p.OpeningAttempts > 0 {
		successRate := float64(p.OpeningSuccesses) / float64(p.OpeningAttempts)
		// Normalize against baseline (50% success rate)
		successRatio := successRate / BaselineOpeningSuccessRate

		// Win conversion after opening kill
		winConversion := 0.0
		if p.OpeningSuccesses > 0 {
			winConversion = float64(p.RoundsWonAfterOpening) / float64(p.OpeningSuccesses)
		}

		// Combined opening rating: 70% success rate, 30% win conversion
		openingRating = successRatio*0.7 + winConversion*0.6
		openingRating = math.Max(0.5, math.Min(1.6, openingRating))
	}

	// === Component 8: Trade Efficiency Rating (4%) ===
	// Measures team coordination - trading teammates and being traded
	tradeRating := 1.0

	// Reward for trading teammates
	tradeKillsPerRound := float64(p.TradeKills) / rounds
	tradeKillRatio := tradeKillsPerRound / BaselineTradeKillsPerRound
	tradeRating += (tradeKillRatio - 1.0) * 0.3

	// Reward for being traded when dying
	if p.Deaths > 0 {
		tradedPct := float64(p.TradedDeaths) / float64(p.Deaths)
		tradeRating += tradedPct * 0.2
	}

	// Reward for saving teammates
	savesPerRound := float64(p.SavedTeammate) / rounds
	tradeRating += savesPerRound * 1.5

	tradeRating = math.Max(0.6, math.Min(1.5, tradeRating))

	// === Component 9: Utility Rating (2%) ===
	// Measures support impact - utility damage and flash assists
	utilityRating := 1.0

	// Utility damage contribution
	utilDmgPerRound := float64(p.UtilityDamage) / rounds
	utilDmgRatio := utilDmgPerRound / BaselineUtilityDamage

	// Flash assist contribution
	flashAssistsPerRound := float64(p.FlashAssists) / rounds
	flashAssistRatio := flashAssistsPerRound / BaselineFlashAssists

	// Enemy flash duration contribution
	enemyFlashPerRound := p.EnemyFlashDuration / rounds
	enemyFlashRatio := enemyFlashPerRound / BaselineEnemyFlashDur

	// Combined utility score (weighted average)
	utilityScore := (utilDmgRatio*0.4 + flashAssistRatio*0.3 + enemyFlashRatio*0.3)
	utilityRating = 0.7 + utilityScore*0.3
	utilityRating = math.Max(0.5, math.Min(1.4, utilityRating))

	// === Additional Penalties/Bonuses ===

	// Proportional clutch modifier (replaces binary penalty)
	clutchModifier := 0.0
	if p.ClutchRounds > 0 {
		clutchWinRate := float64(p.ClutchWins) / float64(p.ClutchRounds)
		if clutchWinRate < 0.3 {
			// Penalty for low clutch win rate
			clutchModifier = -float64(p.ClutchRounds) * (0.3 - clutchWinRate) * 0.04
		} else {
			// Bonus for good clutch performance
			clutchModifier = float64(p.ClutchWins) * 0.015
		}
	}

	// AWP economy penalty - dying with AWP without getting a kill
	awpPenalty := 0.0
	if p.AWPDeathsNoKill > 0 {
		awpPenalty = float64(p.AWPDeathsNoKill) / rounds * 0.12
	}

	// === Combine Components ===
	rating := killRating*WeightKillRating +
		deathRating*WeightDeathRating +
		adrRating*WeightADRRating +
		swingRating*WeightSwingRating +
		multiKillRating*WeightMultiKillRating +
		kastRating*WeightKASTRating +
		openingRating*WeightOpeningRating +
		tradeRating*WeightTradeRating +
		utilityRating*WeightUtilityRating +
		clutchModifier -
		awpPenalty

	// Clamp to reasonable range
	return math.Max(MinRating, math.Min(MaxRating, rating))
}

func sumMulti(m [6]int) int {
	// Weight multi-kills with exponential scaling like HLTV 3.0
	// Higher kill counts are exponentially more valuable
	weights := [6]int{0, 0, 1, 3, 7, 15} // 0, 0, double=1, triple=3, quad=7, ace=15
	total := 0
	for i := 2; i <= 5; i++ {
		total += m[i] * weights[i]
	}
	return total
}

// ComputeSideRating calculates eco rating for a specific side (T or CT)
// Uses the same formula as ComputeFinalRating but with side-specific stats
// Note: Per-side rating uses simplified formula without opening/trade/utility components
// since those stats aren't tracked per-side currently
func ComputeSideRating(rounds int, kills int, deaths int, damage int, ecoKillValue float64,
	roundSwing float64, kast float64, multiKills [6]int, clutchRounds int, clutchWins int) float64 {

	roundsF := float64(rounds)
	if roundsF == 0 {
		return 0
	}

	// === Component 1: Kill Rating (28%) ===
	ecoKPR := ecoKillValue / roundsF
	killRatio := ecoKPR / BaselineKPR
	var killRating float64
	if killRatio >= 1.5 {
		killRating = 1.0 + (killRatio-1.0)*1.3
	} else if killRatio >= 1.2 {
		killRating = 1.0 + (killRatio-1.0)*0.7
	} else if killRatio >= 0.8 {
		killRating = math.Pow(killRatio, 0.9)
	} else {
		killRating = math.Pow(killRatio, 1.1)
	}

	// === Component 2: Death Rating (16%) ===
	dpr := float64(deaths) / roundsF
	deathRatio := dpr / BaselineDPR
	var deathRating float64
	if deathRatio <= 0.5 {
		deathRating = 2.0 - (deathRatio * 0.2)
	} else if deathRatio <= 0.8 {
		deathRating = 1.7 - (deathRatio * 0.4)
	} else if deathRatio <= 1.0 {
		deathRating = 1.4 - (deathRatio * 0.3)
	} else if deathRatio <= 1.3 {
		deathRating = 1.0 / math.Pow(deathRatio, 1.0)
	} else {
		deathRating = 1.0 / math.Pow(deathRatio, 1.2)
	}
	deathRating = math.Max(0.3, math.Min(1.9, deathRating))

	// === Component 3: ADR Rating (18%) ===
	adr := float64(damage) / roundsF
	adrRatio := adr / BaselineADR
	var adrRating float64
	if adrRatio >= 1.4 {
		adrRating = 0.8 + (adrRatio * 0.6)
	} else if adrRatio >= 1.0 {
		adrRating = 0.7 + (adrRatio * 0.5)
	} else if adrRatio >= 0.8 {
		adrRating = 0.4 + (adrRatio * 0.6)
	} else {
		adrRating = 0.3 + (adrRatio * 0.5)
	}

	// === Component 4: Round Swing Rating (10%) ===
	avgSwing := roundSwing / roundsF
	var swingRating float64
	if avgSwing >= 0.05 {
		swingRating = 1.0 + (avgSwing/0.15)*0.4
	} else if avgSwing >= 0 {
		swingRating = 1.0 + (avgSwing/0.10)*0.2
	} else {
		swingRating = 1.0 + (avgSwing/0.10)*0.3
	}
	swingRating = math.Max(0.6, math.Min(1.4, swingRating))

	// === Component 5: Multi-Kill Rating (10%) ===
	multiKillBonus := float64(sumMulti(multiKills)) / roundsF
	multiKillRating := math.Min(math.Pow(multiKillBonus/BaselineMultiKill, 0.8), 2.0)

	kastPct := kast / roundsF
	overallPerformance := (ecoKPR/BaselineKPR + (adr / BaselineADR) + kastPct/BaselineKAST) / 3.0
	if multiKillRating > 1.0 {
		penaltyFactor := math.Pow(math.Min(1.0, overallPerformance), 2)
		multiKillRating = 1.0 + (multiKillRating-1.0)*penaltyFactor
	}

	// === Component 6: KAST Rating (6%) ===
	kastRatio := kastPct / BaselineKAST
	var kastRating float64
	if kastRatio >= 1.2 {
		kastRating = 1.0 + (kastRatio-1.0)*0.6
	} else if kastRatio >= 0.9 {
		kastRating = kastRatio
	} else {
		kastRating = math.Pow(kastRatio, 1.2)
	}

	// === Proportional Clutch Modifier ===
	clutchModifier := 0.0
	if clutchRounds > 0 {
		clutchWinRate := float64(clutchWins) / float64(clutchRounds)
		if clutchWinRate < 0.3 {
			clutchModifier = -float64(clutchRounds) * (0.3 - clutchWinRate) * 0.04
		} else {
			clutchModifier = float64(clutchWins) * 0.015
		}
	}

	// === Combine Components ===
	// Per-side uses adjusted weights (opening/trade/utility default to 1.0)
	// Redistributed: Kill 32%, Death 18%, ADR 20%, Swing 12%, Multi 12%, KAST 6%
	rating := killRating*0.32 +
		deathRating*0.18 +
		adrRating*0.20 +
		swingRating*0.12 +
		multiKillRating*0.12 +
		kastRating*0.06 +
		clutchModifier

	return math.Max(MinRating, math.Min(MaxRating, rating))
}
