// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model defines the core data structures for player and round statistics.
// This file contains embedded struct types used to compose PlayerStats,
// organizing related statistics into logical groups for better maintainability.
package model

// CoreStats contains fundamental match statistics.
type CoreStats struct {
	RoundsPlayed int `json:"rounds_played"`
	RoundsWon    int `json:"rounds_won"`
	RoundsLost   int `json:"rounds_lost"`
	Kills        int `json:"kills"`
	Assists      int `json:"assists"`
	Deaths       int `json:"deaths"`
	Damage       int `json:"damage"`
}

// OpeningStats tracks first-kill/first-death statistics.
type OpeningStats struct {
	OpeningKills        int `json:"opening_kills"`
	OpeningDeaths       int `json:"opening_deaths"`
	OpeningDeathsTraded int `json:"opening_deaths_traded"`
	OpeningAttempts     int `json:"opening_attempts"`
	OpeningSuccesses    int `json:"opening_successes"`
}

// ClutchStats tracks clutch situation performance.
type ClutchStats struct {
	ClutchRounds      int `json:"clutch_rounds"`
	ClutchWins        int `json:"clutch_wins"`
	Clutch1v1Attempts int `json:"clutch_1v1_attempts"`
	Clutch1v1Wins     int `json:"clutch_1v1_wins"`
	LastAliveRounds   int `json:"last_alive_rounds"`
}

// TradeStats tracks trade-related statistics.
type TradeStats struct {
	TradeDenials    int `json:"trade_denials"`
	TradedDeaths    int `json:"traded_deaths"`
	TradeKills      int `json:"trade_kills"`
	FastTrades      int `json:"fast_trades"`
	SavedByTeammate int `json:"saved_by_teammate"`
	SavedTeammate   int `json:"saved_teammate"`
}

// AWPStats tracks AWP-specific performance.
type AWPStats struct {
	AWPKills           int `json:"awp_kills"`
	RoundsWithAWPKill  int `json:"rounds_with_awp_kill"`
	AWPMultiKillRounds int `json:"awp_multi_kill_rounds"`
	AWPOpeningKills    int `json:"awp_opening_kills"`
	AWPDeaths          int `json:"awp_deaths"`
	AWPDeathsNoKill    int `json:"awp_deaths_no_kill"`
}

// UtilityStats tracks grenade and flash usage.
type UtilityStats struct {
	UtilityDamage      int     `json:"utility_damage"`
	UtilityKills       int     `json:"utility_kills"`
	FlashesThrown      int     `json:"flashes_thrown"`
	FlashAssists       int     `json:"flash_assists"`
	EnemyFlashDuration float64 `json:"-"`
	TeamFlashCount     int     `json:"team_flash_count"`
	TeamFlashDuration  float64 `json:"-"`
}

// EconomyStats tracks economic impact metrics.
type EconomyStats struct {
	EcoKillValue          float64 `json:"eco_kill_value"`
	EcoDeathValue         float64 `json:"eco_death_value"`
	EconImpact            float64 `json:"econ_impact"`
	LowBuyKills           int     `json:"low_buy_kills"`
	DisadvantagedBuyKills int     `json:"disadvantaged_buy_kills"`
}

// CombatStats tracks combat-related metrics.
type CombatStats struct {
	PerfectKills        int `json:"perfect_kills"`
	RoundsWithKill      int `json:"rounds_with_kill"`
	RoundsWithMultiKill int `json:"rounds_with_multi_kill"`
	KillsInWonRounds    int `json:"kills_in_won_rounds"`
	DamageInWonRounds   int `json:"damage_in_won_rounds"`
	AttackRounds        int `json:"attack_rounds"`
	SupportRounds       int `json:"support_rounds"`
	AssistedKills       int `json:"assisted_kills"`
	KnifeKills          int `json:"knife_kills"`
	PistolVsRifleKills  int `json:"pistol_vs_rifle_kills"`
	ExitFrags           int `json:"exit_frags"`
	EarlyDeaths         int `json:"early_deaths"`
}

// SideStats tracks statistics for a specific side (T or CT).
type SideStats struct {
	RoundsPlayed        int     `json:"rounds_played"`
	Kills               int     `json:"kills"`
	Deaths              int     `json:"deaths"`
	Damage              int     `json:"damage"`
	Survivals           int     `json:"survivals"`
	RoundsWithMultiKill int     `json:"rounds_with_multi_kill"`
	EcoKillValue        float64 `json:"eco_kill_value"`
	RoundSwing          float64 `json:"round_swing"`
	KAST                float64 `json:"kast"`
	MultiKills          [6]int  `json:"-"`
	ClutchRounds        int     `json:"clutch_rounds"`
	ClutchWins          int     `json:"clutch_wins"`
	Rating              float64 `json:"rating"`
	EcoRating           float64 `json:"eco_rating"`
}

