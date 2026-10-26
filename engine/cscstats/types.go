package cscstats

// The aliases export the payload's rows to engine/fold and engine/payload.
type (
	PlayerStats = playerStats
	Round       = round
	TeamStats   = teamStats
)

// Game is the committed rounds' CSC stats; engine/fold folds them into the player and team rows.
type Game struct {
	Rounds           []*round
	TickRate         int
	TotalPlayerStats map[uint64]*playerStats
	CtPlayerStats    map[uint64]*playerStats
	TPlayerStats     map[uint64]*playerStats
	TotalTeamStats   map[string]*teamStats
}

type teamStats struct {
	WinPoints      float64 `json:"winPoints"`
	ImpactPoints   float64 `json:"impactPoints"`
	TWinPoints     float64 `json:"TWinPoints"`
	CtWinPoints    float64 `json:"ctWinPoints"`
	TImpactPoints  float64 `json:"TImpactPoints"`
	CtImpactPoints float64 `json:"ctImpactPoints"`
	FourVFiveW     int     `json:"fourVFiveW"`
	FourVFiveS     int     `json:"fourVFiveS"`
	FiveVFourW     int     `json:"fiveVFourW"`
	FiveVFourS     int     `json:"fiveVFourS"`
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

	//kinda garbo
	Normalizer int `json:"normalizer"`
}

type round struct {
	//round value
	RoundNum            int8                    `json:"roundNum"`
	StartingTick        int                     `json:"startingTick"`
	EndingTick          int                     `json:"endingTick"`
	PlayerStats         map[uint64]*playerStats `json:"-"` // the payload lists them (engine/payload)
	TeamStats           map[string]*teamStats   `json:"-"`
	InitTerroristCount  int                     `json:"initTerroristCount"`
	InitCTerroristCount int                     `json:"initCTerroristCount"`
	WinnerClanName      string                  `json:"winnerClanName"`
	//winnerID            int //this is the unique ID which should not change BUT IT DOES
	WinnerENUM         int     `json:"winnerENUM"` //this effectively represents the side that won: 2 (T) or 3 (CT)
	IntegrityCheck     bool    `json:"integrityCheck"`
	Planter            uint64  `json:"planter,string"`
	Defuser            uint64  `json:"defuser,string"`
	EndDueToBombEvent  bool    `json:"endDueToBombEvent"`
	WinTeamDmg         int     `json:"winTeamDmg"`
	ServerNormalizer   int     `json:"serverNormalizer"`
	ServerImpactPoints float64 `json:"serverImpactPoints"`
	KnifeRound         bool    `json:"knifeRound"`
	RoundEndReason     string  `json:"roundEndReason"`

	BombStartTick int `json:"bombStartTick"`
}

