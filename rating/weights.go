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
	BaselineKPR  = 0.7  // Average kills per round
	BaselineDPR  = 0.7  // Average deaths per round
	BaselineADR  = 75.0 // Average damage per round
	BaselineKAST = 0.70 // KAST percentage (Kill/Assist/Survive/Trade)
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
	KPRContribAbove = 0.75 // Multiplier when KPR >= baseline
	KPRContribBelow = 0.55 // Multiplier when KPR < baseline

	// DPR contribution multipliers (asymmetric - penalizes high DPR more)
	DPRContribBelow = 0.15 // Multiplier when DPR <= baseline (good)
	DPRContribAbove = 0.55 // Multiplier when DPR > baseline (bad)

	// ADR contribution multipliers
	ADRContribAbove = 0.015 // Multiplier when ADR >= baseline
	ADRContribBelow = 0.004 // Multiplier when ADR < baseline

	// KAST contribution multipliers
	KASTContribAbove = 0.20 // Multiplier when KAST >= baseline
	KASTContribBelow = 0.35 // Multiplier when KAST < baseline

	// Round swing contribution multipliers
	SwingContribPositive = 0.75 // Multiplier for positive swing
	SwingContribNegative = 1.00 // Multiplier for negative swing

	// Impact contribution weights
	OpeningKillImpactWeight = 0.3   // Weight for opening kills per round
	MultiKillImpactWeight   = 0.15  // Weight for multi-kill rounds per round
	MultiKillContrib        = 0.015 // Multi-kill bonus contribution multiplier
)

