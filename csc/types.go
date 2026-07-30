package csc

import (
	"encoding/json"
	"strconv"
)

// PlayerStats is an exported alias for the demoScrape2-compatible per-player
// stats struct, letting other packages (e.g. export) read/write its fields
// while keeping the ported field layout identical to demoScrape2.
type PlayerStats = playerStats

type Game struct {
	//winnerID         int
	CoreID                   string                  `json:"coreID"`
	MapNum                   int                     `json:"mapNum"`
	WinnerClanName           string                  `json:"winnerClanName"`
	Result                   string                  `json:"result"`
	Rounds                   []*round                `json:"rounds"`
	PotentialRound           *round                  `json:"potentialRound"`
	Teams                    map[string]*team        `json:"teams"`
	Flags                    flag                    `json:"flags"`
	MapName                  string                  `json:"mapName"`
	TickRate                 int                     `json:"tickRate"`
	TickLength               int                     `json:"tickLength"`
	RoundsToWin              int                     `json:"roundsToWin"` //30 or 16
	TotalPlayerStats         map[uint64]*playerStats `json:"totalPlayerStats"`
	CtPlayerStats            map[uint64]*playerStats `json:"ctPlayerStats"`
	TPlayerStats             map[uint64]*playerStats `json:"TPlayerStats"`
	TotalTeamStats           map[string]*teamStats   `json:"totalTeamStats"`
	ReconnectedPlayers       map[uint64]bool         `json:"reconnectedPlayers"`       // Map of SteamID to reconnection status
	ConnectedAfterRoundStart map[uint64]bool         `json:"ConnectedAfterRoundStart"` // Map of SteamID to reconnection status
	PlayerOrder              []uint64                `json:"playerOrder"`
	TeamOrder                []string                `json:"teamOrder"`
	TotalRounds              int                     `json:"totalRounds"`
	TotalWPAlog              []*wpalog               `json:"totalWPAlog"`
	// EcoStatsOK is true only when the eco-rating merge completed successfully.
	// False means eco* / swing_rating fields are unset/zero and must not be
	// treated as genuine zero ratings.
	EcoStatsOK bool `json:"ecoStatsOK"`
}

// MarshalJSON preserves Game's public numeric PlayerOrder API while emitting
// those Steam64s as strings at the external JSON boundary (matching
// demoScrape2's JSON contract).
func (game Game) MarshalJSON() ([]byte, error) {
	type gameJSON Game
	var playerOrder []string
	if game.PlayerOrder != nil {
		playerOrder = make([]string, len(game.PlayerOrder))
		for i, id := range game.PlayerOrder {
			playerOrder[i] = strconv.FormatUint(id, 10)
		}
	}
	return json.Marshal(struct {
		gameJSON
		PlayerOrder []string `json:"playerOrder"`
	}{
		gameJSON:    gameJSON(game),
		PlayerOrder: playerOrder,
	})
}

type flag struct {
	//all our sentinals and shit
	HasGameStarted            bool `json:"hasGameStarted"`
	IsGameLive                bool `json:"isGameLive"`
	IsGameOver                bool `json:"isGameOver"`
	InRound                   bool `json:"inRound"`
	PrePlant                  bool `json:"prePlant"`
	PostPlant                 bool `json:"postPlant"`
	PostWinCon                bool `json:"postWinCon"`
	RoundIntegrityStart       int  `json:"roundIntegrityStart"`
	RoundIntegrityEnd         int  `json:"roundIntegrityEnd"`
	RoundIntegrityEndOfficial int  `json:"roundIntegrityEndOfficial"`

	//for the round (gets reset on a new round) maybe should be in a new struct
	TAlive             int    `json:"TAlive"`
	CtAlive            int    `json:"ctAlive"`
	TMoney             bool   `json:"TMoney"`
	TClutchVal         int    `json:"TClutchVal"`
	CtClutchVal        int    `json:"ctClutchVal"`
	TClutchSteam       uint64 `json:"TClutchSteam,string"`
	CtClutchSteam      uint64 `json:"ctClutchSteam,string"`
	OpeningKill        bool   `json:"openingKill"`
	LastTickProcessed  int    `json:"lastTickProcessed"`
	TicksProcessed     int    `json:"ticksProcessed"`
	DidRoundEndFire    bool   `json:"didRoundEndFire"`
	DidScoreUpdateFire bool   `json:"didScoreUpdateFire"`
	RoundStartedAt     int    `json:"roundStartedAt"`
	HaveInitRound      bool   `json:"haveInitRound"`
}