type playerStats struct {
	Name    string `json:"name"`
	SteamID string `json:"steamID"`
	IsBot   bool   `json:"isBot"`
	//teamID  int
	TeamENUM     int    `json:"teamENUM"`
	TeamClanName string `json:"teamClanName"`
	Side         int    `json:"side"`
	Rounds       int    `json:"rounds"`
	//playerPoints float32
	//teamPoints float32
	Damage              int     `json:"damage"`
	Kills               uint8   `json:"kills"`
	Assists             uint8   `json:"assists"`
	Deaths              uint8   `json:"deaths"`
	DeathTick           int     `json:"deathTick"`
	DeathPlacement      float64 `json:"deathPlacement"`
	TicksAlive          int     `json:"ticksAlive"`
	Trades              int     `json:"trades"`
	Traded              int     `json:"traded"`
	Ok                  int     `json:"ok"`
	Ol                  int     `json:"ol"`
	Cl_1                int     `json:"cl_1"`
	Cl_2                int     `json:"cl_2"`
	Cl_3                int     `json:"cl_3"`
	Cl_4                int     `json:"cl_4"`
	Cl_5                int     `json:"cl_5"`
	TwoK                int     `json:"twoK"`
	ThreeK              int     `json:"threeK"`
	FourK               int     `json:"fourK"`
	FiveK               int     `json:"fiveK"`
	NadeDmg             int     `json:"nadeDmg"`
	InfernoDmg          int     `json:"infernoDmg"`
	UtilDmg             int     `json:"utilDmg"`
	Ef                  int     `json:"ef"`
	FAss                int     `json:"FAss"`
	EnemyFlashTime      float64 `json:"enemyFlashTime"`
	Hs                  int     `json:"hs"`
	KastRounds          float64 `json:"kastRounds"`
	Saves               int     `json:"saves"`
	Entries             int     `json:"entries"`
	KillPoints          float64 `json:"killPoints"`
	ImpactPoints        float64 `json:"impactPoints"`
	WinPoints           float64 `json:"winPoints"`
	AwpKills            int     `json:"awpKills"`
	RF                  int     `json:"RF"`
	RA                  int     `json:"RA"`
	NadesThrown         int     `json:"nadesThrown"`
	FiresThrown         int     `json:"firesThrown"`
	FlashThrown         int     `json:"flashThrown"`
	SmokeThrown         int     `json:"smokeThrown"`
	DamageTaken         int     `json:"damageTaken"`
	SuppRounds          int     `json:"suppRounds"`
	SuppDamage          int     `json:"suppDamage"`
	LurkerBlips         int     `json:"lurkerBlips"`
	DistanceToTeammates int     `json:"distanceToTeammates"`
	LurkRounds          int     `json:"lurkRounds"`
	Wlp                 float64 `json:"wlp"`
	Mip                 float64 `json:"mip"`
	Rws                 float64 `json:"rws"` //round win shares
	Eac                 int     `json:"eac"` //effective assist contributions

	Rwk int `json:"rwk"` //rounds with Kills

	//derived
	UtilThrown   int     `json:"utilThrown"`
	Atd          int     `json:"atd"`
	Kast         float64 `json:"kast"`
	KillPointAvg float64 `json:"killPointAvg"`
	Iiwr         float64 `json:"iiwr"`
	Adr          float64 `json:"adr"`
	DrDiff       float64 `json:"drDiff"`
	KR           float64 `json:"KR"`
	Tr           float64 `json:"tr"` //trade ratio
	ImpactRating float64 `json:"impactRating"`
	Rating       float64 `json:"rating"`

	//side specific
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
	TKills                uint8   `json:"TKills"`
	TDeaths               uint8   `json:"TDeaths"`
	TKAST                 float64 `json:"TKAST"`
	TKASTRounds           float64 `json:"TKASTRounds"`
	TADR                  float64 `json:"TADR"`
	CtKills               uint8   `json:"ctKills"`
	CtDeaths              uint8   `json:"ctDeaths"`
	CtKAST                float64 `json:"ctKAST"`
	CtKASTRounds          float64 `json:"ctKASTRounds"`
	CtADR                 float64 `json:"ctADR"`
	TTeamsWinPoints       float64 `json:"TTeamsWinPoints"`
	CtTeamsWinPoints      float64 `json:"ctTeamsWinPoints"`
	TWinPointsNormalizer  int     `json:"TWinPointsNormalizer"`
	CtWinPointsNormalizer int     `json:"ctWinPointsNormalizer"`
	TRounds               int     `json:"TRounds"`
	CtRounds              int     `json:"ctRounds"`
	CtRating              float64 `json:"ctRating"`
	CtImpactRating        float64 `json:"ctImpactRating"`
	TRating               float64 `json:"TRating"`
	TImpactRating         float64 `json:"TImpactRating"`
	TADP                  float64 `json:"TADP"`
	CtADP                 float64 `json:"ctADP"`

	TRF   int `json:"TRF"`
	CtAWP int `json:"ctAWP"`

	//kinda garbo
	TeamsWinPoints      float64 `json:"teamsWinPoints"`
	WinPointsNormalizer int     `json:"winPointsNormalizer"`

	//"flags"
	Health             int            `json:"health"`
	MostRecentFlasher  uint64         `json:"mostRecentFlasher,string"`
	MostRecentFlashVal float64        `json:"mostRecentFlashVal"`
	DamageList         map[uint64]int `json:"-"`

	// The eco stats, which engine/fold merges from engine/eco (#591: one definition per concept).

	// Primary eco rating, stored as swing_rating.
	SwingRating float64 `json:"swing_rating"`

	// Probability-swing metrics (eco impact model).
	EcoProbabilitySwing         float64 `json:"ecoProbabilitySwing"`
	EcoProbabilitySwingPerRound float64 `json:"ecoProbabilitySwingPerRound"`
	EcoTProbabilitySwing        float64 `json:"ecoTProbabilitySwing"`
	EcoCTProbabilitySwing       float64 `json:"ecoCTProbabilitySwing"`

	// Eco duel economy.
	EcoKillValue         float64 `json:"ecoKillValue"`
	EcoDeathValue        float64 `json:"ecoDeathValue"`
	EcoDuelSwing         float64 `json:"ecoDuelSwing"`
	EcoDuelSwingPerRound float64 `json:"ecoDuelSwingPerRound"`
	EcoAdjustedKills     float64 `json:"ecoAdjustedKills"`

	// Eco ratings (HLTV 2.0 + eco side ratings), kept distinct from the
	// demoScrape2 rating/impactRating/ctRating/TRating fields above.
	HLTVRating     float64 `json:"hltvRating"`
	HLTVCtRating   float64 `json:"hltvCtRating"`
	HLTVTRating    float64 `json:"hltvTRating"`
	EcoCTEcoRating float64 `json:"ecoCTEcoRating"`
	EcoTEcoRating  float64 `json:"ecoTEcoRating"`
	// Eco display swing rating (1 + 10*swing/round). Distinct from
	// swing_rating (which is eco FinalRating).
	EcoSwingDisplayRating float64 `json:"ecoSwingDisplayRating"`
	PistolRoundRating     float64 `json:"pistolRoundRating"`

	// Eco assisted kills (assist count only) — distinct from demoScrape2 eac.
	AssistedKills int `json:"assistedKills"`

	// Clutch attempts (demoScrape2 only tracks wins in cl_*).
	Clutch1v1Attempts int `json:"clutch1v1Attempts"`
	Clutch1v2Attempts int `json:"clutch1v2Attempts"`
	Clutch1v3Attempts int `json:"clutch1v3Attempts"`
	Clutch1v4Attempts int `json:"clutch1v4Attempts"`
	Clutch1v5Attempts int `json:"clutch1v5Attempts"`

	// Trade detail (eco 5s-window trade model), distinct from demoScrape2 trades/traded.
	TradeKills          int `json:"tradeKills"`
	TradeDenials        int `json:"tradeDenials"`
	FastTrades          int `json:"fastTrades"`
	OpeningDeathsTraded int `json:"openingDeathsTraded"`

	// AWP extras.
	AWPOpeningKills    int     `json:"awpOpeningKills"`
	AWPMultiKillRounds int     `json:"awpMultiKillRounds"`
	AWPDeaths          int     `json:"awpDeaths"`
	RoundsWithAWPKill  int     `json:"roundsWithAWPKill"`
	AWPKillsPerRound   float64 `json:"awpKillsPerRound"`

	// Time-based stats. atd above is demoScrape2 (avg time alive); these are eco's.
	TimeAlivePerRound float64 `json:"timeAlivePerRound"`
	AvgTimeToDeath    float64 `json:"avgTimeToDeath"`
	AvgTimeToKill     float64 `json:"avgTimeToKill"`

	// Other unique eco metrics.
	ManAdvantageKills     int `json:"manAdvantageKills"`
	ManDisadvantageDeaths int `json:"manDisadvantageDeaths"`
	ExitFrags             int `json:"exitFrags"`
	KnifeKills            int `json:"knifeKills"`
	PistolVsRifleKills    int `json:"pistolVsRifleKills"`
	KillsVsLowerBuy       int `json:"killsVsLowerBuy"`
	KillsVsMuchLowerBuy   int `json:"killsVsMuchLowerBuy"`
	UtilityKills          int `json:"utilityKills"`
	PerfectKills          int `json:"perfectKills"`
	EarlyDeaths           int `json:"earlyDeaths"`
	SavedByTeammate       int `json:"savedByTeammate"`
	SavedTeammate         int `json:"savedTeammate"`
}
