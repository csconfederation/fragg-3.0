// Package export provides functionality for exporting player statistics.
// This file handles conversion from ecorating's PlayerStats to demoScrape2-compatible format.
package export

import (
	"strconv"

	"github.com/ethsmith/eco-rating/model"
)

// ConvertToCSCGame converts ecorating's parsed data to a demoScrape2-compatible Game struct.
// This allows ecorating to be a drop-in replacement for csgo-demo-worker.
func ConvertToCSCGame(
	players map[uint64]*model.PlayerStats,
	mapName string,
	totalRounds int,
	tickRate int,
) *CSCGame {
	game := &CSCGame{
		CoreID:           "",
		MapNum:           1,
		WinnerClanName:   determineWinner(players),
		Result:           "Ended",
		MapName:          mapName,
		TickRate:         tickRate,
		TotalRounds:      totalRounds,
		TotalPlayerStats: make(map[uint64]*CSCPlayerStats),
		CtPlayerStats:    make(map[uint64]*CSCPlayerStats),
		TPlayerStats:     make(map[uint64]*CSCPlayerStats),
		TotalTeamStats:   make(map[string]*CSCTeamStats),
		Teams:            make(map[string]*CSCTeam),
		PlayerOrder:      make([]uint64, 0, len(players)),
		TeamOrder:        make([]string, 0),
	}

	// Track teams
	teamScores := make(map[string]int)
	teamStats := make(map[string]*CSCTeamStats)

	for steamID, p := range players {
		cscPlayer := convertPlayerStats(p)
		game.TotalPlayerStats[steamID] = cscPlayer
		game.PlayerOrder = append(game.PlayerOrder, steamID)

		// Track team
		if p.TeamName != "" {
			if _, exists := teamScores[p.TeamName]; !exists {
				teamScores[p.TeamName] = p.RoundsWon
				game.TeamOrder = append(game.TeamOrder, p.TeamName)
				teamStats[p.TeamName] = &CSCTeamStats{}
			}
			// Aggregate team stats
			ts := teamStats[p.TeamName]
			ts.Deaths += p.Deaths
			ts.Saves += p.SavesOnLoss
			ts.Clutches += p.ClutchWins
			ts.Traded += p.TradedDeaths
			ts.Fass += p.FlashAssists
			ts.Ef += p.EnemiesFlashed
			ts.Ud += p.UtilityDamage
		}
	}

	// Set team data
	for teamName, score := range teamScores {
		game.Teams[teamName] = &CSCTeam{
			Name:          teamName,
			Score:         score,
			ScoreAdjusted: score,
		}
		game.TotalTeamStats[teamName] = teamStats[teamName]
	}

	return game
}

