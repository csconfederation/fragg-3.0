// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file implements the round swing calculation algorithm, which measures
// a player's impact on round outcomes based on their actions, timing, and context.
package parser

import (
	"eco-rating/model"
	"eco-rating/rating"
	"math"
)

// RoundSwingCalculator computes player impact scores for rounds.
// It encapsulates all the calculation logic for round swing values.
type RoundSwingCalculator struct {
	roundStats       *model.RoundStats
	context          *model.RoundContext
	playerEquipValue float64
	teamEquipValue   float64
}

// NewRoundSwingCalculator creates a new calculator for the given round data.
func NewRoundSwingCalculator(roundStats *model.RoundStats, context *model.RoundContext, playerEquipValue, teamEquipValue float64) *RoundSwingCalculator {
	return &RoundSwingCalculator{
		roundStats:       roundStats,
		context:          context,
		playerEquipValue: playerEquipValue,
		teamEquipValue:   teamEquipValue,
	}
}

// Calculate computes the total round swing value.
// Returns a value typically between -0.30 and +0.40 per round.
func (c *RoundSwingCalculator) Calculate() float64 {
	involvement := c.calculateInvolvement()
	baseSwing := c.calculateBaseSwing(involvement)

	totalSwing := baseSwing +
		c.PerformanceContribution() +
		c.SituationalBonus() +
		c.ImpactActions() +
		c.MultiKillBonus() +
		c.ClutchModifier() +
		c.UtilityImpact() +
		c.TradeSpeedBonus() +
		c.WeaponBonus() -
		c.ExitFragPenalty() -
		c.DeathTimingPenalty() -
		c.TeamFlashPenalty() -
		c.FailedTradePenalty()

	totalSwing *= c.EconomyModifier()
	totalSwing *= c.context.RoundImportance

	return math.Max(rating.SwingMinClamp, math.Min(rating.SwingMaxClamp, totalSwing))
}

// calculateInvolvement computes the player's involvement level in the round.
func (c *RoundSwingCalculator) calculateInvolvement() float64 {
	involvement := float64(c.roundStats.Kills)
	involvement += float64(c.roundStats.Assists) * rating.InvolvementAssistWeight
	involvement += math.Min(1.0, float64(c.roundStats.Damage)/rating.InvolvementDamageNorm)
	if c.roundStats.PlantedBomb {
		involvement += rating.InvolvementBombWeight
	}
	if c.roundStats.DefusedBomb {
		involvement += rating.InvolvementBombWeight
	}
	if c.roundStats.Survived {
		involvement += rating.InvolvementSurvivalWeight
	}
	return math.Max(0.0, math.Min(involvement/rating.InvolvementNormDivisor, 1.0))
}

// calculateBaseSwing computes the base swing from round outcome and involvement.
func (c *RoundSwingCalculator) calculateBaseSwing(involvement float64) float64 {
	if c.roundStats.TeamWon {
		return rating.SwingBaseWin * involvement
	}
	return rating.SwingBaseLoss * involvement
}

// CalculateAdvancedRoundSwing computes a player's impact score for a single round.
// This is the legacy function that delegates to RoundSwingCalculator.
func CalculateAdvancedRoundSwing(roundStats *model.RoundStats, context *model.RoundContext, playerEquipValue float64, teamEquipValue float64) float64 {
	calc := NewRoundSwingCalculator(roundStats, context, playerEquipValue, teamEquipValue)
	return calc.Calculate()
}

// PerformanceContribution computes the base performance score from
// kills, damage, assists, and survival status.
func (c *RoundSwingCalculator) PerformanceContribution() float64 {
	contribution := 0.0

	if c.roundStats.Kills > 0 {
		contribution += float64(c.roundStats.Kills) * rating.KillContribPerKill
		if c.roundStats.Kills >= 2 {
			contribution += float64(c.roundStats.Kills-1) * rating.KillContribMultiBonus
		}
	}

	effectiveDamage := c.roundStats.Damage
	if c.roundStats.IsExitFrag {
		effectiveDamage = int(float64(effectiveDamage) * rating.ExitFragDamageMultiplier)
	}
	damageContrib := float64(effectiveDamage) / rating.DamageContribDivisor
	contribution += math.Min(damageContrib, rating.DamageContribMax)

	contribution += float64(c.roundStats.Assists) * rating.AssistContrib
	contribution += float64(c.roundStats.FlashAssists) * rating.FlashAssistContrib

	if c.roundStats.Survived {
		if c.roundStats.TeamWon {
			contribution += rating.SurvivalContribWin
		} else {
			contribution += rating.SurvivalContribLoss
			c.roundStats.SavedWeapons = true
		}
	} else {
		if c.roundStats.TradeDeath {
			contribution -= rating.DeathPenaltyTraded
		} else {
			contribution -= rating.DeathPenaltyUntraded
		}
	}

	return contribution
}