type team struct {
	//id    int //meaningless?
	Name          string `json:"name"`
	Score         int    `json:"score"`
	ScoreAdjusted int    `json:"scoreAdjusted"`
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
	PlayerStats         map[uint64]*playerStats `json:"playerStats"`
	TeamStats           map[string]*teamStats   `json:"teamStats"`
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

	WPAlog        []*wpalog `json:"WPAlog"`
	BombStartTick int       `json:"bombStartTick"`
}

type wpalog struct {
	Round               int `json:"round"`
	Tick                int `json:"tick"`
	Clock               int `json:"clock"`
	Planted             int `json:"planted"`
	CtAlive             int `json:"ctAlive"`
	TAlive              int `json:"TAlive"`
	CtEquipVal          int `json:"ctEquipVal"`
	TEquipVal           int `json:"TEquipVal"`
	CtFlashes           int `json:"ctFlashes"`
	CtSmokes            int `json:"ctSmokes"`
	CtMolys             int `json:"ctMolys"`
	CtFrags             int `json:"ctFrags"`
	TFlashes            int `json:"TFlashes"`
	TSmokes             int `json:"TSmokes"`
	TMolys              int `json:"TMolys"`
	TFrags              int `json:"TFrags"`
	ClosestCTDisttoBomb int `json:"closestCTDisttoBomb"`
	Kits                int `json:"kits"`
	CtArmor             int `json:"ctArmor"`
	TArmor              int `json:"TArmor"`
	Winner              int `json:"winner"`
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
	TradeList          map[uint64]int `json:"tradeList"`
	MostRecentFlasher  uint64         `json:"mostRecentFlasher,string"`
	MostRecentFlashVal float64        `json:"mostRecentFlashVal"`
	DamageList         map[uint64]int `json:"damageList"`

	// ================================================================
	// Additive eco-rating metrics (NOT part of demoScrape2).
	// These are merged in by the export package from eco-rating's own
	// pipeline. They never overwrite a demoScrape2 field above; where a
	// concept exists in both parsers with a different calculation, the
	// eco value lives here under a distinct name (per the field-merge
	// decisions) so nothing is duplicated or silently replaced.
	// ================================================================

	// Primary eco rating. Exported as swing_rating (the new rating),
	// intentionally NOT as final_rating.
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
	EcoHLTVRating   float64 `json:"ecoHLTVRating"`
	EcoHLTVCtRating float64 `json:"ecoHLTVCtRating"`
	EcoHLTVTRating  float64 `json:"ecoHLTVTRating"`
	EcoCTEcoRating  float64 `json:"ecoCTEcoRating"`
	EcoTEcoRating   float64 `json:"ecoTEcoRating"`
	// Eco display swing rating (1 + 10*swing/round). Distinct from
	// swing_rating (which is eco FinalRating).
	EcoSwingDisplayRating float64 `json:"ecoSwingDisplayRating"`
	EcoPistolRoundRating  float64 `json:"ecoPistolRoundRating"`

	// Eco assisted kills (assist count only) — distinct from demoScrape2 eac.
	EcoAssistedKills int `json:"ecoAssistedKills"`

	// Clutch attempts (demoScrape2 only tracks wins in cl_*).
	EcoClutch1v1Attempts int `json:"ecoClutch1v1Attempts"`
	EcoClutch1v2Attempts int `json:"ecoClutch1v2Attempts"`
	EcoClutch1v3Attempts int `json:"ecoClutch1v3Attempts"`
	EcoClutch1v4Attempts int `json:"ecoClutch1v4Attempts"`
	EcoClutch1v5Attempts int `json:"ecoClutch1v5Attempts"`

	// Trade detail (eco 5s-window trade model), distinct from demoScrape2 trades/traded.
	EcoTradeKills          int `json:"ecoTradeKills"`
	EcoTradeDenials        int `json:"ecoTradeDenials"`
	EcoFastTrades          int `json:"ecoFastTrades"`
	EcoOpeningDeathsTraded int `json:"ecoOpeningDeathsTraded"`

	// AWP extras.
	EcoAWPOpeningKills    int     `json:"ecoAWPOpeningKills"`
	EcoAWPMultiKillRounds int     `json:"ecoAWPMultiKillRounds"`
	EcoAWPDeaths          int     `json:"ecoAWPDeaths"`
	EcoRoundsWithAWPKill  int     `json:"ecoRoundsWithAWPKill"`
	EcoAWPKillsPerRound   float64 `json:"ecoAWPKillsPerRound"`

	// Time-based stats. atd above is demoScrape2 (avg time alive); these are eco's.
	EcoTimeAlivePerRound float64 `json:"ecoTimeAlivePerRound"`
	EcoAvgTimeToDeath    float64 `json:"ecoAvgTimeToDeath"`
	EcoAvgTimeToKill     float64 `json:"ecoAvgTimeToKill"`

	// Other unique eco metrics.
	EcoManAdvantageKills     int `json:"ecoManAdvantageKills"`
	EcoManDisadvantageDeaths int `json:"ecoManDisadvantageDeaths"`
	EcoExitFrags             int `json:"ecoExitFrags"`
	EcoKnifeKills            int `json:"ecoKnifeKills"`
	EcoPistolVsRifleKills    int `json:"ecoPistolVsRifleKills"`
	EcoLowBuyKills           int `json:"ecoLowBuyKills"`
	EcoDisadvantagedBuyKills int `json:"ecoDisadvantagedBuyKills"`
	EcoUtilityKills          int `json:"ecoUtilityKills"`
	EcoPerfectKills          int `json:"ecoPerfectKills"`
	EcoEarlyDeaths           int `json:"ecoEarlyDeaths"`
	EcoSavedByTeammate       int `json:"ecoSavedByTeammate"`
	EcoSavedTeammate         int `json:"ecoSavedTeammate"`
}

