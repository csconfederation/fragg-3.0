// Package export provides functionality for exporting player statistics.
// This file defines demoScrape2-compatible types for CSC integration.
package export

// CSCGame represents the top-level game structure matching demoScrape2's Game struct.
// This allows ecorating to be a drop-in replacement for csgo-demo-worker.
type CSCGame struct {
	CoreID           string                     `json:"coreID"`
	MapNum           int                        `json:"mapNum"`
	WinnerClanName   string                     `json:"winnerClanName"`
	Result           string                     `json:"result"`
	Rounds           []*CSCRound                `json:"rounds"`
	Teams            map[string]*CSCTeam        `json:"teams"`
	MapName          string                     `json:"mapName"`
	TickRate         int                        `json:"tickRate"`
	TotalPlayerStats map[uint64]*CSCPlayerStats `json:"totalPlayerStats"`
	CtPlayerStats    map[uint64]*CSCPlayerStats `json:"ctPlayerStats"`
	TPlayerStats     map[uint64]*CSCPlayerStats `json:"TPlayerStats"`
	TotalTeamStats   map[string]*CSCTeamStats   `json:"totalTeamStats"`
	PlayerOrder      []uint64                   `json:"playerOrder"`
	TeamOrder        []string                   `json:"teamOrder"`
	TotalRounds      int                        `json:"totalRounds"`
}

// CSCTeam represents team information.
type CSCTeam struct {
	Name          string `json:"name"`
	Score         int    `json:"score"`
	ScoreAdjusted int    `json:"scoreAdjusted"`
}

// CSCTeamStats represents aggregated team statistics.
type CSCTeamStats struct {
	WinPoints      float64 `json:"winPoints"`
	ImpactPoints   float64 `json:"impactPoints"`
	TWinPoints     float64 `json:"TWinPoints"`
	CtWinPoints    float64 `json:"ctWinPoints"`
	TImpactPoints  float64 `json:"TImpactPoints"`
	CtImpactPoints float64 `json:"ctImpactPoints"`
	Pistols        int     `json:"pistols"`
	PistolsW       int     `json:"pistolsW"`
	Saves          int     `json:"saves"`
	Clutches       int     `json:"clutches"`
	Traded         int     `json:"traded"`
	Fass           int     `json:"fass"`
	Ef             int     `json:"ef"`
	Ud             int     `json:"ud"`
	Util           int     `json:"util"`
	CtR            int     `json:"ctR"`
	CtRW           int     `json:"ctRW"`
	TR             int     `json:"TR"`
	TRW            int     `json:"TRW"`
	Deaths         int     `json:"deaths"`
	Normalizer     int     `json:"normalizer"`
}

// CSCRound represents per-round data.
type CSCRound struct {
	RoundNum          int8                       `json:"roundNum"`
	StartingTick      int                        `json:"startingTick"`
	EndingTick        int                        `json:"endingTick"`
	PlayerStats       map[uint64]*CSCPlayerStats `json:"playerStats"`
	TeamStats         map[string]*CSCTeamStats   `json:"teamStats"`
	WinnerClanName    string                     `json:"winnerClanName"`
	WinnerENUM        int                        `json:"winnerENUM"`
	IntegrityCheck    bool                       `json:"integrityCheck"`
	Planter           uint64                     `json:"planter"`
	Defuser           uint64                     `json:"defuser"`
	EndDueToBombEvent bool                       `json:"endDueToBombEvent"`
	KnifeRound        bool                       `json:"knifeRound"`
	RoundEndReason    string                     `json:"roundEndReason"`
}

