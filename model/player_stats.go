// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package model holds engine/eco's per-player and per-round working stats.
package model

// PlayerStats is a player's eco stats for the match, folded from their RoundStats; engine/fold
// merges them into the payload's rows.
type PlayerStats struct {
	TeamName string

	RoundsPlayed       int
	Kills              int
	Deaths             int
	Damage             int
	KPR                float64
	DPR                float64
	TotalTimeToKill    float64
	KillsWithTTK       int
	AvgTimeToKill      float64
	PerfectKills       int
	TradeDenials       int
	AWPKills           int
	AWPKillsPerRound   float64
	RoundsWithAWPKill  int
	AWPMultiKillRounds int
	AWPOpeningKills    int

	MultiKillsRaw [6]int

	Survival               float64
	KAST                   float64
	EcoKillValue           float64
	EcoDeathValue          float64
	DuelSwing              float64
	DuelSwingPerRound      float64
	SavedByTeammate        int
	SavedTeammate          int
	OpeningDeathsTraded    int
	AssistedKills          int
	TradeKills             int
	FastTrades             int
	ManAdvantageKills      int
	ManDisadvantageDeaths  int
	Clutch1v1Attempts      int
	TotalTimeAlive         float64
	TimeAlivePerRound      float64
	UtilityKills           int
	ExitFrags              int
	AWPDeaths              int
	KnifeKills             int
	PistolVsRifleKills     int
	EarlyDeaths            int
	LowBuyKills            int
	DisadvantagedBuyKills  int
	PistolRoundsPlayed     int
	PistolRoundKills       int
	PistolRoundSurvivals   int
	PistolRoundMultiKills  int
	PistolRoundRating      float64
	HLTVRating             float64
	TRoundsPlayed          int
	TKills                 int
	TDeaths                int
	TDamage                int
	TSurvivals             int
	TEcoKillValue          float64
	TEcoDeathValue         float64
	TProbabilitySwing      float64
	TKAST                  float64
	TMultiKills            [6]int
	TManAdvantageKills     int
	TManDisadvantageDeaths int
	TRating                float64
	TEcoRating             float64
	// T-side counterparts of whole-match eco counters.
	TAssistedKills          int
	TClutch1v1Attempts      int
	TClutch1v2Attempts      int
	TClutch1v3Attempts      int
	TClutch1v4Attempts      int
	TClutch1v5Attempts      int
	TTradeKills             int
	TTradeDenials           int
	TFastTrades             int
	TOpeningDeathsTraded    int
	TAWPKills               int
	TAWPOpeningKills        int
	TAWPMultiKillRounds     int
	TAWPDeaths              int
	TRoundsWithAWPKill      int
	TExitFrags              int
	TKnifeKills             int
	TPistolVsRifleKills     int
	TUtilityKills           int
	TEarlyDeaths            int
	TSavedByTeammate        int
	TSavedTeammate          int
	CTRoundsPlayed          int
	CTKills                 int
	CTDeaths                int
	CTDamage                int
	CTSurvivals             int
	CTEcoKillValue          float64
	CTEcoDeathValue         float64
	CTProbabilitySwing      float64
	CTKAST                  float64
	CTMultiKills            [6]int
	CTManAdvantageKills     int
	CTManDisadvantageDeaths int
	CTRating                float64
	CTEcoRating             float64
	// CT-side counterparts of whole-match eco counters.
	CTAssistedKills       int
	CTClutch1v1Attempts   int
	CTClutch1v2Attempts   int
	CTClutch1v3Attempts   int
	CTClutch1v4Attempts   int
	CTClutch1v5Attempts   int
	CTTradeKills          int
	CTTradeDenials        int
	CTFastTrades          int
	CTOpeningDeathsTraded int
	CTAWPKills            int
	CTAWPOpeningKills     int
	CTAWPMultiKillRounds  int
	CTAWPDeaths           int
	CTRoundsWithAWPKill   int
	CTExitFrags           int
	CTKnifeKills          int
	CTPistolVsRifleKills  int
	CTUtilityKills        int
	CTEarlyDeaths         int
	CTSavedByTeammate     int
	CTSavedTeammate       int

	FinalRating float64

	// Clutch attempts by opponent count.
	Clutch1v2Attempts int
	Clutch1v3Attempts int
	Clutch1v4Attempts int
	Clutch1v5Attempts int

	// Average Time to Death - derived from TimeAlivePerRound
	// ATD = average time survived in rounds where player died
	TotalDeathTime  float64
	DeathTimeRounds int
	AvgTimeToDeath  float64

	// Probability-based swing metrics (new for v3.0)
	ProbabilitySwing         float64 // Cumulative win probability contribution
	ProbabilitySwingPerRound float64 // Average swing per round
	EcoAdjustedKills         float64 // Kills weighted by duel difficulty
}
