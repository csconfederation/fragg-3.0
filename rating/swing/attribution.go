package swing

import "math"

// Credit attribution constants for kill swing distribution.
const (
	// KillerBaseCredit is the base credit given to the player who got the kill.
	KillerBaseCredit = 0.60

	// DamageShareCredit is the portion of credit distributed based on damage share.
	DamageShareCredit = 0.25

	// FlashAssistMaxCredit is the maximum credit for flash assists.
	FlashAssistMaxCredit = 0.15

	// TradeKillPenalty reduces credit for trade kills (kill was "expected").
	TradeKillPenalty = 0.30

	// PlantCreditShare is the portion of plant swing given to the planter.
	PlantCreditShare = 0.60

	// DefuseCreditShare is the portion of defuse swing given to the defuser.
	DefuseCreditShare = 0.80

	// SavePenalty is the small negative swing for surviving a lost round.
	SavePenalty = 0.02

	// ClutchBonusMultiplier scales additional clutch bonuses.
	ClutchBonusMultiplier = 1.0
)

// Clutch bonus values for winning clutches (added on top of probability swing).
var ClutchBonuses = map[int]float64{
	1: 0.02, // 1v1 clutch
	2: 0.04, // 1v2 clutch
	3: 0.08, // 1v3 clutch
	4: 0.15, // 1v4 clutch
	5: 0.25, // 1v5 clutch
}

// Attributor handles the distribution of kill credit among contributors.
type Attributor struct{}

// NewAttributor creates a new Attributor.
func NewAttributor() *Attributor {
	return &Attributor{}
}

// AttributeKillCredit distributes swing credit for a kill among all contributors.
func (a *Attributor) AttributeKillCredit(
	playerSwing map[uint64]float64,
	kill *KillEvent,
	delta float64,
) {
	// Calculate killer's credit and determine shareable pool for helpers
	killerCredit := a.calculateKillerCredit(kill)
	if kill.IsTradeKill {
		killerCredit *= (1.0 - TradeKillPenalty)
	}

	if killerCredit < 0 {
		killerCredit = 0
	}
	if killerCredit > 1 {
		killerCredit = 1
	}

	shareable := delta * (1.0 - killerCredit)
	remainingShareable := shareable

	allocated := a.attributeDamageContributors(playerSwing, kill, delta, &remainingShareable)
	allocated += a.attributeFlashAssists(playerSwing, kill, delta, &remainingShareable)

	if allocated > shareable {
		allocated = shareable
	}

	killerShare := delta - allocated
	if killerShare < 0 {
		killerShare = 0
	}
	playerSwing[kill.KillerID] += killerShare
}

// calculateKillerCredit computes the killer's share of credit.
func (a *Attributor) calculateKillerCredit(kill *KillEvent) float64 {
	credit := KillerBaseCredit

	// Add damage share bonus if killer dealt significant damage
	if kill.TotalDamageToVictim > 0 && kill.KillerDamageDealt > 0 {
		damageRatio := float64(kill.KillerDamageDealt) / float64(kill.TotalDamageToVictim)
		credit += DamageShareCredit * damageRatio
	}

	return credit
}

// attributeDamageContributors distributes credit to players who dealt damage.
func (a *Attributor) attributeDamageContributors(
	playerSwing map[uint64]float64,
	kill *KillEvent,
	delta float64,
	shareable *float64,
) float64 {
	allocated := 0.0

	if kill.TotalDamageToVictim <= 0 || len(kill.DamageContributors) == 0 {
		return 0
	}

	for _, contributor := range kill.DamageContributors {
		// Skip the killer (they already got credit)
		if contributor.PlayerID == kill.KillerID {
			continue
		}

		// Calculate damage share
		share := float64(contributor.Damage) / float64(kill.TotalDamageToVictim)

		desired := delta * DamageShareCredit * share
		alloc := allocateShare(desired, shareable)
		if alloc <= 0 {
			break
		}
		playerSwing[contributor.PlayerID] += alloc
		allocated += alloc
	}

	return allocated
}

// attributeFlashAssists distributes credit to players who flash assisted.
func (a *Attributor) attributeFlashAssists(
	playerSwing map[uint64]float64,
	kill *KillEvent,
	delta float64,
	shareable *float64,
) float64 {
	allocated := 0.0

	for _, flash := range kill.FlashAssists {
		// Flash credit scales with duration (up to 3 seconds for full credit)
		flashCredit := math.Min(flash.Duration/3.0, 1.0) * FlashAssistMaxCredit
		desired := delta * flashCredit
		alloc := allocateShare(desired, shareable)
		if alloc <= 0 {
			break
		}
		playerSwing[flash.PlayerID] += alloc
		allocated += alloc
	}

	return allocated
}

// allocateShare deducts from the remaining shareable pool and returns the allocated amount.
func allocateShare(desired float64, shareable *float64) float64 {
	if desired <= 0 || shareable == nil || *shareable <= 0 {
		return 0
	}
	alloc := math.Min(desired, *shareable)
	*shareable -= alloc
	return alloc
}

// AddClutchBonus adds additional bonus for winning a clutch.
func (a *Attributor) AddClutchBonus(
	playerSwing map[uint64]float64,
	clutcherID uint64,
	clutchSize int,
	won bool,
) {
	if !won || clutchSize < 1 {
		return
	}

	bonus, ok := ClutchBonuses[clutchSize]
	if !ok {
		// For clutches larger than 5 (shouldn't happen), use 1v5 bonus
		bonus = ClutchBonuses[5]
	}

	playerSwing[clutcherID] += bonus * ClutchBonusMultiplier
}

// DistributeRemainingSwing distributes any remaining probability to contributors.
// Used when players save and the round ends without reaching 0% or 100%.
func (a *Attributor) DistributeRemainingSwing(
	playerSwing map[uint64]float64,
	contributors map[uint64]float64, // PlayerID -> contribution weight
	remainingProb float64,
	shareMultiplier float64,
) {
	// Normalize contribution weights
	totalWeight := 0.0
	for _, weight := range contributors {
		totalWeight += weight
	}

	if totalWeight <= 0 {
		return
	}

	// Distribute proportionally
	for playerID, weight := range contributors {
		share := (weight / totalWeight) * remainingProb * shareMultiplier
		playerSwing[playerID] += share
	}
}
