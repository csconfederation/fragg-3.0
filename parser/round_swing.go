package parser

import (
	"eco-rating/model"
	"math"
)

// CalculateAdvancedRoundSwing computes context-aware round swing based on player actions and situational factors
func CalculateAdvancedRoundSwing(roundStats *model.RoundStats, context *model.RoundContext, playerEquipValue float64, teamEquipValue float64) float64 {
	baseSwing := 0.0

	// === Base Round Outcome ===
	if roundStats.TeamWon {
		baseSwing = 0.05 // Base positive swing for winners
	} else {
		baseSwing = -0.08 // Base negative swing for losers
	}

	// === Performance Contribution ===
	performanceBonus := calculatePerformanceContribution(roundStats)

	// === Situational Modifiers ===
	situationalBonus := calculateSituationalBonus(roundStats, context)

	// === Impact Actions ===
	impactBonus := calculateImpactActions(roundStats, context)

	// === Economy Context ===
	economyModifier := calculateEconomyModifier(roundStats, playerEquipValue, teamEquipValue, context.RoundType)

	// === Multi-kill Scaling ===
	multiKillBonus := calculateMultiKillBonus(roundStats)

	// === Clutch Situations ===
	clutchModifier := calculateClutchModifier(roundStats)

	// === NEW: Utility Impact ===
	utilityBonus := calculateUtilityImpact(roundStats)

	// === NEW: Trade Speed Bonus ===
	tradeSpeedBonus := calculateTradeSpeedBonus(roundStats)

	// === NEW: Exit Frag Penalty ===
	exitFragPenalty := calculateExitFragPenalty(roundStats)

	// === NEW: Death Timing Penalty ===
	deathTimingPenalty := calculateDeathTimingPenalty(roundStats)

	// === NEW: Team Flash Penalty ===
	teamFlashPenalty := calculateTeamFlashPenalty(roundStats)

	// === NEW: Weapon-Based Adjustments ===
	weaponBonus := calculateWeaponBonus(roundStats)

	// Combine all factors
	totalSwing := baseSwing + performanceBonus + situationalBonus + impactBonus + multiKillBonus + clutchModifier +
		utilityBonus + tradeSpeedBonus - exitFragPenalty - deathTimingPenalty - teamFlashPenalty + weaponBonus

	// Apply economy modifier as multiplier
	totalSwing *= economyModifier

	// === NEW: Score Differential Modifier ===
	totalSwing *= context.RoundImportance

	// Clamp to reasonable range (expanded slightly for new factors)
	return math.Max(-0.30, math.Min(0.40, totalSwing))
}

// calculatePerformanceContribution calculates swing based on kills, damage, and survival
func calculatePerformanceContribution(roundStats *model.RoundStats) float64 {
	contribution := 0.0

	// Kill contribution with diminishing returns
	if roundStats.Kills > 0 {
		contribution += float64(roundStats.Kills) * 0.04
		// Bonus for multiple kills
		if roundStats.Kills >= 2 {
			contribution += float64(roundStats.Kills-1) * 0.02
		}
	}

	// Damage contribution (normalized) - exclude exit frag damage
	effectiveDamage := roundStats.Damage
	if roundStats.IsExitFrag {
		effectiveDamage = int(float64(effectiveDamage) * 0.5) // Reduce exit frag damage contribution
	}
	damageContrib := float64(effectiveDamage) / 400.0 // 400 damage = 0.1 swing
	contribution += math.Min(damageContrib, 0.08)

	// Assist contribution
	contribution += float64(roundStats.Assists) * 0.015

	// Flash assist contribution
	contribution += float64(roundStats.FlashAssists) * 0.01

	// Survival bonus (context-dependent)
	if roundStats.Survived {
		if roundStats.TeamWon {
			contribution += 0.02 // Small bonus for surviving a won round
		} else {
			contribution += 0.04 // Larger bonus for surviving a lost round (save)
			roundStats.SavedWeapons = true
		}
	}

	return contribution
}

