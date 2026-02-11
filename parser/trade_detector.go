// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file implements trade detection logic, which identifies when a player's
// death is "traded" by a teammate killing the original attacker within a time window.
package parser

import (
	"eco-rating/model"
	"eco-rating/rating"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// pendingTrade tracks a potential trade opportunity after a kill.
type pendingTrade struct {
	KillerID           uint64
	KillerTeam         common.Team
	TeammateID         uint64
	DeathTick          int
	TeammatePos        [3]float64
	PotentialTraderPos [3]float64
}

// recentKill tracks a recent kill for trade detection.
type recentKill struct {
	VictimID   uint64
	VictimTeam common.Team
	Tick       int
}

// TradeDetector handles trade kill detection logic.
// A trade occurs when a teammate kills the player who killed you within a time window.
type TradeDetector struct {
	recentKills      map[uint64]recentKill
	recentTeamDeaths map[uint64]float64
	pendingTrades    map[uint64][]pendingTrade
}

// NewTradeDetector creates a new TradeDetector with initialized maps.
func NewTradeDetector() *TradeDetector {
	return &TradeDetector{
		recentKills:      make(map[uint64]recentKill),
		recentTeamDeaths: make(map[uint64]float64),
		pendingTrades:    make(map[uint64][]pendingTrade),
	}
}

// Reset clears all trade detection state for a new round.
func (td *TradeDetector) Reset() {
	td.recentKills = make(map[uint64]recentKill)
	td.recentTeamDeaths = make(map[uint64]float64)
	td.pendingTrades = make(map[uint64][]pendingTrade)
}

// TradeResult contains the results of trade detection for a kill event.
type TradeResult struct {
	IsTrade          bool
	TradedPlayerID   uint64
	TradedPlayerName string
	TradeSpeed       float64
	WasOpeningDeath  bool
}

// RecordDeath records a death for potential trade detection.
// Returns pending trade opportunities that were created.
func (td *TradeDetector) RecordDeath(
	victim *common.Player,
	attacker *common.Player,
	currentTick int,
	timeInRound float64,
	participants []*common.Player,
) {
	if victim == nil {
		return
	}

	td.recentTeamDeaths[victim.SteamID64] = timeInRound

	// Create pending trade opportunities for nearby teammates
	if attacker != nil {
		victimPos := victim.Position()
		for _, teammate := range participants {
			if teammate.Team == victim.Team && teammate.IsAlive() && teammate.SteamID64 != victim.SteamID64 {
				teammatePos := teammate.Position()
				dx := victimPos.X - teammatePos.X
				dy := victimPos.Y - teammatePos.Y
				distance := math.Sqrt(dx*dx + dy*dy)

				if distance < rating.TradeProximityUnits {
					pt := pendingTrade{
						KillerID:           attacker.SteamID64,
						KillerTeam:         attacker.Team,
						TeammateID:         teammate.SteamID64,
						DeathTick:          currentTick,
						TeammatePos:        [3]float64{teammatePos.X, teammatePos.Y, teammatePos.Z},
						PotentialTraderPos: [3]float64{teammatePos.X, teammatePos.Y, teammatePos.Z},
					}
					td.pendingTrades[attacker.SteamID64] = append(td.pendingTrades[attacker.SteamID64], pt)
				}
			}
		}
	}
}

// CheckForTrade checks if the current kill is a trade for a previous death.
// Returns trade information if this kill trades a teammate's death.
func (td *TradeDetector) CheckForTrade(
	attacker *common.Player,
	victim *common.Player,
	currentTick int,
	timeInRound float64,
	players map[uint64]*model.PlayerStats,
	rounds map[uint64]*model.RoundStats,
) TradeResult {
	result := TradeResult{}

	if attacker == nil || victim == nil {
		return result
	}

	// Check if this kill trades a recent teammate death
	if recent, ok := td.recentKills[victim.SteamID64]; ok {
		if recent.VictimTeam == attacker.Team && currentTick-recent.Tick <= rating.TradeWindowTicks {
			// This is a trade kill
			if tradedRound, exists := rounds[recent.VictimID]; exists {
				tradedRound.Traded = true
				tradedRound.TradeDeath = true
				tradedRound.SavedByTeammate = true

				// Retroactively refund 30% of the death penalty since the death was traded.
				// A traded death is less impactful because the team recovers the man disadvantage.
				tradeRefund := tradedRound.LastDeathSwing * 0.30
				if tradeRefund < 0 {
					// LastDeathSwing is negative, so refund is negative; we add the positive amount back
					tradedRound.ProbabilitySwing -= tradeRefund
					tradedRound.AddSwingContribution(model.SwingContribution{
						Type:   "trade_refund",
						Amount: -tradeRefund,
						Notes:  "Death traded — penalty reduced",
					})
				}
			}

			if tradedPlayer, exists := players[recent.VictimID]; exists {
				tradedPlayer.TradedDeaths++
				result.TradedPlayerName = tradedPlayer.Name
				result.TradedPlayerID = recent.VictimID

				if tradedRound, exists := rounds[recent.VictimID]; exists {
					if tradedRound.OpeningDeath {
						tradedPlayer.OpeningDeathsTraded++
						result.WasOpeningDeath = true
					}
				}
			}

			result.IsTrade = true
		}
	}

	// Remove pending trades for the victim (they're dead now)
	delete(td.pendingTrades, victim.SteamID64)

	return result
}

// CheckTradeKill checks if the attacker's kill is a trade for their own team.
func (td *TradeDetector) CheckTradeKill(
	attacker *common.Player,
	victim *common.Player,
	currentTick int,
	timeInRound float64,
) (isTradeKill bool, tradeSpeed float64) {
	if attacker == nil || victim == nil {
		return false, 0
	}

	if recent, ok := td.recentKills[victim.SteamID64]; ok {
		if recent.VictimTeam == attacker.Team && currentTick-recent.Tick <= rating.TradeWindowTicks {
			isTradeKill = true
			if deathTime, exists := td.recentTeamDeaths[recent.VictimID]; exists {
				tradeSpeed = timeInRound - deathTime
			}
		}
	}

	return isTradeKill, tradeSpeed
}

// RecordKill records a kill for future trade detection.
func (td *TradeDetector) RecordKill(attacker *common.Player, victim *common.Player, currentTick int) {
	if attacker == nil || victim == nil {
		return
	}

	td.recentKills[attacker.SteamID64] = recentKill{
		VictimID:   victim.SteamID64,
		VictimTeam: victim.Team,
		Tick:       currentTick,
	}
}

// ProcessExpiredTrades checks for expired pending trades and marks them as failed.
// Returns the number of expired trades per killer.
func (td *TradeDetector) ProcessExpiredTrades(
	currentTick int,
	rounds map[uint64]*model.RoundStats,
) map[uint64]int {
	expiredByKiller := make(map[uint64]int)

	for killerID, pendingList := range td.pendingTrades {
		var remainingPending []pendingTrade
		expiredCount := 0

		for _, pt := range pendingList {
			if currentTick-pt.DeathTick > rating.TradeWindowTicks {
				if roundStats, exists := rounds[pt.TeammateID]; exists {
					roundStats.FailedTrades++
				}
				expiredCount++
			} else {
				remainingPending = append(remainingPending, pt)
			}
		}

		if expiredCount > 0 {
			expiredByKiller[killerID] = expiredCount
			if killerRound, exists := rounds[killerID]; exists {
				killerRound.TradeDenials++
			}
		}

		if len(remainingPending) > 0 {
			td.pendingTrades[killerID] = remainingPending
		} else {
			delete(td.pendingTrades, killerID)
		}
	}

	return expiredByKiller
}

// ProcessRoundEndTrades processes any remaining pending trades at round end.
func (td *TradeDetector) ProcessRoundEndTrades(
	currentTick int,
	rounds map[uint64]*model.RoundStats,
) {
	for _, pendingList := range td.pendingTrades {
		for _, pt := range pendingList {
			if currentTick-pt.DeathTick > rating.TradeWindowTicks {
				if roundStats, exists := rounds[pt.TeammateID]; exists {
					roundStats.FailedTrades++
				}
			}
		}
	}
}
