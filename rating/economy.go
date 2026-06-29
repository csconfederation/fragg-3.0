// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package rating implements the eco-rating calculation system.
// This file contains functions for computing economic kill values and death penalties
// based on equipment value ratios between attacker and victim.
package rating

// EcoKillValue calculates the economic value multiplier for a kill.
// Kills against better-equipped opponents are worth more (up to 1.80x),
// while kills against worse-equipped opponents are worth less (down to 0.70x).
// This rewards players who perform well in disadvantaged situations.
func EcoKillValue(attackerEquip, victimEquip float64) float64 {
	if attackerEquip < MinEquipmentValue {
		attackerEquip = MinEquipmentValue
	}

	ratio := victimEquip / attackerEquip

	if ratio > 4.0 {
		return EcoKillPistolVsRifle
	} else if ratio > 2.0 {
		return EcoKillEcoVsForce
	} else if ratio > 1.3 {
		return EcoKillForceVsFullBuy
	} else if ratio > 1.1 {
		return EcoKillSlightDisadvantage
	} else if ratio > 0.9 {
		return EcoKillEqual
	} else if ratio > 0.75 {
		return EcoKillSlightAdvantage
	} else if ratio > 0.5 {
		return EcoKillAdvantage
	} else {
		return EcoKillRifleVsPistol
	}
}

// EcoDeathPenalty calculates the penalty multiplier for a death.
// Dying to worse-equipped opponents incurs a higher penalty (up to 1.60x),
// while dying to better-equipped opponents has a reduced penalty (down to 0.70x).
// This penalizes players who die in advantaged situations.
func EcoDeathPenalty(victimEquip, killerEquip float64) float64 {
	if killerEquip < MinEquipmentValue {
		killerEquip = MinEquipmentValue
	}
	ratio := victimEquip / killerEquip

	if ratio > 4.0 {
		return EcoDeathToPistol
	} else if ratio > 2.0 {
		return EcoDeathToEco
	} else if ratio > 1.3 {
		return EcoDeathToForceBuy
	} else if ratio > 1.1 {
		return EcoDeathSlightAdvantage
	} else if ratio > 0.9 {
		return EcoDeathEqual
	} else if ratio > 0.75 {
		return EcoDeathSlightDisadvantage
	} else if ratio > 0.5 {
		return EcoDeathDisadvantage
	} else {
		return EcoDeathPistolVsRifle
	}
}
