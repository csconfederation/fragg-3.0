package parser

import (
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	// SurvivalCreditShare is the fraction of a teammate's kill swing
	// credited to each alive advantage creator on the same team.
	// E.g., if a player created a man advantage and a teammate gets a kill
	// while that advantage persists, the creator earns 15% of the kill swing.
	SurvivalCreditShare = 0.15
)

// AdvantageSlot represents a man advantage created by a player's kill.
// The slot persists until neutralized by a teammate's death or the creator dies.
type AdvantageSlot struct {
	PlayerID uint64
	Side     common.Team
}

// AdvantageTracker tracks man advantages created by kills within a round.
// When a player gets a kill, they create an advantage slot on their team.
// While that advantage persists, subsequent teammate kills generate
// survival credit for the advantage creator.
type AdvantageTracker struct {
	// Advantage slots per team, ordered FIFO (oldest first)
	tSlots  []AdvantageSlot
	ctSlots []AdvantageSlot
}

// NewAdvantageTracker creates a new tracker.
func NewAdvantageTracker() *AdvantageTracker {
	return &AdvantageTracker{
		tSlots:  make([]AdvantageSlot, 0),
		ctSlots: make([]AdvantageSlot, 0),
	}
}

// Reset clears all advantage slots for a new round.
func (at *AdvantageTracker) Reset() {
	at.tSlots = make([]AdvantageSlot, 0)
	at.ctSlots = make([]AdvantageSlot, 0)
}

// RecordKill adds an advantage slot for the killer's team.
// Returns the list of alive advantage creators on the killer's team
// (excluding the killer themselves) who should receive survival credit.
func (at *AdvantageTracker) RecordKill(killerID uint64, killerSide common.Team) []uint64 {
	// Collect alive advantage creators on the killer's team BEFORE adding the new slot.
	// These players created prior advantages that are still active — the new kill
	// happened while their advantage persisted, so they earn survival credit.
	slots := at.getSlots(killerSide)
	survivalBeneficiaries := make([]uint64, 0)
	seen := make(map[uint64]bool)
	for _, slot := range slots {
		if slot.PlayerID != killerID && !seen[slot.PlayerID] {
			survivalBeneficiaries = append(survivalBeneficiaries, slot.PlayerID)
			seen[slot.PlayerID] = true
		}
	}

	// Add the new advantage slot for the killer
	at.addSlot(killerSide, AdvantageSlot{
		PlayerID: killerID,
		Side:     killerSide,
	})

	return survivalBeneficiaries
}

// RecordDeath consumes the oldest advantage slot on the victim's team.
// When a player dies, the enemy team neutralizes one man advantage.
// Also removes any slots owned by the dying player (they can't benefit from survival anymore).
func (at *AdvantageTracker) RecordDeath(victimID uint64, victimSide common.Team) {
	// Remove the oldest advantage slot on the victim's team (enemy neutralized it)
	slots := at.getSlots(victimSide)
	if len(slots) > 0 {
		at.setSlots(victimSide, slots[1:])
	}

	// Also remove any remaining slots owned by the dying player
	// (they can no longer earn survival credit)
	at.removePlayerSlots(victimID, victimSide)
}

// getSlots returns the advantage slots for a team.
func (at *AdvantageTracker) getSlots(side common.Team) []AdvantageSlot {
	if side == common.TeamTerrorists {
		return at.tSlots
	}
	return at.ctSlots
}

// setSlots sets the advantage slots for a team.
func (at *AdvantageTracker) setSlots(side common.Team, slots []AdvantageSlot) {
	if side == common.TeamTerrorists {
		at.tSlots = slots
	} else {
		at.ctSlots = slots
	}
}

// addSlot adds an advantage slot for a team.
func (at *AdvantageTracker) addSlot(side common.Team, slot AdvantageSlot) {
	if side == common.TeamTerrorists {
		at.tSlots = append(at.tSlots, slot)
	} else {
		at.ctSlots = append(at.ctSlots, slot)
	}
}

// removePlayerSlots removes all slots owned by a specific player.
func (at *AdvantageTracker) removePlayerSlots(playerID uint64, side common.Team) {
	slots := at.getSlots(side)
	filtered := make([]AdvantageSlot, 0, len(slots))
	for _, slot := range slots {
		if slot.PlayerID != playerID {
			filtered = append(filtered, slot)
		}
	}
	at.setSlots(side, filtered)
}