// convertPlayerStats converts a single ecorating PlayerStats to CSCPlayerStats.
func convertPlayerStats(p *model.PlayerStats) *CSCPlayerStats {
	steamID64, _ := strconv.ParseUint(p.SteamID, 10, 64)

	// Calculate ATD in ticks (assuming 64 tick, ~15.6ms per tick)
	// ATD in seconds * 64 = ATD in ticks
	atdTicks := int(p.AvgTimeToDeath * 64)

	return &CSCPlayerStats{
		// Core identification
		Name:         p.Name,
		SteamID:      p.SteamID,
		IsBot:        steamID64 == 0,
		TeamENUM:     0, // Not tracked by ecorating
		TeamClanName: p.TeamName,
		Side:         0, // Not tracked per-player
		Rounds:       p.RoundsPlayed,

		// Combat stats
		Damage:         p.Damage,
		Kills:          uint8(p.Kills),
		Assists:        uint8(p.Assists),
		Deaths:         uint8(p.Deaths),
		DeathTick:      0,                          // Per-round stat
		DeathPlacement: 0,                          // Not tracked
		TicksAlive:     int(p.TotalTimeAlive * 64), // Convert seconds to ticks

		// Trade stats
		Trades: p.TradeKills,
		Traded: p.TradedDeaths,

		// Opening duels
		Ok: p.OpeningKills,
		Ol: p.OpeningDeaths,

		// Clutch wins
		Cl_1: p.Clutch1v1Wins,
		Cl_2: p.Clutch1v2Wins,
		Cl_3: p.Clutch1v3Wins,
		Cl_4: p.Clutch1v4Wins,
		Cl_5: p.Clutch1v5Wins,

		// Multi-kills
		TwoK:   p.MultiKillsRaw[2],
		ThreeK: p.MultiKillsRaw[3],
		FourK:  p.MultiKillsRaw[4],
		FiveK:  p.MultiKillsRaw[5],

		// Utility stats
		NadeDmg:        p.HEDamage,
		InfernoDmg:     p.FireDamage,
		UtilDmg:        p.UtilityDamage,
		Ef:             p.EnemiesFlashed,
		FAss:           p.FlashAssists,
		EnemyFlashTime: p.EnemyFlashDuration,

		// Other combat stats
		Hs:         p.Headshots,
		KastRounds: p.KAST * float64(p.RoundsPlayed), // Convert percentage to rounds
		Saves:      p.SavesOnLoss,
		Entries:    p.OpeningKills, // Entries = opening kills

		// Rating components (map to probability swing)
		KillPoints:   p.EcoKillValue,
		ImpactPoints: p.ProbabilitySwing,
		WinPoints:    p.ProbabilitySwing,

		// Weapon stats
		AwpKills: p.AWPKills,
		RF:       p.Kills - p.AWPKills, // Rifle frags = non-AWP kills
		RA:       0,                    // Not tracked

		// Utility thrown
		NadesThrown: p.TotalNadesThrown,
		FiresThrown: p.MolotovsThrown,
		FlashThrown: p.FlashesThrown,
		SmokeThrown: p.SmokesThrown,

		// Damage taken
		DamageTaken: p.DamageTaken,

		// Support stats
		SuppRounds: p.SupportRounds,
		SuppDamage: 0, // Not tracked separately

		// Positioning (not tracked)
		LurkerBlips:         0,
		DistanceToTeammates: 0,
		LurkRounds:          0,

		// Advanced metrics
		Wlp: p.ProbabilitySwing,
		Mip: p.ProbabilitySwing,
		Rws: p.RoundWinShares,
		Eac: p.AssistedKills,
		Rwk: p.RoundsWithKill,

		// Derived stats
		UtilThrown:   p.TotalNadesThrown,
		Atd:          atdTicks,
		Kast:         p.KAST,
		KillPointAvg: safeDiv64(p.EcoKillValue, float64(p.Kills)),
		Iiwr:         0, // Not tracked
		Adr:          p.ADR,
		DrDiff:       p.ADR - safeDiv64(float64(p.DamageTaken), float64(p.RoundsPlayed)),
		KR:           p.KPR,
		Tr:           safeDiv64(float64(p.TradeKills), float64(p.TradedDeaths)),
		ImpactRating: p.RoundImpact,
		Rating:       p.FinalRating,

		// Side-specific stats
		TDamage:        p.TDamage,
		CtDamage:       p.CTDamage,
		TImpactPoints:  p.TProbabilitySwing,
		TWinPoints:     p.TProbabilitySwing,
		TOK:            p.TOpeningKills,
		TOL:            p.TOpeningDeaths,
		CtImpactPoints: p.CTProbabilitySwing,
		CtWinPoints:    p.CTProbabilitySwing,
		CtOK:           p.CTOpeningKills,
		CtOL:           p.CTOpeningDeaths,
		TKills:         uint8(p.TKills),
		TDeaths:        uint8(p.TDeaths),
		TKAST:          p.TKAST,
		TKASTRounds:    p.TKAST * float64(p.TRoundsPlayed),
		TADR:           safeDiv64(float64(p.TDamage), float64(p.TRoundsPlayed)),
		CtKills:        uint8(p.CTKills),
		CtDeaths:       uint8(p.CTDeaths),
		CtKAST:         p.CTKAST,
		CtKASTRounds:   p.CTKAST * float64(p.CTRoundsPlayed),
		CtADR:          safeDiv64(float64(p.CTDamage), float64(p.CTRoundsPlayed)),
		TRounds:        p.TRoundsPlayed,
		CtRounds:       p.CTRoundsPlayed,
		CtRating:       p.CTRating,
		CtImpactRating: p.CTEcoRating,
		TRating:        p.TRating,
		TImpactRating:  p.TEcoRating,

		// Ecorating-specific stats
		EcoClutch1v1Attempts: p.Clutch1v1Attempts,
		EcoClutch1v2Attempts: p.Clutch1v2Attempts,
		EcoClutch1v3Attempts: p.Clutch1v3Attempts,
		EcoClutch1v4Attempts: p.Clutch1v4Attempts,
		EcoClutch1v5Attempts: p.Clutch1v5Attempts,

		EcoProbabilitySwing:         p.ProbabilitySwing,
		EcoProbabilitySwingPerRound: p.ProbabilitySwingPerRound,
		EcoTProbabilitySwing:        p.TProbabilitySwing,
		EcoCTProbabilitySwing:       p.CTProbabilitySwing,

		EcoKillValue:  p.EcoKillValue,
		EcoDeathValue: p.EcoDeathValue,
		EcoDuelSwing:  p.DuelSwing,

		EcoFinalRating: p.FinalRating,
		EcoHLTVRating:  p.HLTVRating,
		EcoTEcoRating:  p.TEcoRating,
		EcoCTEcoRating: p.CTEcoRating,

		EcoTradeDenials:        p.TradeDenials,
		EcoTradeKills:          p.TradeKills,
		EcoFastTrades:          p.FastTrades,
		EcoOpeningDeathsTraded: p.OpeningDeathsTraded,

		EcoAWPOpeningKills:    p.AWPOpeningKills,
		EcoAWPMultiKillRounds: p.AWPMultiKillRounds,
		EcoAWPDeaths:          p.AWPDeaths,

		EcoTimeAlivePerRound: p.TimeAlivePerRound,
		EcoAvgTimeToDeath:    p.AvgTimeToDeath,
		EcoAvgTimeToKill:     p.AvgTimeToKill,
	}
}

// determineWinner finds the team with the most rounds won.
func determineWinner(players map[uint64]*model.PlayerStats) string {
	teamWins := make(map[string]int)
	for _, p := range players {
		if p.TeamName != "" {
			teamWins[p.TeamName] = p.RoundsWon
		}
	}

	var winner string
	var maxWins int
	for team, wins := range teamWins {
		if wins > maxWins {
			maxWins = wins
			winner = team
		}
	}
	return winner
}

// safeDiv64 performs safe division returning 0 if denominator is 0.
func safeDiv64(num, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return num / denom
}
