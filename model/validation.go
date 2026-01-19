// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model defines the core data structures for player and round statistics.
// This file provides validation methods for detecting data inconsistencies
// and ensuring data integrity in player and round statistics.
package model

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation issue found in the data.
type ValidationError struct {
	Field   string
	Message string
}

// ValidationResult contains the results of a validation check.
type ValidationResult struct {
	IsValid bool
	Errors  []ValidationError
}

// Error returns a string representation of all validation errors.
func (v *ValidationResult) Error() string {
	if v.IsValid {
		return ""
	}
	var msgs []string
	for _, e := range v.Errors {
		msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
	}
	return strings.Join(msgs, "; ")
}

// AddError adds a validation error to the result.
func (v *ValidationResult) AddError(field, message string) {
	v.IsValid = false
	v.Errors = append(v.Errors, ValidationError{Field: field, Message: message})
}

// Validate checks PlayerStats for data inconsistencies.
// Returns a ValidationResult with any issues found.
func (p *PlayerStats) Validate() *ValidationResult {
	result := &ValidationResult{IsValid: true}

	// Basic sanity checks
	if p.RoundsPlayed < 0 {
		result.AddError("RoundsPlayed", "cannot be negative")
	}
	if p.Kills < 0 {
		result.AddError("Kills", "cannot be negative")
	}
	if p.Deaths < 0 {
		result.AddError("Deaths", "cannot be negative")
	}
	if p.Damage < 0 {
		result.AddError("Damage", "cannot be negative")
	}

	// Logical consistency checks
	if p.RoundsPlayed > 0 {
		if p.Deaths > p.RoundsPlayed {
			result.AddError("Deaths", fmt.Sprintf("cannot exceed rounds played (%d > %d)", p.Deaths, p.RoundsPlayed))
		}
	}

	// Side stats consistency
	totalSideRounds := p.TRoundsPlayed + p.CTRoundsPlayed
	if totalSideRounds > 0 && totalSideRounds != p.RoundsPlayed {
		result.AddError("SideRounds", fmt.Sprintf("T+CT rounds (%d) != total rounds (%d)", totalSideRounds, p.RoundsPlayed))
	}

	totalSideKills := p.TKills + p.CTKills
	if totalSideKills != p.Kills {
		result.AddError("SideKills", fmt.Sprintf("T+CT kills (%d) != total kills (%d)", totalSideKills, p.Kills))
	}

	totalSideDeaths := p.TDeaths + p.CTDeaths
	if totalSideDeaths != p.Deaths {
		result.AddError("SideDeaths", fmt.Sprintf("T+CT deaths (%d) != total deaths (%d)", totalSideDeaths, p.Deaths))
	}

	// Opening stats consistency
	if p.OpeningKills > p.Kills {
		result.AddError("OpeningKills", fmt.Sprintf("cannot exceed total kills (%d > %d)", p.OpeningKills, p.Kills))
	}
	if p.OpeningDeaths > p.Deaths {
		result.AddError("OpeningDeaths", fmt.Sprintf("cannot exceed total deaths (%d > %d)", p.OpeningDeaths, p.Deaths))
	}

	// Clutch stats consistency
	if p.ClutchWins > p.ClutchRounds {
		result.AddError("ClutchWins", fmt.Sprintf("cannot exceed clutch rounds (%d > %d)", p.ClutchWins, p.ClutchRounds))
	}
	if p.Clutch1v1Wins > p.Clutch1v1Attempts {
		result.AddError("Clutch1v1Wins", fmt.Sprintf("cannot exceed attempts (%d > %d)", p.Clutch1v1Wins, p.Clutch1v1Attempts))
	}

	// AWP stats consistency
	if p.AWPKills > p.Kills {
		result.AddError("AWPKills", fmt.Sprintf("cannot exceed total kills (%d > %d)", p.AWPKills, p.Kills))
	}

	// Multi-kill consistency
	totalMultiKillRounds := p.MultiKillsRaw[2] + p.MultiKillsRaw[3] + p.MultiKillsRaw[4] + p.MultiKillsRaw[5]
	if totalMultiKillRounds > p.RoundsPlayed {
		result.AddError("MultiKills", fmt.Sprintf("multi-kill rounds (%d) exceed total rounds (%d)", totalMultiKillRounds, p.RoundsPlayed))
	}

	// Rating bounds check
	if p.HLTVRating < 0 {
		result.AddError("HLTVRating", "cannot be negative")
	}
	if p.FinalRating < 0 {
		result.AddError("FinalRating", "cannot be negative")
	}

	// Percentage bounds check
	if p.KAST < 0 || p.KAST > 1 {
		result.AddError("KAST", fmt.Sprintf("must be between 0 and 1, got %.2f", p.KAST))
	}
	if p.Survival < 0 || p.Survival > 1 {
		result.AddError("Survival", fmt.Sprintf("must be between 0 and 1, got %.2f", p.Survival))
	}

	return result
}

// Validate checks RoundStats for data inconsistencies.
// Returns a ValidationResult with any issues found.
func (r *RoundStats) Validate() *ValidationResult {
	result := &ValidationResult{IsValid: true}

	// Basic sanity checks
	if r.Kills < 0 {
		result.AddError("Kills", "cannot be negative")
	}
	if r.Kills > 5 {
		result.AddError("Kills", fmt.Sprintf("cannot exceed 5 in a round, got %d", r.Kills))
	}
	if r.Assists < 0 {
		result.AddError("Assists", "cannot be negative")
	}
	if r.Damage < 0 {
		result.AddError("Damage", "cannot be negative")
	}
	if r.Damage > 500 {
		result.AddError("Damage", fmt.Sprintf("unusually high damage: %d", r.Damage))
	}

	// Death timing consistency
	if r.DeathTime < 0 {
		result.AddError("DeathTime", "cannot be negative")
	}
	if r.Survived && r.DeathTime > 0 {
		result.AddError("DeathTime", "player survived but has death time set")
	}

	// Clutch consistency
	if r.ClutchWon && !r.ClutchAttempt {
		result.AddError("ClutchWon", "clutch won without clutch attempt")
	}
	if r.ClutchKills > 0 && !r.ClutchAttempt {
		result.AddError("ClutchKills", "clutch kills without clutch attempt")
	}

	// Opening consistency
	if r.OpeningKill && r.OpeningDeath {
		result.AddError("Opening", "cannot have both opening kill and opening death")
	}

	// AWP consistency
	if r.AWPKills > r.Kills {
		result.AddError("AWPKills", fmt.Sprintf("cannot exceed total kills (%d > %d)", r.AWPKills, r.Kills))
	}

	// Side validation
	if r.PlayerSide != "" && r.PlayerSide != "T" && r.PlayerSide != "CT" {
		result.AddError("PlayerSide", fmt.Sprintf("invalid side: %s", r.PlayerSide))
	}

	return result
}

// ValidateConsistency checks that RoundStats is consistent with the round context.
func (r *RoundStats) ValidateConsistency(ctx *RoundContext) *ValidationResult {
	result := &ValidationResult{IsValid: true}

	if ctx == nil {
		return result
	}

	// Bomb event consistency
	if r.DefusedBomb && r.PlayerSide == "T" {
		result.AddError("DefusedBomb", "T-side player cannot defuse bomb")
	}
	if r.PlantedBomb && r.PlayerSide == "CT" {
		result.AddError("PlantedBomb", "CT-side player cannot plant bomb")
	}

	return result
}
