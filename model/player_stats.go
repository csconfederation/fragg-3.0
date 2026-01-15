package model

type PlayerStats struct {
	SteamID string
	Name    string

	RoundsPlayed int
	RoundsWon    int // Rounds where player's team won
	RoundsLost   int // Rounds where player's team lost

	// Core stats
	Kills        int
	Assists      int
	Deaths       int
	Damage       int
	OpeningKills int

	// Per-round stats (calculated at end)
	ADR          float64 // Average Damage per Round
	KPR          float64 // Kills per Round
	DPR          float64 // Deaths per Round
	PerfectKills int
	TradeDenials int
	TradedDeaths int

	// Kill tracking
	RoundsWithKill      int // Rounds where player got at least one kill
	RoundsWithMultiKill int // Rounds where player got 2+ kills
	KillsInWonRounds    int // Kills in rounds that were won
	DamageInWonRounds   int // Damage in rounds that were won

	// AWP/Sniper specific stats
	AWPKills           int
	AWPKillsPerRound   float64
	RoundsWithAWPKill  int // Rounds where player got at least one AWP kill
	AWPMultiKillRounds int // Rounds with 2+ AWP kills
	AWPOpeningKills    int // Opening kills with AWP

	MultiKills [6]int // index = kills in round

	RoundImpact float64
	Survival    float64
	KAST        float64
	EconImpact  float64

	// Eco-adjusted values
	EcoKillValue  float64 // Sum of eco-adjusted kill values
	EcoDeathValue float64 // Sum of eco-adjusted death penalties

	// Round Swing - measures contribution to round wins/losses
	RoundSwing   float64 // Cumulative round swing score
	ClutchRounds int     // Rounds where player was last alive
	ClutchWins   int     // Clutch rounds won

	// Support stats
	SavedByTeammate     int // Times saved by teammate (teammate killed attacker)
	SavedTeammate       int // Times saved a teammate
	OpeningDeaths       int // Times died first in round
	OpeningDeathsTraded int // Opening deaths that were traded
	SupportRounds       int // Rounds with assist or flash assist
	AssistedKills       int // Kills where player assisted (for assisted kills %)

	// Trade stats
	TradeKills int // Total trade kills
	FastTrades int // Trade kills within 2 seconds

	// Entry/Opening stats
	OpeningAttempts       int // Rounds where player was involved in first duel
	OpeningSuccesses      int // Opening duels won (got opening kill)
	RoundsWonAfterOpening int // Rounds won where player got opening kill
	AttackRounds          int // Rounds where player got a kill (attacks)

	// Clutch stats
	Clutch1v1Attempts int     // 1v1 clutch attempts
	Clutch1v1Wins     int     // 1v1 clutch wins
	TotalTimeAlive    float64 // Total time alive across all rounds (seconds)
	LastAliveRounds   int     // Rounds where player was last alive on team
	SavesOnLoss       int     // Rounds where player survived a lost round

	// Utility stats
	UtilityDamage      int     // Total utility damage (HE, molotov, incendiary)
	UtilityKills       int     // Kills with utility (HE, molotov, incendiary)
	FlashesThrown      int     // Total flashes thrown
	FlashAssists       int     // Total flash assists
	EnemyFlashDuration float64 // Total time enemies were flashed by this player
	TeamFlashCount     int     // Total times flashed teammates
	TeamFlashDuration  float64 // Total duration of team flashes

	// Misc stats
	ExitFrags             int // Total exit frags
	AWPDeaths             int // Times died with AWP
	AWPDeathsNoKill       int // Times died with AWP without getting AWP kill
	KnifeKills            int // Total knife kills
	PistolVsRifleKills    int // Total pistol kills vs rifle players
	EarlyDeaths           int // Deaths within first 30 seconds
	LowBuyKills           int // Kills on lower-equipped opponents (EcoKillValue < 1.0)
	DisadvantagedBuyKills int // Kills on significantly lower-equipped opponents (EcoKillValue <= 0.85)

	// Pistol round stats
	PistolRoundsPlayed    int
	PistolRoundKills      int
	PistolRoundDeaths     int
	PistolRoundDamage     int
	PistolRoundsWon       int
	PistolRoundSurvivals  int     // Times survived pistol rounds
	PistolRoundMultiKills int     // Pistol rounds with 2+ kills
	PistolRoundRating     float64 // HLTV Rating 1.0 for pistol rounds only

	// HLTV Rating 1.0 components
	HLTVRating float64

	// Per-side stats (T = Terrorist, CT = Counter-Terrorist)
	// T-side raw stats
	TRoundsPlayed        int
	TKills               int
	TDeaths              int
	TDamage              int
	TSurvivals           int
	TRoundsWithMultiKill int
	TEcoKillValue        float64
	TRoundSwing          float64
	TKAST                float64 // Count of KAST rounds on T-side
	TMultiKills          [6]int
	TClutchRounds        int
	TClutchWins          int

	// T-side calculated ratings
	TRating    float64 // HLTV Rating 1.0 for T-side
	TEcoRating float64 // Eco Rating (FinalRating) for T-side

	// CT-side raw stats
	CTRoundsPlayed        int
	CTKills               int
	CTDeaths              int
	CTDamage              int
	CTSurvivals           int
	CTRoundsWithMultiKill int
	CTEcoKillValue        float64
	CTRoundSwing          float64
	CTKAST                float64 // Count of KAST rounds on CT-side
	CTMultiKills          [6]int
	CTClutchRounds        int
	CTClutchWins          int

	// CT-side calculated ratings
	CTRating    float64 // HLTV Rating 1.0 for CT-side
	CTEcoRating float64 // Eco Rating (FinalRating) for CT-side

	FinalRating float64
}
