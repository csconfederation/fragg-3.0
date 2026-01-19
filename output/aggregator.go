package output

import (
	"eco-rating/model"
	"eco-rating/rating"
)

// MultiKillStats holds multi-kill counts with explicit labels
type MultiKillStats struct {
	OneK   int `json:"1k"`
	TwoK   int `json:"2k"`
	ThreeK int `json:"3k"`
	FourK  int `json:"4k"`
	FiveK  int `json:"5k"`
}

// AggregatedStats holds cumulative stats for a player across multiple games
type AggregatedStats struct {
	SteamID    string `json:"steam_id"`
	Name       string `json:"name"`
	Tier       string `json:"tier"`
	GamesCount int    `json:"games_count"`

	// Cumulative totals
	RoundsPlayed int `json:"rounds_played"`
	RoundsWon    int `json:"rounds_won"`
	RoundsLost   int `json:"rounds_lost"`
	Kills        int `json:"kills"`
	Assists      int `json:"assists"`
	Deaths       int `json:"deaths"`
	Damage       int `json:"damage"`
	OpeningKills int `json:"opening_kills"`

	// Per-round averages (calculated from totals)
	ADR float64 `json:"adr"`
	KPR float64 `json:"kpr"`
	DPR float64 `json:"dpr"`

	PerfectKills int `json:"perfect_kills"`
	TradeDenials int `json:"trade_denials"`
	TradedDeaths int `json:"traded_deaths"`

	// Kill tracking
	RoundsWithKill      int `json:"rounds_with_kill"`
	RoundsWithMultiKill int `json:"rounds_with_multi_kill"`
	KillsInWonRounds    int `json:"kills_in_won_rounds"`
	DamageInWonRounds   int `json:"damage_in_won_rounds"`

	// AWP/Sniper stats
	AWPKills           int     `json:"awp_kills"`
	AWPKillsPerRound   float64 `json:"awp_kills_per_round"`
	RoundsWithAWPKill  int     `json:"rounds_with_awp_kill"`
	AWPMultiKillRounds int     `json:"awp_multi_kill_rounds"`
	AWPOpeningKills    int     `json:"awp_opening_kills"`

	MultiKills MultiKillStats `json:"multi_kills"`

	// Averages across games
	RoundImpact float64 `json:"round_impact"`
	Survival    float64 `json:"survival"`
	KAST        float64 `json:"kast"`
	EconImpact  float64 `json:"econ_impact"`

	// Eco-adjusted values (cumulative)
	EcoKillValue  float64 `json:"eco_kill_value"`
	EcoDeathValue float64 `json:"eco_death_value"`

	// Round swing stats
	RoundSwing   float64 `json:"round_swing"`
	ClutchRounds int     `json:"clutch_rounds"`
	ClutchWins   int     `json:"clutch_wins"`

	// Support stats
	SavedByTeammate     int `json:"saved_by_teammate"`
	SavedTeammate       int `json:"saved_teammate"`
	OpeningDeaths       int `json:"opening_deaths"`
	OpeningDeathsTraded int `json:"opening_deaths_traded"`
	SupportRounds       int `json:"support_rounds"`
	AssistedKills       int `json:"assisted_kills"`

	// Entry/Opening stats
	OpeningAttempts       int `json:"opening_attempts"`
	OpeningSuccesses      int `json:"opening_successes"`
	RoundsWonAfterOpening int `json:"rounds_won_after_opening"`
	AttackRounds          int `json:"attack_rounds"`

	// Clutch stats
	Clutch1v1Attempts int     `json:"clutch_1v1_attempts"`
	Clutch1v1Wins     int     `json:"clutch_1v1_wins"`
	TimeAlivePerRound float64 `json:"time_alive_per_round"`
	LastAliveRounds   int     `json:"last_alive_rounds"`
	SavesOnLoss       int     `json:"saves_on_loss"`

	// Utility stats
	UtilityDamage              int     `json:"utility_damage"`
	UtilityKills               int     `json:"utility_kills"`
	FlashesThrown              int     `json:"flashes_thrown"`
	FlashAssists               int     `json:"flash_assists"`
	EnemyFlashDurationPerRound float64 `json:"enemy_flash_duration_per_round"`
	TeamFlashCount             int     `json:"team_flash_count"`
	TeamFlashDurationPerRound  float64 `json:"team_flash_duration_per_round"`

	// Internal tracking for time-based stats (not exported)
	totalTimeAlive     float64
	totalEnemyFlashDur float64
	totalTeamFlashDur  float64

	// Misc stats
	ExitFrags                int     `json:"exit_frags"`
	AWPDeaths                int     `json:"awp_deaths"`
	AWPDeathsNoKill          int     `json:"awp_deaths_no_kill"`
	KnifeKills               int     `json:"knife_kills"`
	PistolVsRifleKills       int     `json:"pistol_vs_rifle_kills"`
	TradeKills               int     `json:"trade_kills"`
	FastTrades               int     `json:"fast_trades"`
	EarlyDeaths              int     `json:"early_deaths"`
	LowBuyKills              int     `json:"low_buy_kills"`
	LowBuyKillsPct           float64 `json:"low_buy_kills_pct"`
	DisadvantagedBuyKills    int     `json:"disadvantaged_buy_kills"`
	DisadvantagedBuyKillsPct float64 `json:"disadvantaged_buy_kills_pct"`

	// Pistol round stats
	PistolRoundsPlayed    int     `json:"pistol_rounds_played"`
	PistolRoundKills      int     `json:"pistol_round_kills"`
	PistolRoundDeaths     int     `json:"pistol_round_deaths"`
	PistolRoundDamage     int     `json:"pistol_round_damage"`
	PistolRoundsWon       int     `json:"pistol_rounds_won"`
	PistolRoundSurvivals  int     `json:"pistol_round_survivals"`
	PistolRoundMultiKills int     `json:"pistol_round_multi_kills"`
	PistolRoundRating     float64 `json:"pistol_round_rating"`

	// Per-side stats (T = Terrorist, CT = Counter-Terrorist)
	TRoundsPlayed        int     `json:"t_rounds_played"`
	TKills               int     `json:"t_kills"`
	TDeaths              int     `json:"t_deaths"`
	TDamage              int     `json:"t_damage"`
	TSurvivals           int     `json:"t_survivals"`
	TRoundsWithMultiKill int     `json:"t_rounds_with_multi_kill"`
	TEcoKillValue        float64 `json:"t_eco_kill_value"`
	TRoundSwing          float64 `json:"t_round_swing"`
	TKAST                float64 `json:"t_kast"`
	TClutchRounds        int     `json:"t_clutch_rounds"`
	TClutchWins          int     `json:"t_clutch_wins"`
	TRating              float64 `json:"t_rating"`     // HLTV 1.0 for T-side
	TEcoRating           float64 `json:"t_eco_rating"` // Eco Rating for T-side

	CTRoundsPlayed        int     `json:"ct_rounds_played"`
	CTKills               int     `json:"ct_kills"`
	CTDeaths              int     `json:"ct_deaths"`
	CTDamage              int     `json:"ct_damage"`
	CTSurvivals           int     `json:"ct_survivals"`
	CTRoundsWithMultiKill int     `json:"ct_rounds_with_multi_kill"`
	CTEcoKillValue        float64 `json:"ct_eco_kill_value"`
	CTRoundSwing          float64 `json:"ct_round_swing"`
	CTKAST                float64 `json:"ct_kast"`
	CTClutchRounds        int     `json:"ct_clutch_rounds"`
	CTClutchWins          int     `json:"ct_clutch_wins"`
	CTRating              float64 `json:"ct_rating"`     // HLTV 1.0 for CT-side
	CTEcoRating           float64 `json:"ct_eco_rating"` // Eco Rating for CT-side

	// Internal tracking for per-side multi-kills (not exported)
	tMultiKills  [6]int
	ctMultiKills [6]int

	// Ratings
	HLTVRating  float64 `json:"hltv_rating"`
	FinalRating float64 `json:"final_rating"`

	// Calculated per-round stats
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

	// Per-map stats
	MapRatings     map[string]float64 `json:"map_ratings"`
	MapGamesPlayed map[string]int     `json:"map_games_played"`

	// Track sum of ratings for averaging
	ratingSum       float64
	hltvRatingSum   float64
	pistolRatingSum float64
	mapRatingSum    map[string]float64
	mapGamesCount   map[string]int
}