// CSCPlayerStats represents player statistics matching demoScrape2's playerStats struct.
// Fields are ordered to match demoScrape2 exactly, with ecorating-specific fields at the end.
type CSCPlayerStats struct {
	// Core identification
	Name         string `json:"name"`
	SteamID      string `json:"steamID"`
	IsBot        bool   `json:"isBot"`
	TeamENUM     int    `json:"teamENUM"`
	TeamClanName string `json:"teamClanName"`
	Side         int    `json:"side"`
	Rounds       int    `json:"rounds"`

	// Combat stats
	Damage         int     `json:"damage"`
	Kills          uint8   `json:"kills"`
	Assists        uint8   `json:"assists"`
	Deaths         uint8   `json:"deaths"`
	DeathTick      int     `json:"deathTick"`
	DeathPlacement float64 `json:"deathPlacement"`
	TicksAlive     int     `json:"ticksAlive"`

	// Trade stats
	Trades int `json:"trades"`
	Traded int `json:"traded"`

	// Opening duels
	Ok int `json:"ok"` // Opening kills
	Ol int `json:"ol"` // Opening losses/deaths

	// Clutch stats (wins)
	Cl_1 int `json:"cl_1"`
	Cl_2 int `json:"cl_2"`
	Cl_3 int `json:"cl_3"`
	Cl_4 int `json:"cl_4"`
	Cl_5 int `json:"cl_5"`

	// Multi-kills
	TwoK   int `json:"twoK"`
	ThreeK int `json:"threeK"`
	FourK  int `json:"fourK"`
	FiveK  int `json:"fiveK"`

	// Utility stats
	NadeDmg        int     `json:"nadeDmg"`
	InfernoDmg     int     `json:"infernoDmg"`
	UtilDmg        int     `json:"utilDmg"`
	Ef             int     `json:"ef"`   // Enemies flashed
	FAss           int     `json:"FAss"` // Flash assists
	EnemyFlashTime float64 `json:"enemyFlashTime"`

	// Other combat stats
	Hs         int     `json:"hs"` // Headshots
	KastRounds float64 `json:"kastRounds"`
	Saves      int     `json:"saves"`
	Entries    int     `json:"entries"`

	// Rating components
	KillPoints   float64 `json:"killPoints"`
	ImpactPoints float64 `json:"impactPoints"`
	WinPoints    float64 `json:"winPoints"`

	// Weapon stats
	AwpKills int `json:"awpKills"`
	RF       int `json:"RF"` // Rifle frags
	RA       int `json:"RA"` // Rifle assists

	// Utility thrown
	NadesThrown int `json:"nadesThrown"`
	FiresThrown int `json:"firesThrown"`
	FlashThrown int `json:"flashThrown"`
	SmokeThrown int `json:"smokeThrown"`

	// Damage taken
	DamageTaken int `json:"damageTaken"`

	// Support stats
	SuppRounds int `json:"suppRounds"`
	SuppDamage int `json:"suppDamage"`

	// Positioning (not tracked by ecorating, set to 0)
	LurkerBlips         int `json:"lurkerBlips"`
	DistanceToTeammates int `json:"distanceToTeammates"`
	LurkRounds          int `json:"lurkRounds"`

	// Advanced metrics
	Wlp float64 `json:"wlp"` // Win loss points
	Mip float64 `json:"mip"` // Match impact points
	Rws float64 `json:"rws"` // Round win shares
	Eac int     `json:"eac"` // Effective assist contributions
	Rwk int     `json:"rwk"` // Rounds with kills

	// Derived stats
	UtilThrown   int     `json:"utilThrown"`
	Atd          int     `json:"atd"` // Average time to death (in ticks)
	Kast         float64 `json:"kast"`
	KillPointAvg float64 `json:"killPointAvg"`
	Iiwr         float64 `json:"iiwr"`
	Adr          float64 `json:"adr"`
	DrDiff       float64 `json:"drDiff"`
	KR           float64 `json:"KR"` // Kill ratio
	Tr           float64 `json:"tr"` // Trade ratio
	ImpactRating float64 `json:"impactRating"`
	Rating       float64 `json:"rating"`

	// Side-specific stats
	TDamage        int     `json:"TDamage"`
	CtDamage       int     `json:"ctDamage"`
	TImpactPoints  float64 `json:"TImpactPoints"`
	TWinPoints     float64 `json:"TWinPoints"`
	TOK            int     `json:"TOK"` // T-side opening kills
	TOL            int     `json:"TOL"` // T-side opening losses
	CtImpactPoints float64 `json:"ctImpactPoints"`
	CtWinPoints    float64 `json:"ctWinPoints"`
	CtOK           int     `json:"ctOK"` // CT-side opening kills
	CtOL           int     `json:"ctOL"` // CT-side opening losses
	TKills         uint8   `json:"TKills"`
	TDeaths        uint8   `json:"TDeaths"`
	TKAST          float64 `json:"TKAST"`
	TKASTRounds    float64 `json:"TKASTRounds"`
	TADR           float64 `json:"TADR"`
	CtKills        uint8   `json:"ctKills"`
	CtDeaths       uint8   `json:"ctDeaths"`
	CtKAST         float64 `json:"ctKAST"`
	CtKASTRounds   float64 `json:"ctKASTRounds"`
	CtADR          float64 `json:"ctADR"`
	TRounds        int     `json:"TRounds"`
	CtRounds       int     `json:"ctRounds"`
	CtRating       float64 `json:"ctRating"`
	CtImpactRating float64 `json:"ctImpactRating"`
	TRating        float64 `json:"TRating"`
	TImpactRating  float64 `json:"TImpactRating"`

	// ============================================
	// Ecorating-specific stats (not in demoScrape2)
	// ============================================

	// Clutch attempts (demoScrape2 only tracks wins)
	EcoClutch1v1Attempts int `json:"ecoClutch1v1Attempts"`
	EcoClutch1v2Attempts int `json:"ecoClutch1v2Attempts"`
	EcoClutch1v3Attempts int `json:"ecoClutch1v3Attempts"`
	EcoClutch1v4Attempts int `json:"ecoClutch1v4Attempts"`
	EcoClutch1v5Attempts int `json:"ecoClutch1v5Attempts"`

	// Probability-based metrics
	EcoProbabilitySwing         float64 `json:"ecoProbabilitySwing"`
	EcoProbabilitySwingPerRound float64 `json:"ecoProbabilitySwingPerRound"`
	EcoTProbabilitySwing        float64 `json:"ecoTProbabilitySwing"`
	EcoCTProbabilitySwing       float64 `json:"ecoCTProbabilitySwing"`

	// Economic impact
	EcoKillValue  float64 `json:"ecoKillValue"`
	EcoDeathValue float64 `json:"ecoDeathValue"`
	EcoDuelSwing  float64 `json:"ecoDuelSwing"`

	// Advanced ratings
	EcoFinalRating float64 `json:"ecoFinalRating"`
	EcoHLTVRating  float64 `json:"ecoHLTVRating"`
	EcoTEcoRating  float64 `json:"ecoTEcoRating"`
	EcoCTEcoRating float64 `json:"ecoCTEcoRating"`

	// Trade details
	EcoTradeDenials        int `json:"ecoTradeDenials"`
	EcoTradeKills          int `json:"ecoTradeKills"`
	EcoFastTrades          int `json:"ecoFastTrades"`
	EcoOpeningDeathsTraded int `json:"ecoOpeningDeathsTraded"`

	// AWP stats
	EcoAWPOpeningKills    int `json:"ecoAWPOpeningKills"`
	EcoAWPMultiKillRounds int `json:"ecoAWPMultiKillRounds"`
	EcoAWPDeaths          int `json:"ecoAWPDeaths"`

	// Time-based stats
	EcoTimeAlivePerRound float64 `json:"ecoTimeAlivePerRound"`
	EcoAvgTimeToDeath    float64 `json:"ecoAvgTimeToDeath"`
	EcoAvgTimeToKill     float64 `json:"ecoAvgTimeToKill"`
}