// PistolStats tracks pistol round performance.
type PistolStats struct {
	RoundsPlayed int     `json:"rounds_played"`
	Kills        int     `json:"kills"`
	Deaths       int     `json:"deaths"`
	Damage       int     `json:"damage"`
	RoundsWon    int     `json:"rounds_won"`
	Survivals    int     `json:"survivals"`
	MultiKills   int     `json:"multi_kills"`
	Rating       float64 `json:"rating"`
}

// DerivedMetrics contains calculated per-round and percentage metrics.
// These are computed after parsing from raw statistics.
type DerivedMetrics struct {
	ADR                        float64 `json:"adr"`
	KPR                        float64 `json:"kpr"`
	DPR                        float64 `json:"dpr"`
	AWPKillsPerRound           float64 `json:"awp_kills_per_round"`
	TimeAlivePerRound          float64 `json:"time_alive_per_round"`
	EnemyFlashDurationPerRound float64 `json:"enemy_flash_duration_per_round"`
	TeamFlashDurationPerRound  float64 `json:"team_flash_duration_per_round"`
	RoundsWithKillPct          float64 `json:"rounds_with_kill_pct"`
	RoundsWithMultiKillPct     float64 `json:"rounds_with_multi_kill_pct"`
	KillsPerRoundWin           float64 `json:"kills_per_round_win"`
	DamagePerRoundWin          float64 `json:"damage_per_round_win"`
	SavedByTeammatePerRound    float64 `json:"saved_by_teammate_per_round"`
	TradedDeathsPerRound       float64 `json:"traded_deaths_per_round"`
	TradedDeathsPct            float64 `json:"traded_deaths_pct"`
	OpeningDeathsTradedPct     float64 `json:"opening_deaths_traded_pct"`
	AssistsPerRound            float64 `json:"assists_per_round"`
	SupportRoundsPct           float64 `json:"support_rounds_pct"`
	SavedTeammatePerRound      float64 `json:"saved_teammate_per_round"`
	TradeKillsPerRound         float64 `json:"trade_kills_per_round"`
	TradeKillsPct              float64 `json:"trade_kills_pct"`
	AssistedKillsPct           float64 `json:"assisted_kills_pct"`
	DamagePerKill              float64 `json:"damage_per_kill"`
	OpeningKillsPerRound       float64 `json:"opening_kills_per_round"`
	OpeningDeathsPerRound      float64 `json:"opening_deaths_per_round"`
	OpeningAttemptsPct         float64 `json:"opening_attempts_pct"`
	OpeningSuccessPct          float64 `json:"opening_success_pct"`
	WinPctAfterOpeningKill     float64 `json:"win_pct_after_opening_kill"`
	AttacksPerRound            float64 `json:"attacks_per_round"`
	ClutchPointsPerRound       float64 `json:"clutch_points_per_round"`
	LastAlivePct               float64 `json:"last_alive_pct"`
	Clutch1v1WinPct            float64 `json:"clutch_1v1_win_pct"`
	SavesPerRoundLoss          float64 `json:"saves_per_round_loss"`
	AWPKillsPct                float64 `json:"awp_kills_pct"`
	RoundsWithAWPKillPct       float64 `json:"rounds_with_awp_kill_pct"`
	AWPMultiKillRoundsPerRound float64 `json:"awp_multi_kill_rounds_per_round"`
	AWPOpeningKillsPerRound    float64 `json:"awp_opening_kills_per_round"`
	UtilityDamagePerRound      float64 `json:"utility_damage_per_round"`
	UtilityKillsPer100Rounds   float64 `json:"utility_kills_per_100_rounds"`
	FlashesThrownPerRound      float64 `json:"flashes_thrown_per_round"`
	FlashAssistsPerRound       float64 `json:"flash_assists_per_round"`
	LowBuyKillsPct             float64 `json:"low_buy_kills_pct"`
	DisadvantagedBuyKillsPct   float64 `json:"disadvantaged_buy_kills_pct"`
}

// RatingMetrics contains all rating-related values.
type RatingMetrics struct {
	RoundImpact float64 `json:"round_impact"`
	Survival    float64 `json:"survival"`
	KAST        float64 `json:"kast"`
	RoundSwing  float64 `json:"round_swing"`
	HLTVRating  float64 `json:"hltv_rating"`
	FinalRating float64 `json:"final_rating"`
}
