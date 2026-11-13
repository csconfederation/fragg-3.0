// Package row is the payload's one row model: a player's, team's and round's stats as per-family
// structs. A struct is a family's fields only, so each JSON name lives in exactly one struct
// (encoding/json drops duplicate names at the same depth).
//
// Tags: `fold:"-"` marks a round-only field that the fold's sum skips; `json:"-"` an internal sum.
// Counters are int and floats float64. Steam64 values are uint64 with `,string`.
package row

// Player is one player's row: a round row, a whole-match row (side 4) or a side row (2 T, 3 CT).
type Player struct {
	ID
	Combat
	KAST
	Trades
	Openings
	Clutches
	Utility
	Objectives
	Survival
	Economy
	Positioning
	Swing
	Rates      // fold only
	Ratings    // fold only
	Projection // fold only
	Buys       // fold only
}

// ID is who the row is for.
type ID struct {
	Name         string `json:"name"`
	SteamID      string `json:"steamID"`
	IsBot        bool   `json:"isBot"`
	TeamENUM     int    `json:"teamENUM" fold:"-"`
	TeamClanName string `json:"teamClanName"`
	Side         int    `json:"side"`
	Rounds       int    `json:"rounds"`
	Health       int    `json:"health" fold:"-"`
}

type Combat struct {
	Damage             int `json:"damage"`
	Kills              int `json:"kills"`
	Assists            int `json:"assists"`
	Deaths             int `json:"deaths"`
	NadeDmg            int `json:"nadeDmg"`
	InfernoDmg         int `json:"infernoDmg"`
	UtilDmg            int `json:"utilDmg"`
	Hs                 int `json:"hs"`
	AwpKills           int `json:"awpKills"`
	DamageTaken        int `json:"damageTaken"`
	SuppRounds         int `json:"suppRounds"`
	SuppDamage         int `json:"suppDamage"`
	Eac                int `json:"eac"`
	CtAWP              int `json:"ctAWP" fold:"-"` // the fold adds it once, to the whole row
	AssistedKills      int `json:"assistedKills"`
	AWPMultiKillRounds int `json:"awpMultiKillRounds"`
	AWPDeaths          int `json:"awpDeaths"`
	RoundsWithAWPKill  int `json:"roundsWithAWPKill"`
	KnifeKills         int `json:"knifeKills"`
	PistolVsRifleKills int `json:"pistolVsRifleKills"`
	UtilityKills       int `json:"utilityKills"`
	PerfectKills       int `json:"perfectKills"`

	// Eco's kills, damage and AWP kills, by its gates, for the eco ratings (D31 unifies them with the CSC ones).
	EcoKills    int `json:"-"`
	EcoDamage   int `json:"-"`
	EcoAWPKills int `json:"-"`
}

type KAST struct {
	KastRounds    float64 `json:"kastRounds"`
	EcoKastRounds float64 `json:"-"` // eco's KAST rounds (its kill gate), for the eco ratings
}

type Trades struct {
	Trades              int `json:"trades"`
	Traded              int `json:"traded"`
	TradeKills          int `json:"tradeKills"`
	TradeDenials        int `json:"tradeDenials"`
	FastTrades          int `json:"fastTrades"`
	OpeningDeathsTraded int `json:"openingDeathsTraded"`
	SavedByTeammate     int `json:"savedByTeammate"`
	SavedTeammate       int `json:"savedTeammate"`
}

type Openings struct {
	Ok                    int `json:"ok"`
	Ol                    int `json:"ol"`
	TwoK                  int `json:"twoK"`
	ThreeK                int `json:"threeK"`
	FourK                 int `json:"fourK"`
	FiveK                 int `json:"fiveK"`
	Entries               int `json:"entries"`
	Rwk                   int `json:"rwk"`
	AWPOpeningKills       int `json:"awpOpeningKills"`
	ManAdvantageKills     int `json:"manAdvantageKills"`
	ManDisadvantageDeaths int `json:"manDisadvantageDeaths"`
}

