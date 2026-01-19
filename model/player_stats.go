package model

// MultiKillStats holds multi-kill counts with explicit labels
type MultiKillStats struct {
	OneK   int `json:"1k"`
	TwoK   int `json:"2k"`
	ThreeK int `json:"3k"`
	FourK  int `json:"4k"`
	FiveK  int `json:"5k"`
}

type PlayerStats struct {
	SteamID string `json:"steam_id"`
	Name    string `json:"name"`

	RoundsPlayed int `json:"rounds_played"`
	RoundsWon    int `json:"rounds_won"`  // Rounds where player's team won
	RoundsLost   int `json:"rounds_lost"` // Rounds where player's team lost

	// Core stats
	Kills        int `json:"kills"`
	Assists      int `json:"assists"`
	Deaths       int `json:"deaths"`
	Damage       int `json:"damage"`
	OpeningKills int `json:"opening_kills"`

	// Per-round stats (calculated at end)
	ADR          float64 `json:"adr"` // Average Damage per Round
	KPR          float64 `json:"kpr"` // Kills per Round
	DPR          float64 `json:"dpr"` // Deaths per Round
	PerfectKills int     `json:"perfect_kills"`
	TradeDenials int     `json:"trade_denials"`
	TradedDeaths int     `json:"traded_deaths"`

	// Kill tracking
	RoundsWithKill      int `json:"rounds_with_kill"`       // Rounds where player got at least one kill
	RoundsWithMultiKill int `json:"rounds_with_multi_kill"` // Rounds where player got 2+ kills
	KillsInWonRounds    int `json:"kills_in_won_rounds"`    // Kills in rounds that were won
	DamageInWonRounds   int `json:"damage_in_won_rounds"`   // Damage in rounds that were won

	// AWP/Sniper specific stats
	AWPKills           int     `json:"awp_kills"`
	AWPKillsPerRound   float64 `json:"awp_kills_per_round"`
	RoundsWithAWPKill  int     `json:"rounds_with_awp_kill"`  // Rounds where player got at least one AWP kill
	AWPMultiKillRounds int     `json:"awp_multi_kill_rounds"` // Rounds with 2+ AWP kills
	AWPOpeningKills    int     `json:"awp_opening_kills"`     // Opening kills with AWP

	MultiKillsRaw [6]int         `json:"-"`           // index = kills in round (internal use)
	MultiKills    MultiKillStats `json:"multi_kills"` // Formatted for JSON export

	RoundImpact float64 `json:"round_impact"`
	Survival    float64 `json:"survival"`
	KAST        float64 `json:"kast"`
	EconImpact  float64 `json:"econ_impact"`

	// Eco-adjusted values
	EcoKillValue  float64 `json:"eco_kill_value"`  // Sum of eco-adjusted kill values
	EcoDeathValue float64 `json:"eco_death_value"` // Sum of eco-adjusted death penalties

	// Round Swing - measures contribution to round wins/losses
	RoundSwing   float64 `json:"round_swing"`   // Cumulative round swing score
	ClutchRounds int     `json:"clutch_rounds"` // Rounds where player was last alive
	ClutchWins   int     `json:"clutch_wins"`   // Clutch rounds won

	// Support stats
	SavedByTeammate     int `json:"saved_by_teammate"`     // Times saved by teammate (teammate killed attacker)
	SavedTeammate       int `json:"saved_teammate"`        // Times saved a teammate
	OpeningDeaths       int `json:"opening_deaths"`        // Times died first in round
	OpeningDeathsTraded int `json:"opening_deaths_traded"` // Opening deaths that were traded
	SupportRounds       int `json:"support_rounds"`        // Rounds with assist or flash assist
	AssistedKills       int `json:"assisted_kills"`        // Kills where player assisted (for assisted kills %)

	// Trade stats
	TradeKills int `json:"trade_kills"` // Total trade kills
	FastTrades int `json:"fast_trades"` // Trade kills within 2 seconds

	// Entry/Opening stats
	OpeningAttempts       int `json:"opening_attempts"`         // Rounds where player was involved in first duel
	OpeningSuccesses      int `json:"opening_successes"`        // Opening duels won (got opening kill)
	RoundsWonAfterOpening int `json:"rounds_won_after_opening"` // Rounds won where player got opening kill
	AttackRounds          int `json:"attack_rounds"`            // Rounds where player got a kill (attacks)

	// Clutch stats
	Clutch1v1Attempts int     `json:"clutch_1v1_attempts"`  // 1v1 clutch attempts
	Clutch1v1Wins     int     `json:"clutch_1v1_wins"`      // 1v1 clutch wins
	TotalTimeAlive    float64 `json:"-"`                    // Total time alive across all rounds (seconds) - internal
	TimeAlivePerRound float64 `json:"time_alive_per_round"` // Average time alive per round
	LastAliveRounds   int     `json:"last_alive_rounds"`    // Rounds where player was last alive on team
	SavesOnLoss       int     `json:"saves_on_loss"`        // Rounds where player survived a lost round

	// Utility stats
	UtilityDamage              int     `json:"utility_damage"`                 // Total utility damage (HE, molotov, incendiary)
	UtilityKills               int     `json:"utility_kills"`                  // Kills with utility (HE, molotov, incendiary)
	FlashesThrown              int     `json:"flashes_thrown"`                 // Total flashes thrown
	FlashAssists               int     `json:"flash_assists"`                  // Total flash assists
	EnemyFlashDuration         float64 `json:"-"`                              // Total time enemies were flashed - internal
	EnemyFlashDurationPerRound float64 `json:"enemy_flash_duration_per_round"` // Per-round average
	TeamFlashCount             int     `json:"team_flash_count"`               // Total times flashed teammates
	TeamFlashDuration          float64 `json:"-"`                              // Total duration of team flashes - internal
	TeamFlashDurationPerRound  float64 `json:"team_flash_duration_per_round"`  // Per-round average

	// Misc stats
	ExitFrags                int     `json:"exit_frags"`                  // Total exit frags
	AWPDeaths                int     `json:"awp_deaths"`                  // Times died with AWP
	AWPDeathsNoKill          int     `json:"awp_deaths_no_kill"`          // Times died with AWP without getting AWP kill
	KnifeKills               int     `json:"knife_kills"`                 // Total knife kills
	PistolVsRifleKills       int     `json:"pistol_vs_rifle_kills"`       // Total pistol kills vs rifle players
	EarlyDeaths              int     `json:"early_deaths"`                // Deaths within first 30 seconds
	LowBuyKills              int     `json:"low_buy_kills"`               // Kills on lower-equipped opponents (EcoKillValue < 1.0)
	LowBuyKillsPct           float64 `json:"low_buy_kills_pct"`           // Percentage of kills that were low buy
	DisadvantagedBuyKills    int     `json:"disadvantaged_buy_kills"`     // Kills on significantly lower-equipped opponents (EcoKillValue <= 0.85)
	DisadvantagedBuyKillsPct float64 `json:"disadvantaged_buy_kills_pct"` // Percentage of kills that were disadvantaged

	// Pistol round stats
	PistolRoundsPlayed    int     `json:"pistol_rounds_played"`
	PistolRoundKills      int     `json:"pistol_round_kills"`
	PistolRoundDeaths     int     `json:"pistol_round_deaths"`
	PistolRoundDamage     int     `json:"pistol_round_damage"`
	PistolRoundsWon       int     `json:"pistol_rounds_won"`
	PistolRoundSurvivals  int     `json:"pistol_round_survivals"`   // Times survived pistol rounds
	PistolRoundMultiKills int     `json:"pistol_round_multi_kills"` // Pistol rounds with 2+ kills
	PistolRoundRating     float64 `json:"pistol_round_rating"`      // HLTV Rating 1.0 for pistol rounds only

	// HLTV Rating 1.0 components
	HLTVRating float64 `json:"hltv_rating"`

	// Per-side stats (T = Terrorist, CT = Counter-Terrorist)
	// T-side raw stats
	TRoundsPlayed        int     `json:"t_rounds_played"`
	TKills               int     `json:"t_kills"`
	TDeaths              int     `json:"t_deaths"`
	TDamage              int     `json:"t_damage"`
	TSurvivals           int     `json:"t_survivals"`
	TRoundsWithMultiKill int     `json:"t_rounds_with_multi_kill"`
	TEcoKillValue        float64 `json:"t_eco_kill_value"`
	TRoundSwing          float64 `json:"t_round_swing"`
	TKAST                float64 `json:"t_kast"` // Count of KAST rounds on T-side
	TMultiKills          [6]int  `json:"-"`      // Internal use
	TClutchRounds        int     `json:"t_clutch_rounds"`
	TClutchWins          int     `json:"t_clutch_wins"`

	// T-side calculated ratings
	TRating    float64 `json:"t_rating"`     // HLTV Rating 1.0 for T-side
	TEcoRating float64 `json:"t_eco_rating"` // Eco Rating (FinalRating) for T-side

	// CT-side raw stats
	CTRoundsPlayed        int     `json:"ct_rounds_played"`
	CTKills               int     `json:"ct_kills"`
	CTDeaths              int     `json:"ct_deaths"`
	CTDamage              int     `json:"ct_damage"`
	CTSurvivals           int     `json:"ct_survivals"`
	CTRoundsWithMultiKill int     `json:"ct_rounds_with_multi_kill"`
	CTEcoKillValue        float64 `json:"ct_eco_kill_value"`
	CTRoundSwing          float64 `json:"ct_round_swing"`
	CTKAST                float64 `json:"ct_kast"` // Count of KAST rounds on CT-side
	CTMultiKills          [6]int  `json:"-"`       // Internal use
	CTClutchRounds        int     `json:"ct_clutch_rounds"`
	CTClutchWins          int     `json:"ct_clutch_wins"`

	// CT-side calculated ratings
	CTRating    float64 `json:"ct_rating"`     // HLTV Rating 1.0 for CT-side
	CTEcoRating float64 `json:"ct_eco_rating"` // Eco Rating (FinalRating) for CT-side

	FinalRating float64 `json:"final_rating"`

	// Calculated per-round stats (matching AggregatedStats)
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
}
