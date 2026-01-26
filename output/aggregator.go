// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package output provides functionality for aggregating player statistics across
// multiple games and exporting results. The Aggregator accumulates raw stats and
// computes derived metrics like ratings, percentages, and per-round averages.
package output

import (
	"eco-rating/model"
	"eco-rating/rating"
)

// MultiKillStats tracks multi-kill round counts for aggregated statistics.
type MultiKillStats struct {
	OneK   int `json:"1k"`
	TwoK   int `json:"2k"`
	ThreeK int `json:"3k"`
	FourK  int `json:"4k"`
	FiveK  int `json:"5k"`
}

// AggregatedStats contains cumulative statistics for a player across multiple games.
// Raw counts are accumulated during AddGame, and derived metrics (rates, percentages)
// are calculated during Finalize. The struct also tracks per-map performance.
type AggregatedStats struct {
	SteamID         string  `json:"steam_id"`
	Name            string  `json:"name"`
	Tier            string  `json:"tier"`
	GamesCount      int     `json:"games_count"`
	RoundsPlayed    int     `json:"rounds_played"`
	RoundsWon       int     `json:"rounds_won"`
	RoundsLost      int     `json:"rounds_lost"`
	Kills           int     `json:"kills"`
	Assists         int     `json:"assists"`
	Deaths          int     `json:"deaths"`
	Damage          int     `json:"damage"`
	OpeningKills    int     `json:"opening_kills"`
	ADR             float64 `json:"adr"`
	KPR             float64 `json:"kpr"`
	DPR             float64 `json:"dpr"`
	Headshots       int     `json:"headshots"`
	HeadshotPct     float64 `json:"headshot_pct"`
	TotalTimeToKill float64 `json:"-"`
	KillsWithTTK    int     `json:"-"`
	AvgTimeToKill   float64 `json:"avg_time_to_kill"`

	PerfectKills        int     `json:"perfect_kills"`
	TradeDenials        int     `json:"trade_denials"`
	TradedDeaths        int     `json:"traded_deaths"`
	RoundsWithKill      int     `json:"rounds_with_kill"`
	RoundsWithMultiKill int     `json:"rounds_with_multi_kill"`
	KillsInWonRounds    int     `json:"kills_in_won_rounds"`
	DamageInWonRounds   int     `json:"damage_in_won_rounds"`
	AWPKills            int     `json:"awp_kills"`
	AWPKillsPerRound    float64 `json:"awp_kills_per_round"`
	RoundsWithAWPKill   int     `json:"rounds_with_awp_kill"`
	AWPMultiKillRounds  int     `json:"awp_multi_kill_rounds"`
	AWPOpeningKills     int     `json:"awp_opening_kills"`

	MultiKills                 MultiKillStats `json:"multi_kills"`
	RoundImpact                float64        `json:"round_impact"`
	Survival                   float64        `json:"survival"`
	KAST                       float64        `json:"kast"`
	EconImpact                 float64        `json:"econ_impact"`
	EcoKillValue               float64        `json:"eco_kill_value"`
	EcoDeathValue              float64        `json:"eco_death_value"`
	ProbabilitySwing           float64        `json:"probability_swing"`
	ProbabilitySwingPerRound   float64        `json:"probability_swing_per_round"`
	ClutchRounds               int            `json:"clutch_rounds"`
	ClutchWins                 int            `json:"clutch_wins"`
	SavedByTeammate            int            `json:"saved_by_teammate"`
	SavedTeammate              int            `json:"saved_teammate"`
	OpeningDeaths              int            `json:"opening_deaths"`
	OpeningDeathsTraded        int            `json:"opening_deaths_traded"`
	SupportRounds              int            `json:"support_rounds"`
	AssistedKills              int            `json:"assisted_kills"`
	OpeningAttempts            int            `json:"opening_attempts"`
	OpeningSuccesses           int            `json:"opening_successes"`
	RoundsWonAfterOpening      int            `json:"rounds_won_after_opening"`
	AttackRounds               int            `json:"attack_rounds"`
	Clutch1v1Attempts          int            `json:"clutch_1v1_attempts"`
	Clutch1v1Wins              int            `json:"clutch_1v1_wins"`
	TimeAlivePerRound          float64        `json:"time_alive_per_round"`
	LastAliveRounds            int            `json:"last_alive_rounds"`
	SavesOnLoss                int            `json:"saves_on_loss"`
	UtilityDamage              int            `json:"utility_damage"`
	UtilityKills               int            `json:"utility_kills"`
	FlashesThrown              int            `json:"flashes_thrown"`
	FlashAssists               int            `json:"flash_assists"`
	EnemyFlashDurationPerRound float64        `json:"enemy_flash_duration_per_round"`
	TeamFlashCount             int            `json:"team_flash_count"`
	TeamFlashDurationPerRound  float64        `json:"team_flash_duration_per_round"`
	totalTimeAlive             float64
	totalEnemyFlashDur         float64
	totalTeamFlashDur          float64
	ExitFrags                  int     `json:"exit_frags"`
	AWPDeaths                  int     `json:"awp_deaths"`
	AWPDeathsNoKill            int     `json:"awp_deaths_no_kill"`
	KnifeKills                 int     `json:"knife_kills"`
	PistolVsRifleKills         int     `json:"pistol_vs_rifle_kills"`
	TradeKills                 int     `json:"trade_kills"`
	FastTrades                 int     `json:"fast_trades"`
	EarlyDeaths                int     `json:"early_deaths"`
	LowBuyKills                int     `json:"low_buy_kills"`
	LowBuyKillsPct             float64 `json:"low_buy_kills_pct"`
	DisadvantagedBuyKills      int     `json:"disadvantaged_buy_kills"`
	DisadvantagedBuyKillsPct   float64 `json:"disadvantaged_buy_kills_pct"`
	PistolRoundsPlayed         int     `json:"pistol_rounds_played"`
	PistolRoundKills           int     `json:"pistol_round_kills"`
	PistolRoundDeaths          int     `json:"pistol_round_deaths"`
	PistolRoundDamage          int     `json:"pistol_round_damage"`
	PistolRoundsWon            int     `json:"pistol_rounds_won"`
	PistolRoundSurvivals       int     `json:"pistol_round_survivals"`
	PistolRoundMultiKills      int     `json:"pistol_round_multi_kills"`
	PistolRoundRating          float64 `json:"pistol_round_rating"`
	TRoundsPlayed              int     `json:"t_rounds_played"`
	TKills                     int     `json:"t_kills"`
	TDeaths                    int     `json:"t_deaths"`
	TDamage                    int     `json:"t_damage"`
	TSurvivals                 int     `json:"t_survivals"`
	TRoundsWithMultiKill       int     `json:"t_rounds_with_multi_kill"`
	TEcoKillValue              float64 `json:"t_eco_kill_value"`
	TProbabilitySwing          float64 `json:"t_probability_swing"`
	TKAST                      float64 `json:"t_kast"`
	TClutchRounds              int     `json:"t_clutch_rounds"`
	TClutchWins                int     `json:"t_clutch_wins"`
	TRating                    float64 `json:"t_rating"`
	TEcoRating                 float64 `json:"t_eco_rating"`

	CTRoundsPlayed             int     `json:"ct_rounds_played"`
	CTKills                    int     `json:"ct_kills"`
	CTDeaths                   int     `json:"ct_deaths"`
	CTDamage                   int     `json:"ct_damage"`
	CTSurvivals                int     `json:"ct_survivals"`
	CTRoundsWithMultiKill      int     `json:"ct_rounds_with_multi_kill"`
	CTEcoKillValue             float64 `json:"ct_eco_kill_value"`
	CTProbabilitySwing         float64 `json:"ct_probability_swing"`
	CTKAST                     float64 `json:"ct_kast"`
	CTClutchRounds             int     `json:"ct_clutch_rounds"`
	CTClutchWins               int     `json:"ct_clutch_wins"`
	CTRating                   float64 `json:"ct_rating"`
	CTEcoRating                float64 `json:"ct_eco_rating"`
	tMultiKills                [6]int
	ctMultiKills               [6]int
	HLTVRating                 float64            `json:"hltv_rating"`
	FinalRating                float64            `json:"final_rating"`
	RoundsWithKillPct          float64            `json:"rounds_with_kill_pct"`
	KillsPerRoundWin           float64            `json:"kills_per_round_win"`
	RoundsWithMultiKillPct     float64            `json:"rounds_with_multi_kill_pct"`
	DamagePerRoundWin          float64            `json:"damage_per_round_win"`
	SavedByTeammatePerRound    float64            `json:"saved_by_teammate_per_round"`
	TradedDeathsPerRound       float64            `json:"traded_deaths_per_round"`
	TradedDeathsPct            float64            `json:"traded_deaths_pct"`
	OpeningDeathsTradedPct     float64            `json:"opening_deaths_traded_pct"`
	AssistsPerRound            float64            `json:"assists_per_round"`
	SupportRoundsPct           float64            `json:"support_rounds_pct"`
	SavedTeammatePerRound      float64            `json:"saved_teammate_per_round"`
	TradeKillsPerRound         float64            `json:"trade_kills_per_round"`
	TradeKillsPct              float64            `json:"trade_kills_pct"`
	AssistedKillsPct           float64            `json:"assisted_kills_pct"`
	DamagePerKill              float64            `json:"damage_per_kill"`
	OpeningKillsPerRound       float64            `json:"opening_kills_per_round"`
	OpeningDeathsPerRound      float64            `json:"opening_deaths_per_round"`
	OpeningAttemptsPct         float64            `json:"opening_attempts_pct"`
	OpeningSuccessPct          float64            `json:"opening_success_pct"`
	WinPctAfterOpeningKill     float64            `json:"win_pct_after_opening_kill"`
	AttacksPerRound            float64            `json:"attacks_per_round"`
	ClutchPointsPerRound       float64            `json:"clutch_points_per_round"`
	LastAlivePct               float64            `json:"last_alive_pct"`
	Clutch1v1WinPct            float64            `json:"clutch_1v1_win_pct"`
	SavesPerRoundLoss          float64            `json:"saves_per_round_loss"`
	AWPKillsPct                float64            `json:"awp_kills_pct"`
	RoundsWithAWPKillPct       float64            `json:"rounds_with_awp_kill_pct"`
	AWPMultiKillRoundsPerRound float64            `json:"awp_multi_kill_rounds_per_round"`
	AWPOpeningKillsPerRound    float64            `json:"awp_opening_kills_per_round"`
	UtilityDamagePerRound      float64            `json:"utility_damage_per_round"`
	UtilityKillsPer100Rounds   float64            `json:"utility_kills_per_100_rounds"`
	FlashesThrownPerRound      float64            `json:"flashes_thrown_per_round"`
	FlashAssistsPerRound       float64            `json:"flash_assists_per_round"`
	MapRatings                 map[string]float64 `json:"map_ratings"`
	MapGamesPlayed             map[string]int     `json:"map_games_played"`
	ratingSum                  float64
	hltvRatingSum              float64
	pistolRatingSum            float64
	mapRatingSum               map[string]float64
	mapGamesCount              map[string]int
}

