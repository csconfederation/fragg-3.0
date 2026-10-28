package eco

import (
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
)

// pendingTrade is a teammate near a death who could trade its killer.
type pendingTrade struct {
	DeathTick int
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
					DeathTick: currentTick,
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

// ProcessExpiredTrades drops pending trades older than the trade window; each killer whose kill
// went untraded gets a trade denial.
func (td *TradeDetector) ProcessExpiredTrades(currentTick int, rounds map[uint64]*model.RoundStats) {
	for killerID, pendingList := range td.pendingTrades {
		var remainingPending []pendingTrade
		expiredCount := 0

		for _, pt := range pendingList {
			if currentTick-pt.DeathTick > rating.TradeWindowTicks {
				expiredCount++
			} else {
				remainingPending = append(remainingPending, pt)
			}
		}

		if expiredCount > 0 {
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
}