type Accolades struct {
	Awp        int `json:"awp"`
	Deagle     int `json:"deagle"`
	Knife      int `json:"knife"`
	Dinks      int `json:"dinks"`
	BlindKills int `json:"blindKills"`
	BombPlants int `json:"bombPlants"`
	Jumps      int `json:"jumps"`
	TeamDMG    int `json:"teamDMG"`
	SelfDMG    int `json:"selfDMG"`
	Ping       int `json:"ping"`
	PingPoints int `json:"pingPoints"`
	//footsteps         int //unnecessary processing?
	BombTaps          int `json:"bombTaps"`
	KillsThroughSmoke int `json:"killsThroughSmoke"`
	Penetrations      int `json:"penetrations"`
	NoScopes          int `json:"noScopes"`
	MidairKills       int `json:"midairKills"`
	CrouchedKills     int `json:"crouchedKills"`
	BombzoneKills     int `json:"bombzoneKills"`
	KillsWhileMoving  int `json:"killsWhileMoving"`
	MostMoneySpent    int `json:"mostMoneySpent"`
	MostShotsOnLegs   int `json:"mostShotsOnLegs"`
	ShotsFired        int `json:"shotsFired"`
	Ak                int `json:"ak"`
	M4                int `json:"m4"`
	Pistol            int `json:"pistol"`
	Scout             int `json:"scout"`
}
