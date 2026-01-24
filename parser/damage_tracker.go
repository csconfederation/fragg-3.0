package parser

import (
	"eco-rating/rating/swing"
)

const (
	// EngagementTimeout is the time in seconds after which an engagement is considered ended.
	// If no damage is dealt for this duration, the next damage starts a new engagement.
	EngagementTimeout = 5.0
)

// DamageTracker tracks damage dealt to each player during a round.
// Used to attribute kill credit to damage contributors.
type DamageTracker struct {
	// damageDealt maps victim SteamID -> attacker SteamID -> damage
	damageDealt map[uint64]map[uint64]int

	// firstDamageTime maps victim SteamID -> attacker SteamID -> time of first damage in current engagement
	firstDamageTime map[uint64]map[uint64]float64

	// lastDamageTime maps victim SteamID -> attacker SteamID -> time of last damage (for timeout)
	lastDamageTime map[uint64]map[uint64]float64

	// flashedPlayers maps victim SteamID -> list of flash assists
	flashedPlayers map[uint64][]FlashInfo
}

// FlashInfo tracks flash duration on a victim from an attacker.
type FlashInfo struct {
	AttackerID uint64
	Duration   float64
}

// NewDamageTracker creates a new damage tracker.
func NewDamageTracker() *DamageTracker {
	return &DamageTracker{
		damageDealt:     make(map[uint64]map[uint64]int),
		firstDamageTime: make(map[uint64]map[uint64]float64),
		lastDamageTime:  make(map[uint64]map[uint64]float64),
		flashedPlayers:  make(map[uint64][]FlashInfo),
	}
}

// Reset clears all tracking data for a new round.
func (dt *DamageTracker) Reset() {
	dt.damageDealt = make(map[uint64]map[uint64]int)
	dt.firstDamageTime = make(map[uint64]map[uint64]float64)
	dt.lastDamageTime = make(map[uint64]map[uint64]float64)
	dt.flashedPlayers = make(map[uint64][]FlashInfo)
}

// RecordDamage records damage dealt from attacker to victim.
func (dt *DamageTracker) RecordDamage(attackerID, victimID uint64, damage int, timeInRound float64) {
	if dt.damageDealt[victimID] == nil {
		dt.damageDealt[victimID] = make(map[uint64]int)
	}
	dt.damageDealt[victimID][attackerID] += damage

	// Initialize maps if needed
	if dt.firstDamageTime[victimID] == nil {
		dt.firstDamageTime[victimID] = make(map[uint64]float64)
	}
	if dt.lastDamageTime[victimID] == nil {
		dt.lastDamageTime[victimID] = make(map[uint64]float64)
	}

	// Check if this is a new engagement (first damage or timeout exceeded)
	lastTime, exists := dt.lastDamageTime[victimID][attackerID]
	if !exists || (timeInRound-lastTime) > EngagementTimeout {
		// New engagement - reset first damage time
		dt.firstDamageTime[victimID][attackerID] = timeInRound
	}

	// Always update last damage time
	dt.lastDamageTime[victimID][attackerID] = timeInRound
}

// RecordFlash records that an attacker flashed a victim.
func (dt *DamageTracker) RecordFlash(attackerID, victimID uint64, duration float64) {
	dt.flashedPlayers[victimID] = append(dt.flashedPlayers[victimID], FlashInfo{
		AttackerID: attackerID,
		Duration:   duration,
	})
}

// GetDamageContributors returns all players who damaged a victim and their damage totals.
func (dt *DamageTracker) GetDamageContributors(victimID uint64) []swing.DamageContributor {
	contributors := make([]swing.DamageContributor, 0)

	if damages, ok := dt.damageDealt[victimID]; ok {
		for attackerID, damage := range damages {
			contributors = append(contributors, swing.DamageContributor{
				PlayerID: attackerID,
				Damage:   damage,
			})
		}
	}

	return contributors
}

// GetTotalDamageToVictim returns the total damage dealt to a victim.
func (dt *DamageTracker) GetTotalDamageToVictim(victimID uint64) int {
	total := 0
	if damages, ok := dt.damageDealt[victimID]; ok {
		for _, damage := range damages {
			total += damage
		}
	}
	return total
}

// GetKillerDamage returns the damage the killer dealt to the victim.
func (dt *DamageTracker) GetKillerDamage(killerID, victimID uint64) int {
	if damages, ok := dt.damageDealt[victimID]; ok {
		return damages[killerID]
	}
	return 0
}

// GetFlashAssists returns flash assists for a victim's death.
// Only returns recent flashes (within the flash duration window).
func (dt *DamageTracker) GetFlashAssists(victimID uint64) []swing.FlashAssist {
	assists := make([]swing.FlashAssist, 0)

	if flashes, ok := dt.flashedPlayers[victimID]; ok {
		for _, flash := range flashes {
			// Only count flashes with significant duration
			if flash.Duration >= 0.5 {
				assists = append(assists, swing.FlashAssist{
					PlayerID: flash.AttackerID,
					Duration: flash.Duration,
				})
			}
		}
	}

	return assists
}

// ClearVictimData removes tracking data for a dead player.
// Called after processing a kill to prevent double-counting.
func (dt *DamageTracker) ClearVictimData(victimID uint64) {
	delete(dt.damageDealt, victimID)
	delete(dt.firstDamageTime, victimID)
	delete(dt.lastDamageTime, victimID)
	delete(dt.flashedPlayers, victimID)
}

// GetTimeToKill returns the time between first damage and kill time.
// Returns -1 if no prior damage was recorded (e.g., one-shot kill).
func (dt *DamageTracker) GetTimeToKill(killerID, victimID uint64, killTime float64) float64 {
	if times, ok := dt.firstDamageTime[victimID]; ok {
		if firstTime, exists := times[killerID]; exists {
			return killTime - firstTime
		}
	}
	return -1
}
