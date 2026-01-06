package model

type RoundStats struct {
	Kills      int
	Assists    int
	Damage     int
	Survived   bool
	Traded     bool
	GotKill    bool
	GotAssist  bool
	EconImpact float64

	// AWP specific stats
	AWPKills int

	// Round Swing tracking
	TeamWon      bool // Did this player's team win the round?
	PlayersAlive int  // How many teammates alive at round end (including self if survived)
	EnemiesAlive int  // How many enemies alive at round end
	WasLastAlive bool // Was this player the last alive (clutch situation)?
	ClutchKills  int  // Kills while last alive

	// Advanced Round Swing factors
	PlantedBomb    bool // Did this player plant the bomb?
	DefusedBomb    bool // Did this player defuse the bomb?
	OpeningKill    bool // Did this player get the opening kill?
	OpeningDeath   bool // Did this player die first?
	MultiKillRound int  // Number of kills in this round (for multi-kill detection)
	EntryFragger   bool // Got first kill of the round for their team
	ClutchAttempt  bool // Was in a clutch situation (1vX)
	ClutchWon      bool // Won a clutch situation
	SavedWeapons   bool // Survived a lost round (weapon save)
	EcoKill        bool // Got a kill while on eco/force buy
	AntiEcoKill    bool // Got killed by eco while on full buy
	FlashAssists   int  // Number of flash assists this round
	TradeKill      bool // Got a trade kill
	TradeDeath     bool // Death was traded by teammate

	// New Round Swing factors
	UtilityDamage     int       // Damage dealt via grenades (HE, molotov, incendiary)
	SmokeDamage       int       // Damage dealt through smoke
	DeathTime         float64   // Time of death relative to round start (seconds)
	KillTimes         []float64 // Times of each kill relative to round start
	TradeSpeed        float64   // Time between teammate death and trade kill (seconds)
	IsExitFrag        bool      // Kill happened after round was decided
	ExitFrags         int       // Number of exit frags this round
	TeamFlashCount    int       // Number of times flashed teammates
	TeamFlashDuration float64   // Total duration of team flashes
	AWPKill           bool      // Got a kill with AWP this round
	KnifeKill         bool      // Got a knife kill this round
	PistolVsRifleKill bool      // Got a pistol kill vs rifle
	HadAWP            bool      // Player had AWP this round
	LostAWP           bool      // Died with AWP (lost the weapon)
}

// RoundContext provides situational information for round swing calculation
type RoundContext struct {
	RoundNumber     int
	TotalPlayers    int     // Total players alive at round start
	BombPlanted     bool    // Was bomb planted this round
	BombDefused     bool    // Was bomb defused this round
	RoundType       string  // "pistol", "eco", "force", "full"
	TimeRemaining   float64 // Time left when round ended
	IsOvertimeRound bool    // Is this an overtime round
	MapSide         string  // "T" or "CT"

	// Score context for round importance
	TeamScore       int     // Current team score before this round
	EnemyScore      int     // Current enemy score before this round
	ScoreDiff       int     // Team score - Enemy score (positive = winning)
	IsMatchPoint    bool    // Either team is at match point
	IsCloseGame     bool    // Score difference <= 3
	RoundImportance float64 // Multiplier based on score context (1.0 = normal)

	// Round outcome context
	RoundDecided   bool    // Round outcome is already determined
	RoundDecidedAt float64 // Time when round was decided
}