// Aggregator combines stats from multiple games
type Aggregator struct {
	Players map[string]*AggregatedStats // keyed by SteamID
}

// NewAggregator creates a new stats aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		Players: make(map[string]*AggregatedStats),
	}
}

// AddGame adds stats from a single game to the aggregator
func (a *Aggregator) AddGame(players map[uint64]*model.PlayerStats, mapName string, tier string) {
	for _, p := range players {
		// Key by SteamID+Tier so each player has separate stats per tier
		key := p.SteamID + ":" + tier
		agg := a.ensurePlayer(key, p.SteamID, p.Name, tier)
		agg.GamesCount++

		// Add cumulative stats
		agg.RoundsPlayed += p.RoundsPlayed
		agg.RoundsWon += p.RoundsWon
		agg.RoundsLost += p.RoundsLost
		agg.Kills += p.Kills
		agg.Assists += p.Assists
		agg.Deaths += p.Deaths
		agg.Damage += p.Damage
		agg.OpeningKills += p.OpeningKills
		agg.PerfectKills += p.PerfectKills
		agg.TradeDenials += p.TradeDenials
		agg.TradedDeaths += p.TradedDeaths

		// Kill tracking
		agg.RoundsWithKill += p.RoundsWithKill
		agg.RoundsWithMultiKill += p.RoundsWithMultiKill
		agg.KillsInWonRounds += p.KillsInWonRounds
		agg.DamageInWonRounds += p.DamageInWonRounds

		// AWP/Sniper stats
		agg.AWPKills += p.AWPKills
		agg.RoundsWithAWPKill += p.RoundsWithAWPKill
		agg.AWPMultiKillRounds += p.AWPMultiKillRounds
		agg.AWPOpeningKills += p.AWPOpeningKills

		// Multi-kills
		agg.MultiKills.OneK += p.MultiKillsRaw[1]
		agg.MultiKills.TwoK += p.MultiKillsRaw[2]
		agg.MultiKills.ThreeK += p.MultiKillsRaw[3]
		agg.MultiKills.FourK += p.MultiKillsRaw[4]
		agg.MultiKills.FiveK += p.MultiKillsRaw[5]

		// Eco values
		agg.EcoKillValue += p.EcoKillValue
		agg.EcoDeathValue += p.EcoDeathValue

		// Round swing stats
		agg.RoundSwing += p.RoundSwing
		agg.ClutchRounds += p.ClutchRounds
		agg.ClutchWins += p.ClutchWins

		// Support stats
		agg.SavedByTeammate += p.SavedByTeammate
		agg.SavedTeammate += p.SavedTeammate
		agg.OpeningDeaths += p.OpeningDeaths
		agg.OpeningDeathsTraded += p.OpeningDeathsTraded
		agg.SupportRounds += p.SupportRounds
		agg.AssistedKills += p.AssistedKills

		// Entry/Opening stats
		agg.OpeningAttempts += p.OpeningAttempts
		agg.OpeningSuccesses += p.OpeningSuccesses
		agg.RoundsWonAfterOpening += p.RoundsWonAfterOpening
		agg.AttackRounds += p.AttackRounds

		// Clutch stats
		agg.Clutch1v1Attempts += p.Clutch1v1Attempts
		agg.Clutch1v1Wins += p.Clutch1v1Wins
		agg.totalTimeAlive += p.TotalTimeAlive
		agg.LastAliveRounds += p.LastAliveRounds
		agg.SavesOnLoss += p.SavesOnLoss

		// Utility stats
		agg.UtilityDamage += p.UtilityDamage
		agg.UtilityKills += p.UtilityKills
		agg.FlashesThrown += p.FlashesThrown
		agg.FlashAssists += p.FlashAssists
		agg.totalEnemyFlashDur += p.EnemyFlashDuration
		agg.TeamFlashCount += p.TeamFlashCount
		agg.totalTeamFlashDur += p.TeamFlashDuration

		// Misc stats
		agg.ExitFrags += p.ExitFrags
		agg.AWPDeaths += p.AWPDeaths
		agg.AWPDeathsNoKill += p.AWPDeathsNoKill
		agg.KnifeKills += p.KnifeKills
		agg.PistolVsRifleKills += p.PistolVsRifleKills
		agg.TradeKills += p.TradeKills
		agg.FastTrades += p.FastTrades
		agg.EarlyDeaths += p.EarlyDeaths
		agg.LowBuyKills += p.LowBuyKills
		agg.DisadvantagedBuyKills += p.DisadvantagedBuyKills

		// Pistol round stats
		agg.PistolRoundsPlayed += p.PistolRoundsPlayed
		agg.PistolRoundKills += p.PistolRoundKills
		agg.PistolRoundDeaths += p.PistolRoundDeaths
		agg.PistolRoundDamage += p.PistolRoundDamage
		agg.PistolRoundsWon += p.PistolRoundsWon
		agg.PistolRoundSurvivals += p.PistolRoundSurvivals
		agg.PistolRoundMultiKills += p.PistolRoundMultiKills

		// Per-side stats
		agg.TRoundsPlayed += p.TRoundsPlayed
		agg.TKills += p.TKills
		agg.TDeaths += p.TDeaths
		agg.TDamage += p.TDamage
		agg.TSurvivals += p.TSurvivals
		agg.TRoundsWithMultiKill += p.TRoundsWithMultiKill
		agg.TEcoKillValue += p.TEcoKillValue
		agg.TRoundSwing += p.TRoundSwing
		agg.TKAST += p.TKAST
		agg.TClutchRounds += p.TClutchRounds
		agg.TClutchWins += p.TClutchWins
		for i := 0; i < 6; i++ {
			agg.tMultiKills[i] += p.TMultiKills[i]
		}

		agg.CTRoundsPlayed += p.CTRoundsPlayed
		agg.CTKills += p.CTKills
		agg.CTDeaths += p.CTDeaths
		agg.CTDamage += p.CTDamage
		agg.CTSurvivals += p.CTSurvivals
		agg.CTRoundsWithMultiKill += p.CTRoundsWithMultiKill
		agg.CTEcoKillValue += p.CTEcoKillValue
		agg.CTRoundSwing += p.CTRoundSwing
		agg.CTKAST += p.CTKAST
		agg.CTClutchRounds += p.CTClutchRounds
		agg.CTClutchWins += p.CTClutchWins
		for i := 0; i < 6; i++ {
			agg.ctMultiKills[i] += p.CTMultiKills[i]
		}

		// Track rating sums for averaging
		agg.ratingSum += p.FinalRating
		agg.hltvRatingSum += p.HLTVRating
		agg.pistolRatingSum += p.PistolRoundRating

		// Track per-map rating
		if mapName != "" {
			agg.mapRatingSum[mapName] += p.FinalRating
			agg.mapGamesCount[mapName]++
		}

		// Track weighted sums for percentage-based stats (weighted by rounds played)
		// These are already percentages, so we need to weight them
		rounds := float64(p.RoundsPlayed)
		agg.RoundImpact += p.RoundImpact * rounds
		agg.Survival += p.Survival * rounds
		agg.KAST += p.KAST * rounds
		agg.EconImpact += p.EconImpact * rounds
	}
}

