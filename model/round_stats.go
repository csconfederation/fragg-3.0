// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

package model

// RoundStats tracks a player's performance within a single round.
// This struct is populated during demo parsing and used to calculate
// per-round metrics like round swing, KAST, and clutch statistics.
type RoundStats struct {
	Kills              int
	Assists            int
	Deaths             int // death presence is Deaths > 0; DeathTime can be 0 or less (#591)
	Damage             int
	Survived           bool
	Traded             bool
	GotKill            bool
	GotAssist          bool
	EconImpact         float64
	AWPKills           int
	AWPOpeningKill     bool
	TeamWon            bool
	PlantedBomb        bool
	DefusedBomb        bool
	OpeningDeath       bool
	ClutchAttempt      bool
	ClutchWon          bool
	ClutchSize         int
	SavedWeapons       bool
	FlashAssists       int
	TradeKills         int // trade kills, each kill once (#591)
	FastTrades         int // trade kills within 2 s of the avenged death
	TradeDenials       int
	SavedByTeammate    bool
	SavedTeammate      bool
	UtilityKills       int
	DeathTime          float64
	TimeAlive          float64
	KillTimes          []float64
	ExitFrags          int
	EnemyFlashDuration float64
	AWPKill            bool
	KnifeKill          bool
	PistolVsRifleKill  bool
	LostAWP            bool
	PlayerSide         string

	// Damage taken this round
	DamageTaken int

	// Probability-based swing tracking (new for v3.0)
	ProbabilitySwing float64 // Win probability delta contribution
	LastDeathSwing   float64 // Most recent death swing (for trade refund calculation)

	// Event-time counters stored on the round so match totals can be folded
	// after CSC drops knife/incomplete/redo rounds.
	EcoDeathValue         float64
	TimeToKillSum         float64
	KillsWithTTK          int
	LowBuyKills           int
	DisadvantagedBuyKills int
	ManAdvantageKills     int
	ManDisadvantageDeaths int
	EcoAdjustedKills      float64
	PerfectKills          int
	EnemiesFlashed        int
}