type Clutches struct {
	Cl1               int `json:"cl_1"`
	Cl2               int `json:"cl_2"`
	Cl3               int `json:"cl_3"`
	Cl4               int `json:"cl_4"`
	Cl5               int `json:"cl_5"`
	Clutch1v1Attempts int `json:"clutch1v1Attempts"`
	Clutch1v2Attempts int `json:"clutch1v2Attempts"`
	Clutch1v3Attempts int `json:"clutch1v3Attempts"`
	Clutch1v4Attempts int `json:"clutch1v4Attempts"`
	Clutch1v5Attempts int `json:"clutch1v5Attempts"`
}

type Utility struct {
	Ef                 int     `json:"ef"`
	FAss               int     `json:"FAss"`
	EnemyFlashTime     float64 `json:"enemyFlashTime"`
	NadesThrown        int     `json:"nadesThrown"`
	FiresThrown        int     `json:"firesThrown"`
	FlashThrown        int     `json:"flashThrown"`
	SmokeThrown        int     `json:"smokeThrown"`
	MostRecentFlasher  uint64  `json:"mostRecentFlasher,string" fold:"-"`
	MostRecentFlashVal float64 `json:"mostRecentFlashVal" fold:"-"`
}

type Objectives struct {
	KillPoints   float64 `json:"killPoints"`
	ImpactPoints float64 `json:"impactPoints"`
	WinPoints    float64 `json:"winPoints"`
	RF           int     `json:"RF"`
	RA           int     `json:"RA"`
	Mip          float64 `json:"mip"`
	Rws          float64 `json:"rws"` // the round value on a round row, the rate on a fold row
	ExitFrags    int     `json:"exitFrags"`
}

type Survival struct {
	DeathTick      int     `json:"deathTick" fold:"-"`
	DeathPlacement float64 `json:"deathPlacement"` // the round value on a round row, the rate on a fold row
	TicksAlive     int     `json:"ticksAlive"`
	Saves          int     `json:"saves"`
	EarlyDeaths    int     `json:"earlyDeaths"`
	LeftAlive      int     `json:"-" fold:"-"` // disconnected without dying (D44): not a survivor, for the fold's HLTV survivals

	// Eco's seconds into the round of the last death and of survival, for the fold's time sums; 0 on the bot row.
	EcoDeathTime float64 `json:"-"`
	EcoTimeAlive float64 `json:"-"`
}

type Economy struct {
	EcoKillValue        float64 `json:"ecoKillValue"`
	EcoDeathValue       float64 `json:"ecoDeathValue"`
	KillsVsLowerBuy     int     `json:"killsVsLowerBuy"`
	KillsVsMuchLowerBuy int     `json:"killsVsMuchLowerBuy"`
}

type Positioning struct {
	LurkerBlips         int `json:"lurkerBlips" fold:"-"`
	DistanceToTeammates int `json:"distanceToTeammates" fold:"-"`
	LurkRounds          int `json:"lurkRounds"`
}

type Swing struct {
	EcoProbabilitySwing float64 `json:"ecoProbabilitySwing"`
	EcoAdjustedKills    float64 `json:"ecoAdjustedKills"`
	AvgTimeToKill       float64 `json:"avgTimeToKill"`

	// Seconds from first damage to the kill, over the kills with prior damage, for the fold's average.
	TimeToKillSum float64 `json:"-"`
	KillsWithTTK  int     `json:"-"`
}

// Rates are computed by the fold from the sums.
type Rates struct {
	Adr                         float64 `json:"adr"`
	KR                          float64 `json:"KR"`
	DrDiff                      float64 `json:"drDiff"`
	Kast                        float64 `json:"kast"`
	Tr                          float64 `json:"tr"`
	Atd                         int     `json:"atd"`
	KillPointAvg                float64 `json:"killPointAvg"`
	Iiwr                        float64 `json:"iiwr"`
	UtilThrown                  int     `json:"utilThrown"`
	AWPKillsPerRound            float64 `json:"awpKillsPerRound"`
	TimeAlivePerRound           float64 `json:"timeAlivePerRound"`
	AvgTimeToDeath              float64 `json:"avgTimeToDeath"`
	EcoProbabilitySwingPerRound float64 `json:"ecoProbabilitySwingPerRound"`
	EcoDuelSwing                float64 `json:"ecoDuelSwing"`
	EcoDuelSwingPerRound        float64 `json:"ecoDuelSwingPerRound"`
	Wlp                         float64 `json:"wlp"`
}

