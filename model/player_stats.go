// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model defines the core data structures for player and round statistics.
// These structs are used throughout the application to track, aggregate, and export
// player performance metrics from CS2 demo files.
package model

// MultiKillStats tracks the count of multi-kill rounds by kill count.
// These are used in HLTV rating calculations with weighted scoring.
type MultiKillStats struct {
	OneK   int `json:"1k"` // Rounds with exactly 1 kill
	TwoK   int `json:"2k"` // Rounds with exactly 2 kills (Double Kill)
	ThreeK int `json:"3k"` // Rounds with exactly 3 kills (Triple Kill)
	FourK  int `json:"4k"` // Rounds with exactly 4 kills (Quad Kill)
	FiveK  int `json:"5k"` // Rounds with 5 kills (Ace)
}

// PlayerStats contains all tracked statistics for a single player in a game.
// This is the primary data structure populated by the demo parser and used
// for rating calculations and exports. Fields are organized into categories:
// - Basic stats (kills, deaths, damage)
// - Economy metrics (eco kill value, equipment ratios)
// - Opening/entry statistics
// - Clutch and trade statistics
// - AWP-specific metrics
// - Utility usage (flashes, grenades)
// - Side-specific stats (T/CT)
// - Calculated ratings and percentages
type PlayerStats struct {
	SteamID string `json:"steam_id"`
	Name    string `json:"name"`

	RoundsPlayed        int     `json:"rounds_played"`
	RoundsWon           int     `json:"rounds_won"`
	RoundsLost          int     `json:"rounds_lost"`
	Kills               int     `json:"kills"`
	Assists             int     `json:"assists"`
	Deaths              int     `json:"deaths"`
	Damage              int     `json:"damage"`
	OpeningKills        int     `json:"opening_kills"`
	ADR                 float64 `json:"adr"`
	KPR                 float64 `json:"kpr"`
	DPR                 float64 `json:"dpr"`
	Headshots           int     `json:"headshots"`
	HeadshotPct         float64 `json:"headshot_pct"`
	TotalTimeToKill     float64 `json:"-"`
	KillsWithTTK        int     `json:"-"`
	AvgTimeToKill       float64 `json:"avg_time_to_kill"`
	PerfectKills        int     `json:"perfect_kills"`
	TradeDenials        int     `json:"trade_denials"`
	TradedDeaths        int     `json:"traded_deaths"`
	RoundsWithKill      int     `json:"rounds_with_kill"`
	RoundsWithMultiKill int     `json:"rounds_with_multi_kill"`
	KillsInWonRounds    int     `json:"kills_in_won_rounds"`
	DamageInWonRounds   int     `json:"damage_in_won_rounds"`
	AWPKills            int     `json:"awp_kills"`
	AWPKillsPerRound    float64 `json:"awp_kills_per_round"`
	RoundsWithAWPKill   int     `json:"rounds_with_awp_kill"`
	AWPMultiKillRounds  int     `json:"awp_multi_kill_rounds"`
	AWPOpeningKills     int     `json:"awp_opening_kills"`

	MultiKillsRaw [6]int         `json:"-"`
	MultiKills    MultiKillStats `json:"multi_kills"`

	RoundImpact                float64 `json:"round_impact"`
	Survival                   float64 `json:"survival"`
	KAST                       float64 `json:"kast"`
	EconImpact                 float64 `json:"econ_impact"`
	EcoKillValue               float64 `json:"eco_kill_value"`
	EcoDeathValue              float64 `json:"eco_death_value"`
	RoundSwing                 float64 `json:"round_swing"`
	ClutchRounds               int     `json:"clutch_rounds"`
	ClutchWins                 int     `json:"clutch_wins"`
	SavedByTeammate            int     `json:"saved_by_teammate"`
	SavedTeammate              int     `json:"saved_teammate"`
	OpeningDeaths              int     `json:"opening_deaths"`
	OpeningDeathsTraded        int     `json:"opening_deaths_traded"`
	SupportRounds              int     `json:"support_rounds"`
	AssistedKills              int     `json:"assisted_kills"`
	TradeKills                 int     `json:"trade_kills"`
	FastTrades                 int     `json:"fast_trades"`
	OpeningAttempts            int     `json:"opening_attempts"`
	OpeningSuccesses           int     `json:"opening_successes"`
	RoundsWonAfterOpening      int     `json:"rounds_won_after_opening"`
	AttackRounds               int     `json:"attack_rounds"`
	Clutch1v1Attempts          int     `json:"clutch_1v1_attempts"`
	Clutch1v1Wins              int     `json:"clutch_1v1_wins"`
	TotalTimeAlive             float64 `json:"-"`
	TimeAlivePerRound          float64 `json:"time_alive_per_round"`
	LastAliveRounds            int     `json:"last_alive_rounds"`
	SavesOnLoss                int     `json:"saves_on_loss"`
	UtilityDamage              int     `json:"utility_damage"`
	UtilityKills               int     `json:"utility_kills"`
	FlashesThrown              int     `json:"flashes_thrown"`
	FlashAssists               int     `json:"flash_assists"`
	EnemyFlashDuration         float64 `json:"-"`
	EnemyFlashDurationPerRound float64 `json:"enemy_flash_duration_per_round"`
	TeamFlashCount             int     `json:"team_flash_count"`
	TeamFlashDuration          float64 `json:"-"`
	TeamFlashDurationPerRound  float64 `json:"team_flash_duration_per_round"`
	ExitFrags                  int     `json:"exit_frags"`
	AWPDeaths                  int     `json:"awp_deaths"`
	AWPDeathsNoKill            int     `json:"awp_deaths_no_kill"`
	KnifeKills                 int     `json:"knife_kills"`
	PistolVsRifleKills         int     `json:"pistol_vs_rifle_kills"`
	EarlyDeaths                int     `json:"early_deaths"`
	LowBuyKills                int     `json:"low_buy_kills"`
	LowBuyKillsPct             float64 `json:"low_buy_kills_pct"`
	DisadvantagedBuyKills      int     `json:"disadvantaged_buy_kills"`
	DisadvantagedBuyKillsPct   float64 `json:"disadvantaged_buy_kills_pct"`
	PistolRoundsPlayed         int     `json:"pistol_rounds_played"`
	PistolRoundKills           int     `json:"pistol_round_kills"`
	PistolRoundDeaths          int     `json:"pistol_round_deaths"`
	PistolRoundDamage          int     `json:"pistol_round_damage"`
	PistolRoundsWon            int     `json:"pistol_rounds_won"`
	PistolRoundSurvivals       int     `json:"pistol_round_survivals"`
	PistolRoundMultiKills      int     `json:"pistol_round_multi_kills"`
	PistolRoundRating          float64 `json:"pistol_round_rating"`
	HLTVRating                 float64 `json:"hltv_rating"`
	TRoundsPlayed              int     `json:"t_rounds_played"`
	TKills                     int     `json:"t_kills"`
	TDeaths                    int     `json:"t_deaths"`
	TDamage                    int     `json:"t_damage"`
	TSurvivals                 int     `json:"t_survivals"`
	TRoundsWithMultiKill       int     `json:"t_rounds_with_multi_kill"`
	TEcoKillValue              float64 `json:"t_eco_kill_value"`
	TRoundSwing                float64 `json:"t_round_swing"`
	TProbabilitySwing          float64 `json:"t_probability_swing"`
	TKAST                      float64 `json:"t_kast"`
	TMultiKills                [6]int  `json:"-"`
	TClutchRounds              int     `json:"t_clutch_rounds"`
	TClutchWins                int     `json:"t_clutch_wins"`
	TRating                    float64 `json:"t_rating"`
	TEcoRating                 float64 `json:"t_eco_rating"`
	CTRoundsPlayed             int     `json:"ct_rounds_played"`
	CTKills                    int     `json:"ct_kills"`
	CTDeaths                   int     `json:"ct_deaths"`
	CTDamage                   int     `json:"ct_damage"`
	CTSurvivals                int     `json:"ct_survivals"`
	CTRoundsWithMultiKill      int     `json:"ct_rounds_with_multi_kill"`
	CTEcoKillValue             float64 `json:"ct_eco_kill_value"`
	CTRoundSwing               float64 `json:"ct_round_swing"`
	CTProbabilitySwing         float64 `json:"ct_probability_swing"`
	CTKAST                     float64 `json:"ct_kast"`
	CTMultiKills               [6]int  `json:"-"`
	CTClutchRounds             int     `json:"ct_clutch_rounds"`
	CTClutchWins               int     `json:"ct_clutch_wins"`
	CTRating                   float64 `json:"ct_rating"`
	CTEcoRating                float64 `json:"ct_eco_rating"`

	FinalRating                float64 `json:"final_rating"`
	RoundsWithKillPct          float64 `json:"rounds_with_kill_pct"`
	KillsPerRoundWin           float64 `json:"kills_per_round_win"`
	RoundsWithMultiKillPct     float64 `json:"rounds_with_multi_kill_pct"`
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

	// Probability-based swing metrics (new for v3.0)
	ProbabilitySwing         float64 `json:"probability_swing"`           // Cumulative win probability contribution
	ProbabilitySwingPerRound float64 `json:"probability_swing_per_round"` // Average swing per round
	EcoAdjustedKills         float64 `json:"eco_adjusted_kills"`          // Kills weighted by duel difficulty
	SwingRating              float64 `json:"swing_rating"`                // Swing contribution to final rating
}
