// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package rating implements the eco-rating calculation system.
// This file defines all constants used in rating calculations, including:
// - Component weights for the final rating formula
// - Baseline values for normalization
// - Economic kill/death multipliers
// - Rating bounds
package rating

// Baseline values represent average/expected performance levels.
// These are used to normalize metrics so that average performance = 1.0 contribution.
const (
	BaselineKPR  = 0.72 // Average kills per round
	BaselineDPR  = 0.68 // Average deaths per round
	BaselineADR  = 77.0 // Average damage per round
	BaselineKAST = 0.72 // KAST percentage (Kill/Assist/Survive/Trade)
)

// Economic kill value multipliers - rewards kills against better-equipped opponents.
// Higher values mean the kill is worth more to the rating.
const (
	EcoKillPistolVsRifle      = 1.80 // Pistol killing rifle (huge disadvantage)
	EcoKillEcoVsForce         = 1.50 // Eco killing force/full buy
	EcoKillForceVsFullBuy     = 1.25 // Force buy killing full buy
	EcoKillSlightDisadvantage = 1.10 // Slight equipment disadvantage
	EcoKillEqual              = 1.00 // Equal equipment
	EcoKillSlightAdvantage    = 0.95 // Slight equipment advantage
	EcoKillAdvantage          = 0.85 // Clear equipment advantage
	EcoKillRifleVsPistol      = 0.70 // Rifle killing pistol (expected)
)

// Economic death penalty multipliers - penalizes deaths to worse-equipped opponents.
// Higher values mean the death hurts the rating more.
const (
	EcoDeathToPistol           = 1.60 // Rifle dying to pistol (embarrassing)
	EcoDeathToEco              = 1.40 // Full buy dying to eco
	EcoDeathToForceBuy         = 1.20 // Full buy dying to force
	EcoDeathSlightAdvantage    = 1.10 // Slight advantage, still died
	EcoDeathEqual              = 1.00 // Equal equipment death
	EcoDeathSlightDisadvantage = 0.95 // Slight disadvantage death
	EcoDeathDisadvantage       = 0.85 // Clear disadvantage death
	EcoDeathPistolVsRifle      = 0.70 // Pistol dying to rifle (expected)
)

// Minimum equipment value to prevent division by zero in ratio calculations.
const (
	MinEquipmentValue = 100.0
)

// Rating bounds - final ratings are clamped to this range.
const (
	MinRating = 0.20 // Minimum possible rating
	MaxRating = 3.00 // Maximum possible rating
)

// HLTV 2.0 Rating constants - derived from professional match analysis.
// These are used to calculate the standard HLTV rating for comparison.
const (
	HLTVBaselineKPR    = 0.679 // Average kills per round in pro matches
	HLTVBaselineSPR    = 0.317 // Average survival rate per round
	HLTVBaselineRMK    = 1.277 // Average round multi-kill points
	HLTVSurvivalWeight = 0.7   // Weight for survival component
	HLTVRatingDivisor  = 2.7   // Final rating divisor
)

// Rating formula contribution multipliers - control how much each stat
// affects the final rating above/below baseline.
const (
	RatingBaseline = 1.0 // Starting point for rating calculation

	// KPR contribution multipliers (asymmetric - rewards high KPR more)
	KPRContribAbove = 0.35 // Multiplier when KPR >= baseline
	KPRContribBelow = 0.30 // Multiplier when KPR < baseline

	// DPR contribution multipliers (asymmetric - penalizes high DPR more)
	DPRContribBelow = 0.08 // Multiplier when DPR <= baseline (good)
	DPRContribAbove = 0.25 // Multiplier when DPR > baseline (bad)

	// ADR contribution multipliers
	ADRContribAbove = 0.005 // Multiplier when ADR >= baseline
	ADRContribBelow = 0.004 // Multiplier when ADR < baseline

	// KAST contribution multipliers
	KASTContribAbove = 0.20 // Multiplier when KAST >= baseline
	KASTContribBelow = 0.25 // Multiplier when KAST < baseline

	// Round swing contribution multipliers
	SwingContribPositive = 1.40 // Multiplier for positive swing
	SwingContribNegative = 1.40 // Multiplier for negative swing

	ProbSwingContribMultiplier = 2.5

	// Impact contribution weights
	OpeningKillImpactWeight = 0.15  // Weight for opening kills per round
	MultiKillImpactWeight   = 0.08  // Weight for multi-kill rounds per round
	MultiKillContrib        = 0.005 // Multi-kill bonus contribution multiplier
)

// Trade detection constants - used in handlers.go for trade calculations.
const (
	TradeWindowTicks    = 320    // Trade window in ticks (5 seconds at 64 tick)
	TradeProximityUnits = 1200.0 // Maximum distance for trade opportunity (units)
)

// Round context constants - used for round importance calculations.
const (
	LateRoundTimeThreshold = 30.0 // Time threshold for late bomb plant (seconds)
	ClutchDefuseThreshold  = 10.0 // Time threshold for clutch defuse (seconds)
)

// Round structure constants - CS2 MR12 format.
const (
	FirstHalfPistolRound  = 1  // First pistol round of the match
	SecondHalfPistolRound = 13 // Second half pistol round (MR12)
	RoundsPerHalf         = 12 // Rounds per half in regulation
	RegulationRounds      = 24 // Total regulation rounds (MR12)
	OvertimeLength        = 6  // Rounds per overtime (MR3)
	TickRate              = 64 // Server tick rate for time calculations
)

// IsPistolRound determines if a round number is a pistol round.
// Handles regulation and overtime pistol rounds for MR12 format.
func IsPistolRound(roundNumber int) bool {
	if roundNumber == FirstHalfPistolRound || roundNumber == SecondHalfPistolRound {
		return true
	}
	// Overtime pistol rounds: 25, 31, 37, etc.
	if roundNumber > RegulationRounds && (roundNumber-RegulationRounds-1)%OvertimeLength == 0 {
		return true
	}
	return false
}