// Ratings are the whole-row and side ratings.
type Ratings struct {
	Rating                float64 `json:"rating"`
	ImpactRating          float64 `json:"impactRating"`
	HLTVRating            float64 `json:"hltvRating"`
	HLTVCtRating          float64 `json:"hltvCtRating"`
	HLTVTRating           float64 `json:"hltvTRating"`
	SwingRating           float64 `json:"swing_rating"`
	EcoCTEcoRating        float64 `json:"ecoCTEcoRating"`
	EcoTEcoRating         float64 `json:"ecoTEcoRating"`
	EcoSwingDisplayRating float64 `json:"ecoSwingDisplayRating"`
}

// Projection is the whole-match row's T and CT columns, taken from the side rows, and the
// whole-only rating inputs.
type Projection struct {
	TDamage               int     `json:"TDamage"`
	CtDamage              int     `json:"ctDamage"`
	TImpactPoints         float64 `json:"TImpactPoints"`
	TWinPoints            float64 `json:"TWinPoints"`
	TOK                   int     `json:"TOK"`
	TOL                   int     `json:"TOL"`
	CtImpactPoints        float64 `json:"ctImpactPoints"`
	CtWinPoints           float64 `json:"ctWinPoints"`
	CtOK                  int     `json:"ctOK"`
	CtOL                  int     `json:"ctOL"`
	TKills                int     `json:"TKills"`
	TDeaths               int     `json:"TDeaths"`
	TKAST                 float64 `json:"TKAST"`
	TKASTRounds           float64 `json:"TKASTRounds"`
	TADR                  float64 `json:"TADR"`
	CtKills               int     `json:"ctKills"`
	CtDeaths              int     `json:"ctDeaths"`
	CtKAST                float64 `json:"ctKAST"`
	CtKASTRounds          float64 `json:"ctKASTRounds"`
	CtADR                 float64 `json:"ctADR"`
	TRounds               int     `json:"TRounds"`
	CtRounds              int     `json:"ctRounds"`
	CtRating              float64 `json:"ctRating"`
	CtImpactRating        float64 `json:"ctImpactRating"`
	TRating               float64 `json:"TRating"`
	TImpactRating         float64 `json:"TImpactRating"`
	TADP                  float64 `json:"TADP"`
	CtADP                 float64 `json:"ctADP"`
	TRF                   int     `json:"TRF"`
	EcoTProbabilitySwing  float64 `json:"ecoTProbabilitySwing"`
	EcoCTProbabilitySwing float64 `json:"ecoCTProbabilitySwing"`
	TTeamsWinPoints       float64 `json:"TTeamsWinPoints"`
	CtTeamsWinPoints      float64 `json:"ctTeamsWinPoints"`
	TWinPointsNormalizer  int     `json:"TWinPointsNormalizer"`
	CtWinPointsNormalizer int     `json:"ctWinPointsNormalizer"`
	TeamsWinPoints        float64 `json:"teamsWinPoints"`
	WinPointsNormalizer   int     `json:"winPointsNormalizer"`
}

// Buys are the rounds, round wins and CSC rating over the rounds of each of the player's team's
// buys; whole-match rows only. Anti-eco is a full buy against an eco.
type Buys struct {
	PistolRounds      int     `json:"pistolRounds"`
	PistolRoundsWon   int     `json:"pistolRoundsWon"`
	PistolRoundRating float64 `json:"pistolRoundRating"`
	EcoBuyRounds      int     `json:"ecoBuyRounds"`
	EcoBuyRoundsWon   int     `json:"ecoBuyRoundsWon"`
	EcoBuyRating      float64 `json:"ecoBuyRating"`
	ForceBuyRounds    int     `json:"forceBuyRounds"`
	ForceBuyRoundsWon int     `json:"forceBuyRoundsWon"`
	ForceBuyRating    float64 `json:"forceBuyRating"`
	FullBuyRounds     int     `json:"fullBuyRounds"`
	FullBuyRoundsWon  int     `json:"fullBuyRoundsWon"`
	FullBuyRating     float64 `json:"fullBuyRating"`
	AntiEcoRounds     int     `json:"antiEcoRounds"`
	AntiEcoRoundsWon  int     `json:"antiEcoRoundsWon"`
	AntiEcoRating     float64 `json:"antiEcoRating"`
}