// Round swing calculation constants - used in CalculateAdvancedRoundSwing.
const (
	// Base swing values
	SwingBaseWin  = 0.04  // Base swing for winning team
	SwingBaseLoss = -0.04 // Base swing for losing team

	// Swing clamp bounds
	SwingMinClamp = -0.30 // Minimum possible round swing
	SwingMaxClamp = 0.40  // Maximum possible round swing

	// Involvement calculation
	InvolvementAssistWeight   = 0.5   // Weight for assists in involvement
	InvolvementDamageNorm     = 150.0 // Damage normalization divisor
	InvolvementBombWeight     = 1.0   // Weight for bomb plant/defuse
	InvolvementSurvivalWeight = 0.5   // Weight for survival
	InvolvementNormDivisor    = 4.0   // Involvement normalization divisor

	// Performance contribution
	KillContribPerKill       = 0.04  // Base contribution per kill
	KillContribMultiBonus    = 0.02  // Extra contribution per kill after first
	DamageContribDivisor     = 400.0 // Damage contribution divisor
	DamageContribMax         = 0.08  // Maximum damage contribution
	AssistContrib            = 0.015 // Contribution per assist
	FlashAssistContrib       = 0.01  // Contribution per flash assist
	SurvivalContribWin       = 0.02  // Survival contribution on win
	SurvivalContribLoss      = 0.04  // Survival contribution on loss (save)
	DeathPenaltyTraded       = 0.04  // Death penalty if traded
	DeathPenaltyUntraded     = 0.08  // Death penalty if not traded
	ExitFragDamageMultiplier = 0.5   // Damage multiplier for exit frags

	// Situational bonuses
	OpeningKillBonus       = 0.06  // Bonus for opening kill
	OpeningKillPistolBonus = 0.02  // Extra bonus for pistol round opening
	OpeningDeathTraded     = 0.04  // Penalty for traded opening death
	OpeningDeathUntraded   = 0.15  // Penalty for untraded opening death
	EntryFragBonus         = 0.04  // Bonus for entry fragging
	TradeKillBonus         = 0.02  // Bonus for trade kill
	TradeDeathMitigation   = 0.015 // Mitigation for being traded
	TradeDenialBonus       = 0.04  // Bonus per trade denial

	// Round type multipliers
	PistolRoundMultiplier = 1.3 // Multiplier for pistol rounds
	EcoRoundWinMultiplier = 1.4 // Multiplier for eco round wins
	ForceRoundMultiplier  = 1.1 // Multiplier for force buy rounds
	OvertimeMultiplier    = 1.2 // Multiplier for overtime rounds

	// Impact actions
	BombPlantBonus        = 0.08 // Bonus for planting bomb
	BombPlantLateBonus    = 0.02 // Extra bonus for late plant (<30s)
	BombDefuseBonus       = 0.10 // Bonus for defusing bomb
	BombDefuseClutchBonus = 0.03 // Extra bonus for clutch defuse (<10s)
	EcoKillBonus          = 0.04 // Bonus for eco kill
	AntiEcoDeathPenalty   = 0.10 // Penalty for dying to eco

	// Multi-kill bonuses
	MultiKill2KBonus = 0.03 // Bonus for 2K
	MultiKill3KBonus = 0.08 // Bonus for 3K
	MultiKill4KBonus = 0.15 // Bonus for 4K
	MultiKill5KBonus = 0.25 // Bonus for ACE

	// Clutch modifiers
	ClutchWin4KBonus         = 0.20 // Bonus for 1v4+ clutch win
	ClutchWin3KBonus         = 0.15 // Bonus for 1v3 clutch win
	ClutchWin2KBonus         = 0.10 // Bonus for 1v2 clutch win
	ClutchWin1KBonus         = 0.06 // Bonus for 1v1 clutch win
	ClutchLossPenalty        = 0.02 // Penalty for failed clutch
	ClutchLossKillMitigation = 0.02 // Mitigation per kill in failed clutch

	// Economy modifiers
	LowEquipRatioThreshold  = 0.5   // Threshold for low equipment ratio
	HighEquipRatioThreshold = 1.5   // Threshold for high equipment ratio
	LowEquipWinMultiplier   = 1.3   // Multiplier for winning with low equip
	LowEquipKillMultiplier  = 1.2   // Multiplier for kills with low equip
	HighEquipFailMultiplier = 0.8   // Multiplier for failing with high equip
	EcoRoundKillMultiplier  = 1.4   // Multiplier for kills in eco rounds
	ForceRoundModifier      = 1.1   // Modifier for force buy rounds
	MinTeamEquipValue       = 500.0 // Minimum team equipment for ratio calc

	// Utility impact
	UtilityDamageContribRate = 0.03  // Contribution rate per 100 utility damage
	UtilityDamageContribMax  = 0.06  // Maximum utility damage contribution
	FlashDurationContribRate = 0.02  // Contribution rate per 3s flash duration
	FlashDurationContribMax  = 0.04  // Maximum flash duration contribution
	FlashAssistBonusRate     = 0.015 // Bonus per flash assist

	// Trade speed bonuses
	FastTradeThreshold   = 2.0   // Threshold for fast trade (seconds)
	MediumTradeThreshold = 3.0   // Threshold for medium trade
	SlowTradeThreshold   = 5.0   // Threshold for slow trade
	FastTradeBonus       = 0.025 // Bonus for fast trade
	MediumTradeBonus     = 0.015 // Bonus for medium trade
	SlowTradeBonus       = 0.008 // Bonus for slow trade

	// Exit frag penalty
	ExitFragPenaltyRate = 0.02 // Penalty per exit frag

	// Death timing penalties
	EarlyDeathThreshold = 15.0 // Early death threshold (seconds)
	MidDeathThreshold   = 30.0 // Mid death threshold
	LateDeathThreshold  = 60.0 // Late death threshold
	EarlyDeathPenalty   = 0.08 // Penalty for early death
	MidDeathPenalty     = 0.05 // Penalty for mid death
	LateDeathPenalty    = 0.02 // Penalty for late death

	// Team flash penalties
	TeamFlashCountPenalty    = 0.02  // Penalty per team flash
	TeamFlashDurationPenalty = 0.008 // Penalty per second of team flash
	TeamFlashPenaltyMax      = 0.10  // Maximum team flash penalty

	// Failed trade penalty
	FailedTradePenalty = 0.08 // Penalty per failed trade

	// Weapon bonuses
	AWPKillBonusWithSurvive = 0.02  // AWP kill bonus when survived
	AWPKillBonusWithDeath   = 0.005 // AWP kill bonus when died (still got value)
	AWPLostNokillPenalty    = 0.05  // Penalty for losing AWP without kill
	KnifeKillBonus          = 0.03  // Bonus for knife kill
	PistolVsRifleKillBonus  = 0.025 // Bonus for pistol vs rifle kill
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
