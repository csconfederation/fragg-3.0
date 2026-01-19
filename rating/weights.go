package rating

// === Rating Component Weights ===
// Simplified system: Round Swing captures per-round cost/output factors
// Core stats provide baseline, Round Swing provides the nuanced evaluation
const (
	// Core output metrics (30%)
	WeightKillRating      = 0.12 // Eco-adjusted kills per round
	WeightADRRating       = 0.10 // Damage per round
	WeightMultiKillRating = 0.08 // Multi-kill rating (explosive moments)

	// Core cost metric (20%)
	WeightDeathRating = 0.20 // Death penalty - survival matters

	// K/D Differential (15%) - direct kills minus deaths impact
	WeightKDRating = 0.15

	// Round Swing (35%) - captures per-round evaluation of:
	// - Eco deaths, early deaths, untraded deaths
	// - Trade efficiency, flash assists, utility impact
	// - Clutch performance, weapon economy
	// - Opening duels, bomb plants/defuses
	WeightSwingRating = 0.35
)

// === Normalization Baselines ===
// These represent "average" values used to normalize each component to ~1.0
// Calibrated from actual demo statistics (431 players across all tiers)
const (
	BaselineKPR                = 0.7   // Average kills per round (actual: 0.718)
	BaselineDPR                = 0.7   // Average deaths per round (actual: 0.720)
	BaselineADR                = 75.0  // Average damage per round (actual: 77.6)
	BaselineOpeningKills       = 0.10  // Average opening kills per round (actual: 0.100)
	BaselineMultiKill          = 0.17  // Average multi-kill ratio (actual: 17.4%)
	BaselineAssists            = 0.15  // Average assists per round
	BaselineKAST               = 0.70  // Average KAST percentage (actual: 71.6%)
	BaselineRoundSwing         = 0.00  // Average round swing (zero-sum)
	BaselineOpeningSuccessRate = 0.50  // Average opening duel success rate (50%)
	BaselineTradeKillsPerRound = 0.115 // Average trade kills per round (actual: 0.115)
	BaselineUtilityDamage      = 4.0   // Average utility damage per round (actual: 4.0)
	BaselineFlashAssists       = 0.38  // Average flash assists per round (actual: 0.381)
	BaselineEnemyFlashDur      = 0.9   // Average enemy flash duration per round (actual: 0.92)

	// Cost component baselines
	BaselineEcoDeathValue     = 0.72 // Average eco death value per round
	BaselineEarlyDeaths       = 0.25 // Average early deaths per round (actual: 0.252)
	BaselineUntradedOpenings  = 0.04 // Average untraded opening deaths per round
	BaselineTeamFlashPerRound = 0.24 // Average team flashes per round (actual: 0.247)
	BaselineFailedClutchRate  = 0.60 // Average failed clutch rate (1 - win rate)
)

// === Eco Kill Value Multipliers ===
// Based on equipment ratio (victim/attacker)
const (
	EcoKillPistolVsRifle      = 1.80 // Ratio > 4.0: Pistol kill on full buy
	EcoKillEcoVsForce         = 1.50 // Ratio > 2.0: Eco kill on force/full buy
	EcoKillForceVsFullBuy     = 1.25 // Ratio > 1.3: Force buy vs full buy (~$3000 vs $4500)
	EcoKillSlightDisadvantage = 1.10 // Ratio > 1.1: Slight equipment disadvantage
	EcoKillEqual              = 1.00 // Ratio 0.9-1.1: Roughly equal equipment
	EcoKillSlightAdvantage    = 0.95 // Ratio 0.75-0.9: Slight equipment advantage
	EcoKillAdvantage          = 0.85 // Ratio 0.5-0.75: Equipment advantage
	EcoKillRifleVsPistol      = 0.70 // Ratio < 0.5: Large equipment advantage
)

// === Eco Death Penalty Multipliers ===
// Based on equipment ratio (victim/killer) - higher = more embarrassing death
const (
	EcoDeathToPistol           = 1.60 // Ratio > 4.0: Died to pistol with rifle
	EcoDeathToEco              = 1.40 // Ratio > 2.0: Died to eco on force/full buy
	EcoDeathToForceBuy         = 1.20 // Ratio > 1.3: Full buy died to force buy
	EcoDeathSlightAdvantage    = 1.10 // Ratio > 1.1: Slight equipment advantage
	EcoDeathEqual              = 1.00 // Ratio 0.9-1.1: Roughly equal equipment
	EcoDeathSlightDisadvantage = 0.95 // Ratio 0.75-0.9: Slight equipment disadvantage
	EcoDeathDisadvantage       = 0.85 // Ratio 0.5-0.75: Died to better equipped
	EcoDeathPistolVsRifle      = 0.70 // Ratio < 0.5: Eco vs rifle death
)

// === Equipment Thresholds ===
const (
	MinEquipmentValue = 100.0 // Minimum equipment value to avoid division issues
)

// === Rating Clamp Values ===
const (
	MinRating = 0.20 // Minimum possible rating (HLTV goes to ~0.25)
	MaxRating = 3.00 // Maximum possible rating (allow exceptional performances)
)