// calculateSituationalBonus calculates swing based on round context and timing
func calculateSituationalBonus(roundStats *model.RoundStats, context *model.RoundContext) float64 {
	bonus := 0.0

	// Opening kill/death impact
	if roundStats.OpeningKill {
		bonus += 0.06 // Strong bonus for opening kills
		if context.RoundType == "pistol" {
			bonus += 0.02 // Extra bonus for pistol round opening kills
		}
	}

	if roundStats.OpeningDeath {
		bonus -= 0.04 // Penalty for opening death
	}

	// Entry fragging bonus
	if roundStats.EntryFragger {
		bonus += 0.04
	}

	// Trade kill/death impact (base bonus, speed bonus calculated separately)
	if roundStats.TradeKill {
		bonus += 0.02 // Base bonus for trading teammates
	}

	if roundStats.TradeDeath {
		bonus += 0.015 // Small bonus if death was traded (team play)
	}

	// Round type modifiers
	switch context.RoundType {
	case "pistol":
		bonus *= 1.3 // Pistol rounds are more impactful
	case "eco":
		if roundStats.TeamWon {
			bonus *= 1.4 // Eco wins are very impactful
		}
	case "force":
		bonus *= 1.1 // Force buy rounds slightly more impactful
	}

	// Overtime rounds are more valuable
	if context.IsOvertimeRound {
		bonus *= 1.2
	}

	return bonus
}

// calculateImpactActions calculates swing for high-impact actions like bomb plants/defuses
func calculateImpactActions(roundStats *model.RoundStats, context *model.RoundContext) float64 {
	bonus := 0.0

	// Bomb plant bonus
	if roundStats.PlantedBomb {
		bonus += 0.08
		if context.TimeRemaining < 30.0 {
			bonus += 0.02 // Extra bonus for late plants
		}
	}

	// Bomb defuse bonus
	if roundStats.DefusedBomb {
		bonus += 0.10
		if context.TimeRemaining < 10.0 {
			bonus += 0.03 // Extra bonus for clutch defuses
		}
	}

	// Anti-eco performance
	if roundStats.EcoKill {
		bonus += 0.04 // Bonus for getting eco kills
	}

	if roundStats.AntiEcoKill {
		bonus -= 0.06 // Penalty for dying to eco
	}

	return bonus
}

// calculateMultiKillBonus calculates exponential bonus for multi-kills
func calculateMultiKillBonus(roundStats *model.RoundStats) float64 {
	if roundStats.MultiKillRound < 2 {
		return 0.0
	}

	// Exponential scaling for multi-kills
	switch roundStats.MultiKillRound {
	case 2:
		return 0.03 // Double kill
	case 3:
		return 0.08 // Triple kill
	case 4:
		return 0.15 // Quadruple kill
	case 5:
		return 0.25 // Ace
	default:
		return 0.0
	}
}

// calculateClutchModifier calculates swing for clutch situations
func calculateClutchModifier(roundStats *model.RoundStats) float64 {
	modifier := 0.0

	if roundStats.ClutchAttempt {
		if roundStats.ClutchWon {
			// Clutch win bonus scales with difficulty
			switch {
			case roundStats.ClutchKills >= 4:
				modifier += 0.20 // 1v4+ clutch
			case roundStats.ClutchKills >= 3:
				modifier += 0.15 // 1v3 clutch
			case roundStats.ClutchKills >= 2:
				modifier += 0.10 // 1v2 clutch
			default:
				modifier += 0.06 // 1v1 clutch
			}
		} else {
			// Small penalty for failed clutch, but reward the attempt
			modifier -= 0.02
			// Bonus for kills in failed clutch
			modifier += float64(roundStats.ClutchKills) * 0.02
		}
	}

	return modifier
}

// determineRoundType determines the economic context of the round
func determineRoundType(roundNumber int) string {
	switch {
	case roundNumber == 1 || roundNumber == 16: // Pistol rounds
		return "pistol"
	case roundNumber <= 3 || (roundNumber >= 16 && roundNumber <= 18): // Early rounds often eco
		return "eco"
	case roundNumber%3 == 0: // Every 3rd round might be force
		return "force"
	default:
		return "full"
	}
}

// calculateEconomyModifier calculates multiplier based on equipment context
func calculateEconomyModifier(roundStats *model.RoundStats, playerEquip, teamEquip float64, roundType string) float64 {
	modifier := 1.0

	// Equipment disadvantage/advantage
	avgTeamEquip := teamEquip / 5.0
	equipRatio := playerEquip / math.Max(avgTeamEquip, 500.0)

	if equipRatio < 0.5 {
		// Player on eco/save
		if roundStats.TeamWon {
			modifier *= 1.3 // Eco wins are impressive
		}
		if roundStats.Kills > 0 {
			modifier *= 1.2 // Eco frags are valuable
		}
	} else if equipRatio > 1.5 {
		// Player with expensive equipment
		if !roundStats.TeamWon && roundStats.Kills == 0 {
			modifier *= 0.8 // Penalty for poor performance with good equipment
		}
	}

	// Round type economy context
	switch roundType {
	case "eco":
		if roundStats.Kills > 0 {
			modifier *= 1.4 // Eco frags are very valuable
		}
	case "force":
		modifier *= 1.1 // Force buy performance slightly more valuable
	}

	return modifier
}