// SituationalBonus adds bonuses/penalties for opening kills, trades,
// entry fragging, and round type (pistol, eco, force).
func (c *RoundSwingCalculator) SituationalBonus() float64 {
	bonus := 0.0

	if c.roundStats.OpeningKill {
		bonus += rating.OpeningKillBonus
		if c.context.RoundType == "pistol" {
			bonus += rating.OpeningKillPistolBonus
		}
	}

	if c.roundStats.OpeningDeath {
		if c.roundStats.TradeDeath {
			bonus -= rating.OpeningDeathTraded
		} else {
			bonus -= rating.OpeningDeathUntraded
		}
	}

	if c.roundStats.EntryFragger {
		bonus += rating.EntryFragBonus
	}

	if c.roundStats.TradeKill {
		bonus += rating.TradeKillBonus
	}

	if c.roundStats.TradeDeath {
		bonus += rating.TradeDeathMitigation
	}

	if c.roundStats.TradeDenials > 0 {
		bonus += float64(c.roundStats.TradeDenials) * rating.TradeDenialBonus
	}

	switch c.context.RoundType {
	case "pistol":
		bonus *= rating.PistolRoundMultiplier
	case "eco":
		if c.roundStats.TeamWon {
			bonus *= rating.EcoRoundWinMultiplier
		}
	case "force":
		bonus *= rating.ForceRoundMultiplier
	}

	if c.context.IsOvertimeRound {
		bonus *= rating.OvertimeMultiplier
	}

	return bonus
}

// ImpactActions rewards bomb plants/defuses and eco kills,
// while penalizing anti-eco deaths.
func (c *RoundSwingCalculator) ImpactActions() float64 {
	bonus := 0.0

	if c.roundStats.PlantedBomb {
		bonus += rating.BombPlantBonus
		if c.context.TimeRemaining < rating.LateRoundTimeThreshold {
			bonus += rating.BombPlantLateBonus
		}
	}

	if c.roundStats.DefusedBomb {
		bonus += rating.BombDefuseBonus
		if c.context.TimeRemaining < rating.ClutchDefuseThreshold {
			bonus += rating.BombDefuseClutchBonus
		}
	}

	if c.roundStats.EcoKill {
		bonus += rating.EcoKillBonus
	}

	if c.roundStats.AntiEcoKill {
		bonus -= rating.AntiEcoDeathPenalty
	}

	return bonus
}

// MultiKillBonus returns bonus points for multi-kill rounds.
// 2K=0.03, 3K=0.08, 4K=0.15, 5K(ACE)=0.25
func (c *RoundSwingCalculator) MultiKillBonus() float64 {
	if c.roundStats.MultiKillRound < 2 {
		return 0.0
	}

	switch c.roundStats.MultiKillRound {
	case 2:
		return rating.MultiKill2KBonus
	case 3:
		return rating.MultiKill3KBonus
	case 4:
		return rating.MultiKill4KBonus
	case 5:
		return rating.MultiKill5KBonus
	default:
		return 0.0
	}
}

// ClutchModifier rewards successful clutches and partially rewards
// clutch attempts based on kills achieved.
func (c *RoundSwingCalculator) ClutchModifier() float64 {
	modifier := 0.0

	if c.roundStats.ClutchAttempt {
		if c.roundStats.ClutchWon {
			switch {
			case c.roundStats.ClutchKills >= 4:
				modifier += rating.ClutchWin4KBonus
			case c.roundStats.ClutchKills >= 3:
				modifier += rating.ClutchWin3KBonus
			case c.roundStats.ClutchKills >= 2:
				modifier += rating.ClutchWin2KBonus
			default:
				modifier += rating.ClutchWin1KBonus
			}
		} else {
			modifier -= rating.ClutchLossPenalty
			modifier += float64(c.roundStats.ClutchKills) * rating.ClutchLossKillMitigation
		}
	}

	return modifier
}

// determineRoundType categorizes a round as pistol, eco, force, or full buy
// based on the round number. Uses MR12 format constants.
func determineRoundType(roundNumber int) string {
	if rating.IsPistolRound(roundNumber) {
		return "pistol"
	}

	// Eco rounds: typically rounds 2-3 after pistol (first half) and 14-15 (second half)
	isFirstHalfEco := roundNumber >= 2 && roundNumber <= 3
	isSecondHalfEco := roundNumber >= rating.SecondHalfPistolRound+1 && roundNumber <= rating.SecondHalfPistolRound+2

	if isFirstHalfEco || isSecondHalfEco {
		return "eco"
	}

	// Force buy rounds (simplified heuristic)
	if roundNumber%3 == 0 {
		return "force"
	}

	return "full"
}