// Finalize calculates final averages and per-round stats
func (a *Aggregator) Finalize() {
	// HLTV Rating 1.0 average values
	avgKPR := 0.679
	avgSPR := 0.317 // Average (Survived - Deaths) / Rounds
	avgRMK := 1.277 // Average RMK points per round (weighted multi-kills)

	for _, agg := range a.Players {
		if agg.RoundsPlayed > 0 {
			rounds := float64(agg.RoundsPlayed)

			// Calculate per-round stats from totals
			agg.ADR = float64(agg.Damage) / rounds
			agg.KPR = float64(agg.Kills) / rounds
			agg.DPR = float64(agg.Deaths) / rounds
			agg.AWPKillsPerRound = float64(agg.AWPKills) / rounds

			// Calculate per-round averages for time-based stats
			agg.TimeAlivePerRound = agg.totalTimeAlive / rounds
			agg.EnemyFlashDurationPerRound = agg.totalEnemyFlashDur / rounds
			agg.TeamFlashDurationPerRound = agg.totalTeamFlashDur / rounds

			// Calculate weighted averages for percentage stats
			agg.RoundImpact = agg.RoundImpact / rounds
			agg.Survival = agg.Survival / rounds
			agg.KAST = agg.KAST / rounds
			agg.EconImpact = agg.EconImpact / rounds

			// Calculate HLTV Rating 1.0 from aggregated stats
			killRating := agg.KPR / avgKPR
			// SurvivalRating: (Survived - Deaths) / Rounds, normalized
			survived := agg.Survival * rounds // agg.Survival is already survival rate, convert back to count
			survivalRating := ((survived - float64(agg.Deaths)) / rounds) / avgSPR
			// RMK: weighted multi-kill points (1K=1, 2K=4, 3K=9, 4K=16, 5K=25)
			rmkPoints := float64(agg.MultiKills.OneK*1 + agg.MultiKills.TwoK*4 + agg.MultiKills.ThreeK*9 + agg.MultiKills.FourK*16 + agg.MultiKills.FiveK*25)
			rmkRating := (rmkPoints / rounds) / avgRMK
			agg.HLTVRating = (killRating + 0.7*survivalRating + rmkRating) / 2.7

			// Calculate all per-round and percentage stats
			agg.RoundsWithKillPct = float64(agg.RoundsWithKill) / rounds
			agg.RoundsWithMultiKillPct = float64(agg.RoundsWithMultiKill) / rounds
			agg.SavedByTeammatePerRound = float64(agg.SavedByTeammate) / rounds
			agg.TradedDeathsPerRound = float64(agg.TradedDeaths) / rounds
			agg.AssistsPerRound = float64(agg.Assists) / rounds
			agg.SupportRoundsPct = float64(agg.SupportRounds) / rounds
			agg.SavedTeammatePerRound = float64(agg.SavedTeammate) / rounds
			agg.TradeKillsPerRound = float64(agg.TradeKills) / rounds
			agg.OpeningKillsPerRound = float64(agg.OpeningKills) / rounds
			agg.OpeningDeathsPerRound = float64(agg.OpeningDeaths) / rounds
			agg.OpeningAttemptsPct = float64(agg.OpeningAttempts) / rounds
			agg.AttacksPerRound = float64(agg.AttackRounds) / rounds
			agg.ClutchPointsPerRound = float64(agg.ClutchWins) / rounds
			agg.LastAlivePct = float64(agg.LastAliveRounds) / rounds
			agg.RoundsWithAWPKillPct = float64(agg.RoundsWithAWPKill) / rounds
			agg.AWPMultiKillRoundsPerRound = float64(agg.AWPMultiKillRounds) / rounds
			agg.AWPOpeningKillsPerRound = float64(agg.AWPOpeningKills) / rounds
			agg.UtilityDamagePerRound = float64(agg.UtilityDamage) / rounds
			agg.UtilityKillsPer100Rounds = float64(agg.UtilityKills) * 100 / rounds
			agg.FlashesThrownPerRound = float64(agg.FlashesThrown) / rounds
			agg.FlashAssistsPerRound = float64(agg.FlashAssists) / rounds
		}

		// Stats that depend on RoundsWon
		if agg.RoundsWon > 0 {
			agg.KillsPerRoundWin = float64(agg.KillsInWonRounds) / float64(agg.RoundsWon)
			agg.DamagePerRoundWin = float64(agg.DamageInWonRounds) / float64(agg.RoundsWon)
		}

		// Stats that depend on RoundsLost
		if agg.RoundsLost > 0 {
			agg.SavesPerRoundLoss = float64(agg.SavesOnLoss) / float64(agg.RoundsLost)
		}

		// Stats that depend on Deaths
		if agg.Deaths > 0 {
			agg.TradedDeathsPct = float64(agg.TradedDeaths) / float64(agg.Deaths)
		}

		// Stats that depend on OpeningDeaths
		if agg.OpeningDeaths > 0 {
			agg.OpeningDeathsTradedPct = float64(agg.OpeningDeathsTraded) / float64(agg.OpeningDeaths)
		}

		// Stats that depend on Kills
		if agg.Kills > 0 {
			agg.TradeKillsPct = float64(agg.TradeKills) / float64(agg.Kills)
			agg.AssistedKillsPct = float64(agg.AssistedKills) / float64(agg.Kills)
			agg.DamagePerKill = float64(agg.Damage) / float64(agg.Kills)
			agg.AWPKillsPct = float64(agg.AWPKills) / float64(agg.Kills)
			agg.LowBuyKillsPct = float64(agg.LowBuyKills) / float64(agg.Kills)
			agg.DisadvantagedBuyKillsPct = float64(agg.DisadvantagedBuyKills) / float64(agg.Kills)
		}

		// Stats that depend on OpeningAttempts
		if agg.OpeningAttempts > 0 {
			agg.OpeningSuccessPct = float64(agg.OpeningSuccesses) / float64(agg.OpeningAttempts)
		}

		// Stats that depend on OpeningKills
		if agg.OpeningKills > 0 {
			agg.WinPctAfterOpeningKill = float64(agg.RoundsWonAfterOpening) / float64(agg.OpeningKills)
		}

		// Stats that depend on Clutch1v1Attempts
		if agg.Clutch1v1Attempts > 0 {
			agg.Clutch1v1WinPct = float64(agg.Clutch1v1Wins) / float64(agg.Clutch1v1Attempts)
		}

		// Calculate Pistol Round Rating from aggregated pistol stats
		if agg.PistolRoundsPlayed > 0 {
			pistolRounds := float64(agg.PistolRoundsPlayed)
			pistolKPR := float64(agg.PistolRoundKills) / pistolRounds
			// SurvivalRating: (Survived - Deaths) / Rounds
			pistolSurvivalRating := ((float64(agg.PistolRoundSurvivals) - float64(agg.PistolRoundDeaths)) / pistolRounds) / avgSPR
			// RMK: approximate using multi-kill rounds * 4 (average weight for 2K)
			pistolRMKPoints := float64(agg.PistolRoundMultiKills) * 4.0
			pistolRMKRating := (pistolRMKPoints / pistolRounds) / avgRMK

			pistolKillRating := pistolKPR / avgKPR
			agg.PistolRoundRating = (pistolKillRating + 0.7*pistolSurvivalRating + pistolRMKRating) / 2.7
		}

		// Calculate T-side HLTV Rating 1.0 and Eco Rating
		if agg.TRoundsPlayed > 0 {
			tRounds := float64(agg.TRoundsPlayed)
			tKPR := float64(agg.TKills) / tRounds
			// SurvivalRating: (Survived - Deaths) / Rounds
			tSurvivalRating := ((float64(agg.TSurvivals) - float64(agg.TDeaths)) / tRounds) / avgSPR
			// RMK: weighted multi-kill points
			tRMKPoints := float64(agg.tMultiKills[1]*1 + agg.tMultiKills[2]*4 + agg.tMultiKills[3]*9 + agg.tMultiKills[4]*16 + agg.tMultiKills[5]*25)
			tRMKRating := (tRMKPoints / tRounds) / avgRMK

			tKillRating := tKPR / avgKPR
			agg.TRating = (tKillRating + 0.7*tSurvivalRating + tRMKRating) / 2.7

			// Calculate T-side Eco Rating
			agg.TEcoRating = rating.ComputeSideRating(
				agg.TRoundsPlayed, agg.TKills, agg.TDeaths, agg.TDamage, agg.TEcoKillValue,
				agg.TRoundSwing, agg.TKAST, agg.tMultiKills, agg.TClutchRounds, agg.TClutchWins)
		}

		// Calculate CT-side HLTV Rating 1.0 and Eco Rating
		if agg.CTRoundsPlayed > 0 {
			ctRounds := float64(agg.CTRoundsPlayed)
			ctKPR := float64(agg.CTKills) / ctRounds
			// SurvivalRating: (Survived - Deaths) / Rounds
			ctSurvivalRating := ((float64(agg.CTSurvivals) - float64(agg.CTDeaths)) / ctRounds) / avgSPR
			// RMK: weighted multi-kill points
			ctRMKPoints := float64(agg.ctMultiKills[1]*1 + agg.ctMultiKills[2]*4 + agg.ctMultiKills[3]*9 + agg.ctMultiKills[4]*16 + agg.ctMultiKills[5]*25)
			ctRMKRating := (ctRMKPoints / ctRounds) / avgRMK

			ctKillRating := ctKPR / avgKPR
			agg.CTRating = (ctKillRating + 0.7*ctSurvivalRating + ctRMKRating) / 2.7

			// Calculate CT-side Eco Rating
			agg.CTEcoRating = rating.ComputeSideRating(
				agg.CTRoundsPlayed, agg.CTKills, agg.CTDeaths, agg.CTDamage, agg.CTEcoKillValue,
				agg.CTRoundSwing, agg.CTKAST, agg.ctMultiKills, agg.CTClutchRounds, agg.CTClutchWins)
		}

		// Average rating across games
		if agg.GamesCount > 0 {
			agg.FinalRating = agg.ratingSum / float64(agg.GamesCount)
		}

		// Calculate per-map average ratings and copy games count
		for mapName, ratingSum := range agg.mapRatingSum {
			if count := agg.mapGamesCount[mapName]; count > 0 {
				agg.MapRatings[mapName] = ratingSum / float64(count)
				agg.MapGamesPlayed[mapName] = count
			}
		}
	}
}

// GetResults returns the aggregated stats map
func (a *Aggregator) GetResults() map[string]*AggregatedStats {
	return a.Players
}

func (a *Aggregator) ensurePlayer(key, steamID, name, tier string) *AggregatedStats {
	if _, ok := a.Players[key]; !ok {
		a.Players[key] = &AggregatedStats{
			SteamID:        steamID,
			Name:           name,
			Tier:           tier,
			MapRatings:     make(map[string]float64),
			MapGamesPlayed: make(map[string]int),
			mapRatingSum:   make(map[string]float64),
			mapGamesCount:  make(map[string]int),
		}
	}
	return a.Players[key]
}