// === NEW CALCULATION FUNCTIONS ===

// calculateUtilityImpact calculates swing bonus for utility damage and flash impact
func calculateUtilityImpact(roundStats *model.RoundStats) float64 {
	bonus := 0.0

	// Utility damage contribution (HE, molotov, incendiary)
	if roundStats.UtilityDamage > 0 {
		// Scale utility damage: 100 damage = 0.03 swing
		utilityContrib := float64(roundStats.UtilityDamage) / 100.0 * 0.03
		bonus += math.Min(utilityContrib, 0.06) // Cap at 0.06
	}

	// Enemy flash duration contribution - rewards effective flashes
	if roundStats.EnemyFlashDuration > 0 {
		// Scale: 3 seconds of enemy flash = 0.02 swing
		flashContrib := roundStats.EnemyFlashDuration / 3.0 * 0.02
		bonus += math.Min(flashContrib, 0.04) // Cap at 0.04
	}

	// Flash assist contribution
	if roundStats.FlashAssists > 0 {
		bonus += float64(roundStats.FlashAssists) * 0.015 // 0.015 per flash assist
	}

	return bonus
}

// calculateTradeSpeedBonus calculates bonus for fast trades
func calculateTradeSpeedBonus(roundStats *model.RoundStats) float64 {
	if !roundStats.TradeKill || roundStats.TradeSpeed <= 0 {
		return 0.0
	}

	// Fast trades (< 2 seconds) get full bonus
	// Trades between 2-5 seconds get partial bonus
	// Trades > 5 seconds get minimal bonus
	switch {
	case roundStats.TradeSpeed < 2.0:
		return 0.025 // Fast trade bonus
	case roundStats.TradeSpeed < 3.0:
		return 0.015 // Medium trade bonus
	case roundStats.TradeSpeed < 5.0:
		return 0.008 // Slow trade bonus
	default:
		return 0.0 // Too slow to count as meaningful trade
	}
}

// calculateExitFragPenalty calculates penalty for exit frags (kills after round decided)
func calculateExitFragPenalty(roundStats *model.RoundStats) float64 {
	if !roundStats.IsExitFrag {
		return 0.0
	}

	// Exit frags are worth less - reduce the kill contribution
	// Penalty scales with number of exit frags
	return float64(roundStats.ExitFrags) * 0.02
}

// calculateDeathTimingPenalty calculates penalty for early deaths
func calculateDeathTimingPenalty(roundStats *model.RoundStats) float64 {
	if roundStats.Survived || roundStats.DeathTime <= 0 {
		return 0.0
	}

	// Early deaths (first 30 seconds) are more punishing
	// Deaths after bomb plant are less punishing
	switch {
	case roundStats.DeathTime < 15.0:
		return 0.03 // Very early death - strong penalty
	case roundStats.DeathTime < 30.0:
		return 0.02 // Early death - moderate penalty
	case roundStats.DeathTime < 60.0:
		return 0.01 // Mid-round death - small penalty
	default:
		return 0.0 // Late death - no additional penalty
	}
}

// calculateTeamFlashPenalty calculates penalty for flashing teammates
func calculateTeamFlashPenalty(roundStats *model.RoundStats) float64 {
	if roundStats.TeamFlashCount == 0 {
		return 0.0
	}

	// Penalty based on number of team flashes and duration
	countPenalty := float64(roundStats.TeamFlashCount) * 0.005
	durationPenalty := roundStats.TeamFlashDuration * 0.002 // Per second of team flash

	return math.Min(countPenalty+durationPenalty, 0.04) // Cap at 0.04
}

// calculateWeaponBonus calculates bonus/penalty for weapon-specific achievements
func calculateWeaponBonus(roundStats *model.RoundStats) float64 {
	bonus := 0.0

	// AWP performance - high-risk, high-reward weapon
	if roundStats.AWPKill {
		if roundStats.LostAWP {
			// Got a kill but lost the AWP - reduced reward (still losing economy)
			bonus += 0.005
		} else {
			// Got a kill and kept the AWP - full reward
			bonus += 0.02
		}
	} else if roundStats.LostAWP {
		// No kill and lost the AWP - significant penalty
		bonus -= 0.05
	}

	// Knife kills - humiliation bonus
	if roundStats.KnifeKill {
		bonus += 0.03
	}

	// Pistol vs rifle kills - skill bonus
	if roundStats.PistolVsRifleKill {
		bonus += 0.025
	}

	return bonus
}
