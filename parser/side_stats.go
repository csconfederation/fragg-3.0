// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file provides helper functions for updating side-specific statistics
// (T-side and CT-side) to eliminate code duplication.
package parser

import (
	"eco-rating/model"
)

// SideStatsUpdater handles updating side-specific statistics for a player.
type SideStatsUpdater struct {
	player     *model.PlayerStats
	roundStats *model.RoundStats
}

// NewSideStatsUpdater creates a new updater for the given player and round.
func NewSideStatsUpdater(player *model.PlayerStats, roundStats *model.RoundStats) *SideStatsUpdater {
	return &SideStatsUpdater{
		player:     player,
		roundStats: roundStats,
	}
}

// UpdateSideStats updates the appropriate side statistics based on the player's side.
func (u *SideStatsUpdater) UpdateSideStats() {
	switch u.roundStats.PlayerSide {
	case "T":
		u.updateTSide()
	case "CT":
		u.updateCTSide()
	}
}

// updateTSide updates T-side specific statistics.
func (u *SideStatsUpdater) updateTSide() {
	u.player.TRoundsPlayed++
	u.player.TKills += u.roundStats.Kills
	u.player.TDamage += u.roundStats.Damage
	u.player.TEcoKillValue += u.roundStats.EconImpact

	if u.roundStats.Survived {
		u.player.TSurvivals++
	}
	if u.roundStats.DeathTime > 0 {
		u.player.TDeaths++
	}
	if u.roundStats.Kills >= 2 {
		u.player.TRoundsWithMultiKill++
	}
	if u.roundStats.Kills >= 0 && u.roundStats.Kills <= 5 {
		u.player.TMultiKills[u.roundStats.Kills]++
	}
	if u.roundStats.GotKill || u.roundStats.GotAssist || u.roundStats.Survived || u.roundStats.Traded {
		u.player.TKAST++
	}
	if u.roundStats.ClutchAttempt {
		u.player.TClutchRounds++
		if u.roundStats.ClutchWon {
			u.player.TClutchWins++
		}
	}
}

// updateCTSide updates CT-side specific statistics.
func (u *SideStatsUpdater) updateCTSide() {
	u.player.CTRoundsPlayed++
	u.player.CTKills += u.roundStats.Kills
	u.player.CTDamage += u.roundStats.Damage
	u.player.CTEcoKillValue += u.roundStats.EconImpact

	if u.roundStats.Survived {
		u.player.CTSurvivals++
	}
	if u.roundStats.DeathTime > 0 {
		u.player.CTDeaths++
	}
	if u.roundStats.Kills >= 2 {
		u.player.CTRoundsWithMultiKill++
	}
	if u.roundStats.Kills >= 0 && u.roundStats.Kills <= 5 {
		u.player.CTMultiKills[u.roundStats.Kills]++
	}
	if u.roundStats.GotKill || u.roundStats.GotAssist || u.roundStats.Survived || u.roundStats.Traded {
		u.player.CTKAST++
	}
	if u.roundStats.ClutchAttempt {
		u.player.CTClutchRounds++
		if u.roundStats.ClutchWon {
			u.player.CTClutchWins++
		}
	}
}

// UpdateCommonRoundStats updates statistics that are common to both sides.
func (u *SideStatsUpdater) UpdateCommonRoundStats() {
	if u.roundStats.GotKill || u.roundStats.GotAssist || u.roundStats.Survived || u.roundStats.Traded {
		u.player.KAST++
	}

	if u.roundStats.GotKill {
		u.player.RoundsWithKill++
		u.player.AttackRounds++
	}

	if u.roundStats.Kills >= 2 {
		u.player.RoundsWithMultiKill++
	}

	if u.roundStats.TeamWon {
		u.player.KillsInWonRounds += u.roundStats.Kills
		u.player.DamageInWonRounds += u.roundStats.Damage

		if u.roundStats.OpeningKill {
			u.player.RoundsWonAfterOpening++
		}
	}

	u.updateAWPStats()
	u.updateSupportStats()
	u.updateUtilityStats()
	u.updateTradeStats()
	u.updatePistolStats()
}

// updateAWPStats updates AWP-related statistics.
func (u *SideStatsUpdater) updateAWPStats() {
	if u.roundStats.AWPKill {
		u.player.RoundsWithAWPKill++
	}
	if u.roundStats.AWPKills >= 2 {
		u.player.AWPMultiKillRounds++
	}
	if u.roundStats.LostAWP {
		u.player.AWPDeaths++
		if !u.roundStats.AWPKill {
			u.player.AWPDeathsNoKill++
		}
	}
}

// updateSupportStats updates support-related statistics.
func (u *SideStatsUpdater) updateSupportStats() {
	if u.roundStats.GotAssist || u.roundStats.FlashAssists > 0 {
		u.player.SupportRounds++
		u.roundStats.IsSupportRound = true
	}

	if u.roundStats.GotAssist {
		u.player.AssistedKills += u.roundStats.Assists
	}
}

// updateUtilityStats updates utility-related statistics.
func (u *SideStatsUpdater) updateUtilityStats() {
	u.player.UtilityDamage += u.roundStats.UtilityDamage
	u.player.FlashesThrown += u.roundStats.FlashesThrown
	u.player.FlashAssists += u.roundStats.FlashAssists
	u.player.EnemyFlashDuration += u.roundStats.EnemyFlashDuration
	u.player.TeamFlashCount += u.roundStats.TeamFlashCount
	u.player.TeamFlashDuration += u.roundStats.TeamFlashDuration
	u.player.ExitFrags += u.roundStats.ExitFrags

	if u.roundStats.SavedByTeammate {
		u.player.SavedByTeammate++
	}
}

// updateTradeStats updates trade-related statistics.
func (u *SideStatsUpdater) updateTradeStats() {
	if u.roundStats.KnifeKill {
		u.player.KnifeKills++
	}

	if u.roundStats.PistolVsRifleKill {
		u.player.PistolVsRifleKills++
	}

	if u.roundStats.TradeKill {
		u.player.TradeKills++
		if u.roundStats.TradeSpeed > 0 && u.roundStats.TradeSpeed < 2.0 {
			u.player.FastTrades++
		}
	}

	if u.roundStats.DeathTime > 0 && u.roundStats.DeathTime < 30.0 {
		u.player.EarlyDeaths++
	}
}

// updatePistolStats updates pistol round statistics.
func (u *SideStatsUpdater) updatePistolStats() {
	if !u.roundStats.IsPistolRound {
		return
	}

	u.player.PistolRoundsPlayed++
	u.player.PistolRoundKills += u.roundStats.Kills
	u.player.PistolRoundDamage += u.roundStats.Damage

	if u.roundStats.DeathTime > 0 {
		u.player.PistolRoundDeaths++
	} else if u.roundStats.Survived {
		u.player.PistolRoundSurvivals++
	}

	if u.roundStats.TeamWon {
		u.player.PistolRoundsWon++
	}

	if u.roundStats.Kills >= 2 {
		u.player.PistolRoundMultiKills++
	}
}
