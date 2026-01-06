package model

type PlayerStats struct {
	SteamID string
	Name    string

	RoundsPlayed int

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

	// AWP specific stats
	AWPKills         int
	AWPKillsPerRound float64

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
	RoundsWon    int     // Rounds where player's team won
	ClutchRounds int     // Rounds where player was last alive
	ClutchWins   int     // Clutch rounds won

	// New aggregated stats for export
	UtilityDamage      int     // Total utility damage (HE, molotov, incendiary)
	TeamFlashCount     int     // Total times flashed teammates
	TeamFlashDuration  float64 // Total duration of team flashes
	ExitFrags          int     // Total exit frags
	AWPDeaths          int     // Times died with AWP
	AWPDeathsNoKill    int     // Times died with AWP without getting AWP kill
	KnifeKills         int     // Total knife kills
	PistolVsRifleKills int     // Total pistol kills vs rifle players
	TradeKills         int     // Total trade kills
	FastTrades         int     // Trade kills within 2 seconds
	EarlyDeaths        int     // Deaths within first 30 seconds

	FinalRating float64
}