// Aggregator collects and combines player statistics from multiple games.
// Players are keyed by "SteamID:Tier" to allow separate tracking per tier.
type Aggregator struct {
	Players map[string]*AggregatedStats // Map of player key to aggregated stats
}

// NewAggregator creates a new Aggregator with an empty player map.
func NewAggregator() *Aggregator {
	return &Aggregator{
		Players: make(map[string]*AggregatedStats),
	}
}

// AddGame incorporates statistics from a single game into the aggregator.
// It accumulates raw counts and weighted values for later finalization.
// The mapName is used for per-map rating tracking.
func (a *Aggregator) AddGame(players map[uint64]*model.PlayerStats, mapName string, tier string) {
	for _, p := range players {
		key := p.SteamID + ":" + tier
		agg := a.ensurePlayer(key, p.SteamID, p.Name, tier)
		agg.GamesCount++
		agg.RoundsPlayed += p.RoundsPlayed
		agg.RoundsWon += p.RoundsWon
		agg.RoundsLost += p.RoundsLost
		agg.Kills += p.Kills
		agg.Assists += p.Assists
		agg.Deaths += p.Deaths
		agg.Damage += p.Damage
		agg.OpeningKills += p.OpeningKills
		agg.Headshots += p.Headshots
		agg.TotalTimeToKill += p.TotalTimeToKill
		agg.KillsWithTTK += p.KillsWithTTK
		agg.PerfectKills += p.PerfectKills
		agg.TradeDenials += p.TradeDenials
		agg.TradedDeaths += p.TradedDeaths
		agg.RoundsWithKill += p.RoundsWithKill
		agg.RoundsWithMultiKill += p.RoundsWithMultiKill
		agg.KillsInWonRounds += p.KillsInWonRounds
		agg.DamageInWonRounds += p.DamageInWonRounds
		agg.AWPKills += p.AWPKills
		agg.RoundsWithAWPKill += p.RoundsWithAWPKill
		agg.AWPMultiKillRounds += p.AWPMultiKillRounds
		agg.AWPOpeningKills += p.AWPOpeningKills
		agg.MultiKills.OneK += p.MultiKillsRaw[1]
		agg.MultiKills.TwoK += p.MultiKillsRaw[2]
		agg.MultiKills.ThreeK += p.MultiKillsRaw[3]
		agg.MultiKills.FourK += p.MultiKillsRaw[4]
		agg.MultiKills.FiveK += p.MultiKillsRaw[5]
		agg.EcoKillValue += p.EcoKillValue
		agg.EcoDeathValue += p.EcoDeathValue
		agg.ProbabilitySwing += p.ProbabilitySwing
		agg.ClutchRounds += p.ClutchRounds
		agg.ClutchWins += p.ClutchWins
		agg.SavedByTeammate += p.SavedByTeammate
		agg.SavedTeammate += p.SavedTeammate
		agg.OpeningDeaths += p.OpeningDeaths
		agg.OpeningDeathsTraded += p.OpeningDeathsTraded
		agg.SupportRounds += p.SupportRounds
		agg.AssistedKills += p.AssistedKills
		agg.OpeningAttempts += p.OpeningAttempts
		agg.OpeningSuccesses += p.OpeningSuccesses
		agg.RoundsWonAfterOpening += p.RoundsWonAfterOpening
		agg.AttackRounds += p.AttackRounds
		agg.Clutch1v1Attempts += p.Clutch1v1Attempts
		agg.Clutch1v1Wins += p.Clutch1v1Wins
		agg.totalTimeAlive += p.TotalTimeAlive
		agg.LastAliveRounds += p.LastAliveRounds
		agg.SavesOnLoss += p.SavesOnLoss
		agg.UtilityDamage += p.UtilityDamage
		agg.UtilityKills += p.UtilityKills
		agg.FlashesThrown += p.FlashesThrown
		agg.FlashAssists += p.FlashAssists
		agg.totalEnemyFlashDur += p.EnemyFlashDuration
		agg.TeamFlashCount += p.TeamFlashCount
		agg.totalTeamFlashDur += p.TeamFlashDuration
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
		agg.PistolRoundsPlayed += p.PistolRoundsPlayed
		agg.PistolRoundKills += p.PistolRoundKills
		agg.PistolRoundDeaths += p.PistolRoundDeaths
		agg.PistolRoundDamage += p.PistolRoundDamage
		agg.PistolRoundsWon += p.PistolRoundsWon
		agg.PistolRoundSurvivals += p.PistolRoundSurvivals
		agg.PistolRoundMultiKills += p.PistolRoundMultiKills
		agg.TRoundsPlayed += p.TRoundsPlayed
		agg.TKills += p.TKills
		agg.TDeaths += p.TDeaths
		agg.TDamage += p.TDamage
		agg.TSurvivals += p.TSurvivals
		agg.TRoundsWithMultiKill += p.TRoundsWithMultiKill
		agg.TEcoKillValue += p.TEcoKillValue
		agg.TProbabilitySwing += p.TProbabilitySwing
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
		agg.CTProbabilitySwing += p.CTProbabilitySwing
		agg.CTKAST += p.CTKAST
		agg.CTClutchRounds += p.CTClutchRounds
		agg.CTClutchWins += p.CTClutchWins
		for i := 0; i < 6; i++ {
			agg.ctMultiKills[i] += p.CTMultiKills[i]
		}
		agg.ratingSum += p.FinalRating
		agg.hltvRatingSum += p.HLTVRating
		agg.pistolRatingSum += p.PistolRoundRating
		if mapName != "" {
			agg.mapRatingSum[mapName] += p.FinalRating
			agg.mapGamesCount[mapName]++
		}
		rounds := float64(p.RoundsPlayed)
		agg.RoundImpact += p.RoundImpact * rounds
		agg.Survival += p.Survival * rounds
		agg.KAST += p.KAST * rounds
		agg.EconImpact += p.EconImpact * rounds
	}
}

