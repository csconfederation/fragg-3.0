// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model defines the core data structures for player and round statistics.

package model

// DamageContribution tracks damage dealt by a player to a specific victim.
type DamageContribution struct {
	PlayerID uint64
	Damage   int
}

// FlashAssistInfo tracks flash assist details for a kill.
type FlashAssistInfo struct {
	PlayerID uint64
	Duration float64 // Flash duration in seconds
}

// RoundStats tracks a player's performance within a single round.
// This struct is populated during demo parsing and used to calculate
// per-round metrics like round swing, KAST, and clutch statistics.
type RoundStats struct {
	Kills              int
	Assists            int
	Damage             int
	Survived           bool
	Traded             bool
	GotKill            bool
	GotAssist          bool
	EconImpact         float64
	AWPKills           int
	AWPOpeningKill     bool
	TeamWon            bool
	PlayersAlive       int
	EnemiesAlive       int
	WasLastAlive       bool
	ClutchKills        int
	PlantedBomb        bool
	DefusedBomb        bool
	OpeningKill        bool
	OpeningDeath       bool
	MultiKillRound     int
	EntryFragger       bool
	ClutchAttempt      bool
	ClutchWon          bool
	ClutchSize         int
	SavedWeapons       bool
	EcoKill            bool
	AntiEcoKill        bool
	FlashAssists       int
	TradeKill          bool
	TradeDeath         bool
	FailedTrades       int
	TradeDenials       int
	SavedByTeammate    bool
	SavedTeammate      bool
	IsSupportRound     bool
	InvolvedInOpening  bool
	UtilityDamage      int
	UtilityKills       int
	SmokeDamage        int
	DeathTime          float64
	TimeAlive          float64
	KillTimes          []float64
	TradeSpeed         float64
	IsExitFrag         bool
	ExitFrags          int
	TeamFlashCount     int
	TeamFlashDuration  float64
	FlashesThrown      int
	EnemyFlashDuration float64
	AWPKill            bool
	KnifeKill          bool
	PistolVsRifleKill  bool
	HadAWP             bool
	LostAWP            bool
	IsPistolRound      bool
	PlayerSide         string

	// Probability-based swing tracking (new for v3.0)
	ProbabilitySwing   float64             // Win probability delta contribution
	LastDeathSwing     float64             // Most recent death swing (for trade refund calculation)
	EquipmentValue     float64             // Player's equipment value at round start
	SwingContributions []SwingContribution // Detailed swing events for this round
}

// SwingContribution captures a single event's impact on probability swing.
type SwingContribution struct {
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	TimeInRound   float64 `json:"time_in_round,omitempty"`
	Opponent      string  `json:"opponent,omitempty"`
	Weapon        string  `json:"weapon,omitempty"`
	IsTrade       bool    `json:"is_trade,omitempty"`
	IsHeadshot    bool    `json:"is_headshot,omitempty"`
	EcoMultiplier float64 `json:"eco_multiplier,omitempty"`
	Notes         string  `json:"notes,omitempty"`
}

// AddSwingContribution appends a swing contribution entry for the round.
func (r *RoundStats) AddSwingContribution(contribution SwingContribution) {
	if contribution.Amount == 0 {
		return
	}
	r.SwingContributions = append(r.SwingContributions, contribution)
}

// RoundContext provides contextual information about the round state.
// This is used by the round swing calculator to adjust impact values
// based on round importance, score differential, and game situation.
type RoundContext struct {
	RoundNumber     int
	TotalPlayers    int
	BombPlanted     bool
	BombDefused     bool
	RoundType       string
	TimeRemaining   float64
	IsOvertimeRound bool
	MapSide         string
	TeamScore       int
	EnemyScore      int
	ScoreDiff       int
	IsMatchPoint    bool
	IsCloseGame     bool
	RoundImportance float64
	RoundDecided    bool
	RoundDecidedAt  float64
}
