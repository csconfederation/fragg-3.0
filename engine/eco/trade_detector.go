package eco

import (
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
)

// pendingTrade is a teammate near a death who could trade its killer.
type pendingTrade struct {
	KillerID           uint64
	KillerTeam         fact.Side
	TeammateID         uint64
	DeathTick          int
	TeammatePos        [3]float64
	PotentialTraderPos [3]float64
}

// TradeDetector tracks pending trades for trade denials: a killer who outlives the trade window
// with teammates of his victim nearby denied the trade. Trades themselves are engine/round's (D22).
type TradeDetector struct {
	pendingTrades map[uint64][]pendingTrade // by killer
}

// NewTradeDetector creates a new TradeDetector with initialized maps.
func NewTradeDetector() *TradeDetector {
	return &TradeDetector{pendingTrades: make(map[uint64][]pendingTrade)}
}

// Reset clears all trade detection state for a new round.
func (td *TradeDetector) Reset() {
	td.pendingTrades = make(map[uint64][]pendingTrade)
}

// RecordDeath records the pending trades a death creates: the victim's alive teammates nearby.
func (td *TradeDetector) RecordDeath(victim, attacker *fact.Player, currentTick int, participants []fact.Player) {
	if victim == nil || attacker == nil {
		return
	}
	victimPos := victim.Pos
	for _, teammate := range participants {
		if teammate.Side == victim.Side && teammate.Alive && teammate.Steam64 != victim.Steam64 {
			teammatePos := teammate.Pos
			dx := victimPos[0] - teammatePos[0]
			dy := victimPos[1] - teammatePos[1]
			if math.Sqrt(dx*dx+dy*dy) < rating.TradeProximityUnits {
				td.pendingTrades[attacker.Steam64] = append(td.pendingTrades[attacker.Steam64], pendingTrade{
					KillerID:           attacker.Steam64,
					KillerTeam:         attacker.Side,
					TeammateID:         teammate.Steam64,
					DeathTick:          currentTick,
					TeammatePos:        teammatePos,
					PotentialTraderPos: teammatePos,
				})
			}
		}
	}
}

// Clear drops a killer's pending trades when an enemy kills him; any other death doesn't trade
// him, so his pending trades still expire as denials.
func (td *TradeDetector) Clear(killer uint64) {
	delete(td.pendingTrades, killer)
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