// Finalize computes all derived statistics from accumulated raw values.
// This includes per-round rates, percentages, HLTV ratings, and side-specific ratings.
// Must be called after all games have been added and before exporting results.
func (a *Aggregator) Finalize() {
	for _, agg := range a.Players {
		if agg.RoundsPlayed > 0 {
			rounds := float64(agg.RoundsPlayed)
			agg.ADR = float64(agg.Damage) / rounds
			agg.KPR = float64(agg.Kills) / rounds
			agg.DPR = float64(agg.Deaths) / rounds
			agg.AWPKillsPerRound = float64(agg.AWPKills) / rounds
			agg.TimeAlivePerRound = agg.totalTimeAlive / rounds
			agg.EnemyFlashDurationPerRound = agg.totalEnemyFlashDur / rounds
			agg.TeamFlashDurationPerRound = agg.totalTeamFlashDur / rounds
			agg.RoundImpact = agg.RoundImpact / rounds
			agg.Survival = agg.Survival / rounds
			agg.KAST = agg.KAST / rounds
			agg.EconImpact = agg.EconImpact / rounds
			agg.ProbabilitySwingPerRound = agg.ProbabilitySwing / rounds

			// Calculate HLTV rating using centralized function
			survivals := int(agg.Survival * rounds)
			multiKillsArr := [6]int{0, agg.MultiKills.OneK, agg.MultiKills.TwoK, agg.MultiKills.ThreeK, agg.MultiKills.FourK, agg.MultiKills.FiveK}
			agg.HLTVRating = rating.ComputeHLTVRating(rating.HLTVInput{
				RoundsPlayed: agg.RoundsPlayed,
				Kills:        agg.Kills,
				Deaths:       agg.Deaths,
				Survivals:    survivals,
				MultiKills:   multiKillsArr,
			})
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
		if agg.RoundsWon > 0 {
			agg.KillsPerRoundWin = float64(agg.KillsInWonRounds) / float64(agg.RoundsWon)
			agg.DamagePerRoundWin = float64(agg.DamageInWonRounds) / float64(agg.RoundsWon)
		}
		if agg.RoundsLost > 0 {
			agg.SavesPerRoundLoss = float64(agg.SavesOnLoss) / float64(agg.RoundsLost)
		}
		if agg.Deaths > 0 {
			agg.TradedDeathsPct = float64(agg.TradedDeaths) / float64(agg.Deaths)
		}
		if agg.OpeningDeaths > 0 {
			agg.OpeningDeathsTradedPct = float64(agg.OpeningDeathsTraded) / float64(agg.OpeningDeaths)
		}
		if agg.Kills > 0 {
			agg.TradeKillsPct = float64(agg.TradeKills) / float64(agg.Kills)
			agg.AssistedKillsPct = float64(agg.AssistedKills) / float64(agg.Kills)
			agg.DamagePerKill = float64(agg.Damage) / float64(agg.Kills)
			agg.AWPKillsPct = float64(agg.AWPKills) / float64(agg.Kills)
			agg.LowBuyKillsPct = float64(agg.LowBuyKills) / float64(agg.Kills)
			agg.DisadvantagedBuyKillsPct = float64(agg.DisadvantagedBuyKills) / float64(agg.Kills)
			agg.HeadshotPct = float64(agg.Headshots) / float64(agg.Kills)
		}
		if agg.KillsWithTTK > 0 {
			agg.AvgTimeToKill = agg.TotalTimeToKill / float64(agg.KillsWithTTK)
		}
		if agg.OpeningAttempts > 0 {
			agg.OpeningSuccessPct = float64(agg.OpeningSuccesses) / float64(agg.OpeningAttempts)
		}
		if agg.OpeningKills > 0 {
			agg.WinPctAfterOpeningKill = float64(agg.RoundsWonAfterOpening) / float64(agg.OpeningKills)
		}
		if agg.Clutch1v1Attempts > 0 {
			agg.Clutch1v1WinPct = float64(agg.Clutch1v1Wins) / float64(agg.Clutch1v1Attempts)
		}
		// Pistol round rating using centralized function
		if agg.PistolRoundsPlayed > 0 {
			agg.PistolRoundRating = rating.ComputePistolRoundRating(
				agg.PistolRoundsPlayed, agg.PistolRoundKills, agg.PistolRoundDeaths,
				agg.PistolRoundSurvivals, agg.PistolRoundMultiKills)
		}

		// T-side ratings using centralized functions
		if agg.TRoundsPlayed > 0 {
			agg.TRating = rating.ComputeSideHLTVRating(
				agg.TRoundsPlayed, agg.TKills, agg.TDeaths, agg.TSurvivals, agg.tMultiKills)
			agg.TEcoRating = rating.ComputeSideRating(
				agg.TRoundsPlayed, agg.TKills, agg.TDeaths, agg.TDamage, agg.TEcoKillValue,
				agg.TProbabilitySwing, agg.TKAST, agg.tMultiKills, agg.TClutchRounds, agg.TClutchWins)
		}

		// CT-side ratings using centralized functions
		if agg.CTRoundsPlayed > 0 {
			agg.CTRating = rating.ComputeSideHLTVRating(
				agg.CTRoundsPlayed, agg.CTKills, agg.CTDeaths, agg.CTSurvivals, agg.ctMultiKills)
			agg.CTEcoRating = rating.ComputeSideRating(
				agg.CTRoundsPlayed, agg.CTKills, agg.CTDeaths, agg.CTDamage, agg.CTEcoKillValue,
				agg.CTProbabilitySwing, agg.CTKAST, agg.ctMultiKills, agg.CTClutchRounds, agg.CTClutchWins)
		}
		if agg.GamesCount > 0 {
			agg.FinalRating = agg.ratingSum / float64(agg.GamesCount)
		}
		for mapName, ratingSum := range agg.mapRatingSum {
			if count := agg.mapGamesCount[mapName]; count > 0 {
				agg.MapRatings[mapName] = ratingSum / float64(count)
				agg.MapGamesPlayed[mapName] = count
			}
		}
	}
}

// GetResults returns the map of all aggregated player statistics.
// Should be called after Finalize() to get computed metrics.
func (a *Aggregator) GetResults() map[string]*AggregatedStats {
	return a.Players
}

// ensurePlayer returns the AggregatedStats for a player, creating it if needed.
// The key format is "SteamID:Tier" to track players separately per tier.
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