// EconomyModifier adjusts swing based on equipment advantage/disadvantage.
// Rewards performance when under-equipped, penalizes poor performance when over-equipped.
func (c *RoundSwingCalculator) EconomyModifier() float64 {
	modifier := 1.0

	avgTeamEquip := c.teamEquipValue / 5.0
	equipRatio := c.playerEquipValue / math.Max(avgTeamEquip, rating.MinTeamEquipValue)

	if equipRatio < rating.LowEquipRatioThreshold {
		if c.roundStats.TeamWon {
			modifier *= rating.LowEquipWinMultiplier
		}
		if c.roundStats.Kills > 0 {
			modifier *= rating.LowEquipKillMultiplier
		}
	} else if equipRatio > rating.HighEquipRatioThreshold {
		if !c.roundStats.TeamWon && c.roundStats.Kills == 0 {
			modifier *= rating.HighEquipFailMultiplier
		}
	}

	switch c.context.RoundType {
	case "eco":
		if c.roundStats.Kills > 0 {
			modifier *= rating.EcoRoundKillMultiplier
		}
	case "force":
		modifier *= rating.ForceRoundModifier
	}

	return modifier
}

// UtilityImpact rewards effective utility usage (damage, flashes).
func (c *RoundSwingCalculator) UtilityImpact() float64 {
	bonus := 0.0

	if c.roundStats.UtilityDamage > 0 {
		utilityContrib := float64(c.roundStats.UtilityDamage) / 100.0 * rating.UtilityDamageContribRate
		bonus += math.Min(utilityContrib, rating.UtilityDamageContribMax)
	}

	if c.roundStats.EnemyFlashDuration > 0 {
		flashContrib := c.roundStats.EnemyFlashDuration / 3.0 * rating.FlashDurationContribRate
		bonus += math.Min(flashContrib, rating.FlashDurationContribMax)
	}

	if c.roundStats.FlashAssists > 0 {
		bonus += float64(c.roundStats.FlashAssists) * rating.FlashAssistBonusRate
	}

	return bonus
}

// TradeSpeedBonus rewards fast trades (killing the enemy who killed a teammate).
// Faster trades receive higher bonuses.
func (c *RoundSwingCalculator) TradeSpeedBonus() float64 {
	if !c.roundStats.TradeKill || c.roundStats.TradeSpeed <= 0 {
		return 0.0
	}

	switch {
	case c.roundStats.TradeSpeed < rating.FastTradeThreshold:
		return rating.FastTradeBonus
	case c.roundStats.TradeSpeed < rating.MediumTradeThreshold:
		return rating.MediumTradeBonus
	case c.roundStats.TradeSpeed < rating.SlowTradeThreshold:
		return rating.SlowTradeBonus
	default:
		return 0.0
	}
}

// ExitFragPenalty penalizes kills that occur after the round is decided
// (exit frags have less impact on round outcome).
func (c *RoundSwingCalculator) ExitFragPenalty() float64 {
	if !c.roundStats.IsExitFrag {
		return 0.0
	}

	return float64(c.roundStats.ExitFrags) * rating.ExitFragPenaltyRate
}

// DeathTimingPenalty penalizes early deaths more heavily than late deaths.
// Dying in the first 15 seconds is most penalized.
func (c *RoundSwingCalculator) DeathTimingPenalty() float64 {
	if c.roundStats.Survived || c.roundStats.DeathTime <= 0 {
		return 0.0
	}

	switch {
	case c.roundStats.DeathTime < rating.EarlyDeathThreshold:
		return rating.EarlyDeathPenalty
	case c.roundStats.DeathTime < rating.MidDeathThreshold:
		return rating.MidDeathPenalty
	case c.roundStats.DeathTime < rating.LateDeathThreshold:
		return rating.LateDeathPenalty
	default:
		return 0.0
	}
}

// TeamFlashPenalty penalizes flashing teammates.
func (c *RoundSwingCalculator) TeamFlashPenalty() float64 {
	if c.roundStats.TeamFlashCount == 0 {
		return 0.0
	}

	countPenalty := float64(c.roundStats.TeamFlashCount) * rating.TeamFlashCountPenalty
	durationPenalty := c.roundStats.TeamFlashDuration * rating.TeamFlashDurationPenalty

	return math.Min(countPenalty+durationPenalty, rating.TeamFlashPenaltyMax)
}

// FailedTradePenalty penalizes failing to trade a nearby teammate's death.
func (c *RoundSwingCalculator) FailedTradePenalty() float64 {
	if c.roundStats.FailedTrades == 0 {
		return 0.0
	}

	return float64(c.roundStats.FailedTrades) * rating.FailedTradePenalty
}

// WeaponBonus rewards/penalizes based on weapon-specific performance.
// AWP kills are rewarded, losing an AWP without a kill is penalized.
// Knife kills and pistol vs rifle kills receive bonuses.
func (c *RoundSwingCalculator) WeaponBonus() float64 {
	bonus := 0.0

	if c.roundStats.AWPKill {
		if c.roundStats.LostAWP {
			bonus += rating.AWPKillBonusWithDeath
		} else {
			bonus += rating.AWPKillBonusWithSurvive
		}
	} else if c.roundStats.LostAWP {
		bonus -= rating.AWPLostNokillPenalty
	}

	if c.roundStats.KnifeKill {
		bonus += rating.KnifeKillBonus
	}

	if c.roundStats.PistolVsRifleKill {
		bonus += rating.PistolVsRifleKillBonus
	}

	return bonus
}
