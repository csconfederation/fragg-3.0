// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package export provides CSV file export functionality for player statistics.
// This file implements the FileExportOption which writes statistics to CSV files
// with comprehensive headers covering all tracked metrics.
package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"eco-rating/model"
	"eco-rating/output"
)

// FileExportOption implements ExportOption for CSV file output.
type FileExportOption struct {
	OutputPath string // Path where the CSV file will be written
}

// NewFileExportOption creates a new FileExportOption with the specified output path.
func NewFileExportOption(outputPath string) *FileExportOption {
	return &FileExportOption{OutputPath: outputPath}
}

// Export writes single-game player statistics to a CSV file.
// Players are sorted by FinalRating in descending order.
func (f *FileExportOption) Export(players map[uint64]*model.PlayerStats) error {
	if err := ensureDir(f.OutputPath); err != nil {
		return err
	}

	file, err := os.Create(f.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := getSingleGameHeader()
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	playerList := make([]*model.PlayerStats, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}
	sort.Slice(playerList, func(i, j int) bool {
		return playerList[i].FinalRating > playerList[j].FinalRating
	})

	for _, p := range playerList {
		row := getSingleGameRow(p)
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// ExportAggregated writes aggregated multi-game statistics to a CSV file.
// Players are sorted first by tier (highest to lowest), then by FinalRating.
func (f *FileExportOption) ExportAggregated(players map[string]*output.AggregatedStats) error {
	if err := ensureDir(f.OutputPath); err != nil {
		return err
	}

	file, err := os.Create(f.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := getAggregatedHeader()
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	tierOrder := map[string]int{
		"premier":    0,
		"elite":      1,
		"challenger": 2,
		"contender":  3,
		"prospect":   4,
		"recruit":    5,
	}

	playerList := make([]*output.AggregatedStats, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}
	sort.Slice(playerList, func(i, j int) bool {
		tierI := tierOrder[playerList[i].Tier]
		tierJ := tierOrder[playerList[j].Tier]
		if tierI != tierJ {
			return tierI < tierJ
		}
		return playerList[i].FinalRating > playerList[j].FinalRating
	})

	for _, p := range playerList {
		row := getAggregatedRow(p)
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// ensureDir creates the parent directory for the given path if it doesn't exist.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// getSingleGameHeader returns the CSV header row for single-game exports.
// Contains 140+ columns covering all tracked player metrics.
func getSingleGameHeader() []string {
	return []string{
		"Steam ID", "Name", "Final Rating", "HLTV Rating",
		"Rounds Played", "Rounds Won", "Rounds Lost",
		"Kills", "Assists", "Deaths", "Damage",
		"ADR", "KPR", "DPR", "KAST", "Survival",
		"Opening Kills", "Opening Deaths", "Opening Attempts", "Opening Successes",
		"Opening Kills Per Round", "Opening Deaths Per Round", "Opening Attempts Pct", "Opening Success Pct",
		"Rounds Won After Opening", "Win Pct After Opening Kill",
		"Eco Kill Value", "Eco Death Value", "Econ Impact", "Round Impact", "Round Swing",
		"Clutch Rounds", "Clutch Wins", "Clutch Points Per Round",
		"Clutch 1v1 Attempts", "Clutch 1v1 Wins", "Clutch 1v1 Win Pct",
		"Trade Kills", "Trade Kills Per Round", "Trade Kills Pct", "Fast Trades",
		"Traded Deaths", "Traded Deaths Per Round", "Traded Deaths Pct",
		"Trade Denials", "Saved By Teammate", "Saved By Teammate Per Round",
		"Saved Teammate", "Saved Teammate Per Round",
		"Opening Deaths Traded", "Opening Deaths Traded Pct",
		"AWP Kills", "AWP Kills Per Round", "AWP Kills Pct",
		"Rounds With AWP Kill", "Rounds With AWP Kill Pct",
		"AWP Multi Kill Rounds", "AWP Multi Kill Rounds Per Round",
		"AWP Opening Kills", "AWP Opening Kills Per Round",
		"AWP Deaths", "AWP Deaths No Kill",
		"1K", "2K", "3K", "4K", "5K",
		"Rounds With Kill", "Rounds With Kill Pct",
		"Rounds With Multi Kill", "Rounds With Multi Kill Pct",
		"Kills In Won Rounds", "Kills Per Round Win",
		"Damage In Won Rounds", "Damage Per Round Win",
		"Perfect Kills", "Damage Per Kill", "Knife Kills", "Pistol Vs Rifle Kills",
		"Support Rounds", "Support Rounds Pct",
		"Assisted Kills", "Assisted Kills Pct", "Assists Per Round",
		"Attack Rounds", "Attacks Per Round",
		"Time Alive Per Round", "Last Alive Rounds", "Last Alive Pct",
		"Saves On Loss", "Saves Per Round Loss",
		"Utility Damage", "Utility Damage Per Round",
		"Utility Kills", "Utility Kills Per 100 Rounds",
		"Flashes Thrown", "Flashes Thrown Per Round",
		"Flash Assists", "Flash Assists Per Round",
		"Enemy Flash Duration Per Round",
		"Team Flash Count", "Team Flash Duration Per Round",
		"Exit Frags", "Early Deaths",
		"Low Buy Kills", "Low Buy Kills Pct",
		"Disadvantaged Buy Kills", "Disadvantaged Buy Kills Pct",
		"Pistol Rounds Played", "Pistol Round Kills", "Pistol Round Deaths",
		"Pistol Round Damage", "Pistol Rounds Won", "Pistol Round Survivals",
		"Pistol Round Multi Kills", "Pistol Round Rating",
		"T Rounds Played", "T Kills", "T Deaths", "T Damage", "T Survivals",
		"T Rounds With Multi Kill", "T Eco Kill Value", "T Round Swing", "T KAST",
		"T Clutch Rounds", "T Clutch Wins", "T Rating", "T Eco Rating",
		"CT Rounds Played", "CT Kills", "CT Deaths", "CT Damage", "CT Survivals",
		"CT Rounds With Multi Kill", "CT Eco Kill Value", "CT Round Swing", "CT KAST",
		"CT Clutch Rounds", "CT Clutch Wins", "CT Rating", "CT Eco Rating",
	}
}

// getSingleGameRow converts a PlayerStats struct to a CSV row.
func getSingleGameRow(p *model.PlayerStats) []string {
	return []string{
		p.SteamID,
		p.Name,
		formatFloat(p.FinalRating),
		formatFloat(p.HLTVRating),
		strconv.Itoa(p.RoundsPlayed),
		strconv.Itoa(p.RoundsWon),
		strconv.Itoa(p.RoundsLost),
		strconv.Itoa(p.Kills),
		strconv.Itoa(p.Assists),
		strconv.Itoa(p.Deaths),
		strconv.Itoa(p.Damage),
		formatFloat(p.ADR),
		formatFloat(p.KPR),
		formatFloat(p.DPR),
		formatFloat(p.KAST),
		formatFloat(p.Survival),
		strconv.Itoa(p.OpeningKills),
		strconv.Itoa(p.OpeningDeaths),
		strconv.Itoa(p.OpeningAttempts),
		strconv.Itoa(p.OpeningSuccesses),
		formatFloat(p.OpeningKillsPerRound),
		formatFloat(p.OpeningDeathsPerRound),
		formatFloat(p.OpeningAttemptsPct),
		formatFloat(p.OpeningSuccessPct),
		strconv.Itoa(p.RoundsWonAfterOpening),
		formatFloat(p.WinPctAfterOpeningKill),
		formatFloat(p.EcoKillValue),
		formatFloat(p.EcoDeathValue),
		formatFloat(p.EconImpact),
		formatFloat(p.RoundImpact),
		formatFloat(p.RoundSwing),
		strconv.Itoa(p.ClutchRounds),
		strconv.Itoa(p.ClutchWins),
		formatFloat(p.ClutchPointsPerRound),
		strconv.Itoa(p.Clutch1v1Attempts),
		strconv.Itoa(p.Clutch1v1Wins),
		formatFloat(p.Clutch1v1WinPct),
		strconv.Itoa(p.TradeKills),
		formatFloat(p.TradeKillsPerRound),
		formatFloat(p.TradeKillsPct),
		strconv.Itoa(p.FastTrades),
		strconv.Itoa(p.TradedDeaths),
		formatFloat(p.TradedDeathsPerRound),
		formatFloat(p.TradedDeathsPct),
		strconv.Itoa(p.TradeDenials),
		strconv.Itoa(p.SavedByTeammate),
		formatFloat(p.SavedByTeammatePerRound),
		strconv.Itoa(p.SavedTeammate),
		formatFloat(p.SavedTeammatePerRound),
		strconv.Itoa(p.OpeningDeathsTraded),
		formatFloat(p.OpeningDeathsTradedPct),
		strconv.Itoa(p.AWPKills),
		formatFloat(p.AWPKillsPerRound),
		formatFloat(p.AWPKillsPct),
		strconv.Itoa(p.RoundsWithAWPKill),
		formatFloat(p.RoundsWithAWPKillPct),
		strconv.Itoa(p.AWPMultiKillRounds),
		formatFloat(p.AWPMultiKillRoundsPerRound),
		strconv.Itoa(p.AWPOpeningKills),
		formatFloat(p.AWPOpeningKillsPerRound),
		strconv.Itoa(p.AWPDeaths),
		strconv.Itoa(p.AWPDeathsNoKill),
		strconv.Itoa(p.MultiKills.OneK),
		strconv.Itoa(p.MultiKills.TwoK),
		strconv.Itoa(p.MultiKills.ThreeK),
		strconv.Itoa(p.MultiKills.FourK),
		strconv.Itoa(p.MultiKills.FiveK),
		strconv.Itoa(p.RoundsWithKill),
		formatFloat(p.RoundsWithKillPct),
		strconv.Itoa(p.RoundsWithMultiKill),
		formatFloat(p.RoundsWithMultiKillPct),
		strconv.Itoa(p.KillsInWonRounds),
		formatFloat(p.KillsPerRoundWin),
		strconv.Itoa(p.DamageInWonRounds),
		formatFloat(p.DamagePerRoundWin),
		strconv.Itoa(p.PerfectKills),
		formatFloat(p.DamagePerKill),
		strconv.Itoa(p.KnifeKills),
		strconv.Itoa(p.PistolVsRifleKills),
		strconv.Itoa(p.SupportRounds),
		formatFloat(p.SupportRoundsPct),
		strconv.Itoa(p.AssistedKills),
		formatFloat(p.AssistedKillsPct),
		formatFloat(p.AssistsPerRound),
		strconv.Itoa(p.AttackRounds),
		formatFloat(p.AttacksPerRound),
		formatFloat(p.TimeAlivePerRound),
		strconv.Itoa(p.LastAliveRounds),
		formatFloat(p.LastAlivePct),
		strconv.Itoa(p.SavesOnLoss),
		formatFloat(p.SavesPerRoundLoss),
		strconv.Itoa(p.UtilityDamage),
		formatFloat(p.UtilityDamagePerRound),
		strconv.Itoa(p.UtilityKills),
		formatFloat(p.UtilityKillsPer100Rounds),
		strconv.Itoa(p.FlashesThrown),
		formatFloat(p.FlashesThrownPerRound),
		strconv.Itoa(p.FlashAssists),
		formatFloat(p.FlashAssistsPerRound),
		formatFloat(p.EnemyFlashDurationPerRound),
		strconv.Itoa(p.TeamFlashCount),
		formatFloat(p.TeamFlashDurationPerRound),
		strconv.Itoa(p.ExitFrags),
		strconv.Itoa(p.EarlyDeaths),
		strconv.Itoa(p.LowBuyKills),
		formatFloat(p.LowBuyKillsPct),
		strconv.Itoa(p.DisadvantagedBuyKills),
		formatFloat(p.DisadvantagedBuyKillsPct),
		strconv.Itoa(p.PistolRoundsPlayed),
		strconv.Itoa(p.PistolRoundKills),
		strconv.Itoa(p.PistolRoundDeaths),
		strconv.Itoa(p.PistolRoundDamage),
		strconv.Itoa(p.PistolRoundsWon),
		strconv.Itoa(p.PistolRoundSurvivals),
		strconv.Itoa(p.PistolRoundMultiKills),
		formatFloat(p.PistolRoundRating),
		strconv.Itoa(p.TRoundsPlayed),
		strconv.Itoa(p.TKills),
		strconv.Itoa(p.TDeaths),
		strconv.Itoa(p.TDamage),
		strconv.Itoa(p.TSurvivals),
		strconv.Itoa(p.TRoundsWithMultiKill),
		formatFloat(p.TEcoKillValue),
		formatFloat(p.TRoundSwing),
		formatFloat(p.TKAST),
		strconv.Itoa(p.TClutchRounds),
		strconv.Itoa(p.TClutchWins),
		formatFloat(p.TRating),
		formatFloat(p.TEcoRating),
		strconv.Itoa(p.CTRoundsPlayed),
		strconv.Itoa(p.CTKills),
		strconv.Itoa(p.CTDeaths),
		strconv.Itoa(p.CTDamage),
		strconv.Itoa(p.CTSurvivals),
		strconv.Itoa(p.CTRoundsWithMultiKill),
		formatFloat(p.CTEcoKillValue),
		formatFloat(p.CTRoundSwing),
		formatFloat(p.CTKAST),
		strconv.Itoa(p.CTClutchRounds),
		strconv.Itoa(p.CTClutchWins),
		formatFloat(p.CTRating),
		formatFloat(p.CTEcoRating),
	}
}

// getAggregatedHeader returns the CSV header row for aggregated exports.
// Includes additional columns for games count, tier, and per-map statistics.
func getAggregatedHeader() []string {
	return []string{
		"Steam ID", "Name", "Tier", "Games", "Final Rating", "HLTV Rating",
		"Rounds Played", "Rounds Won", "Rounds Lost",
		"Kills", "Assists", "Deaths", "Damage",
		"ADR", "KPR", "DPR", "KAST", "Survival",
		"Opening Kills", "Opening Deaths", "Opening Attempts", "Opening Successes",
		"Opening Kills Per Round", "Opening Deaths Per Round", "Opening Attempts Pct", "Opening Success Pct",
		"Rounds Won After Opening", "Win Pct After Opening Kill",
		"Eco Kill Value", "Eco Death Value", "Econ Impact", "Round Impact", "Round Swing",
		"Clutch Rounds", "Clutch Wins", "Clutch Points Per Round",
		"Clutch 1v1 Attempts", "Clutch 1v1 Wins", "Clutch 1v1 Win Pct",
		"Trade Kills", "Trade Kills Per Round", "Trade Kills Pct", "Fast Trades",
		"Traded Deaths", "Traded Deaths Per Round", "Traded Deaths Pct",
		"Trade Denials", "Saved By Teammate", "Saved By Teammate Per Round",
		"Saved Teammate", "Saved Teammate Per Round",
		"Opening Deaths Traded", "Opening Deaths Traded Pct",
		"AWP Kills", "AWP Kills Per Round", "AWP Kills Pct",
		"Rounds With AWP Kill", "Rounds With AWP Kill Pct",
		"AWP Multi Kill Rounds", "AWP Multi Kill Rounds Per Round",
		"AWP Opening Kills", "AWP Opening Kills Per Round",
		"AWP Deaths", "AWP Deaths No Kill",
		"1K", "2K", "3K", "4K", "5K",
		"Rounds With Kill", "Rounds With Kill Pct",
		"Rounds With Multi Kill", "Rounds With Multi Kill Pct",
		"Kills In Won Rounds", "Kills Per Round Win",
		"Damage In Won Rounds", "Damage Per Round Win",
		"Perfect Kills", "Damage Per Kill", "Knife Kills", "Pistol Vs Rifle Kills",
		"Support Rounds", "Support Rounds Pct",
		"Assisted Kills", "Assisted Kills Pct", "Assists Per Round",
		"Attack Rounds", "Attacks Per Round",
		"Time Alive Per Round", "Last Alive Rounds", "Last Alive Pct",
		"Saves On Loss", "Saves Per Round Loss",
		"Utility Damage", "Utility Damage Per Round",
		"Utility Kills", "Utility Kills Per 100 Rounds",
		"Flashes Thrown", "Flashes Thrown Per Round",
		"Flash Assists", "Flash Assists Per Round",
		"Enemy Flash Duration Per Round",
		"Team Flash Count", "Team Flash Duration Per Round",
		"Exit Frags", "Early Deaths",
		"Low Buy Kills", "Low Buy Kills Pct",
		"Disadvantaged Buy Kills", "Disadvantaged Buy Kills Pct",
		"Pistol Rounds Played", "Pistol Round Kills", "Pistol Round Deaths",
		"Pistol Round Damage", "Pistol Rounds Won", "Pistol Round Survivals",
		"Pistol Round Multi Kills", "Pistol Round Rating",
		"T Rounds Played", "T Kills", "T Deaths", "T Damage", "T Survivals",
		"T Rounds With Multi Kill", "T Eco Kill Value", "T Round Swing", "T KAST",
		"T Clutch Rounds", "T Clutch Wins", "T Rating", "T Eco Rating",
		"CT Rounds Played", "CT Kills", "CT Deaths", "CT Damage", "CT Survivals",
		"CT Rounds With Multi Kill", "CT Eco Kill Value", "CT Round Swing", "CT KAST",
		"CT Clutch Rounds", "CT Clutch Wins", "CT Rating", "CT Eco Rating",
		"Ancient Rating", "Ancient Games",
		"Anubis Rating", "Anubis Games",
		"Dust2 Rating", "Dust2 Games",
		"Inferno Rating", "Inferno Games",
		"Mirage Rating", "Mirage Games",
		"Nuke Rating", "Nuke Games",
		"Overpass Rating", "Overpass Games",
	}
}

// getAggregatedRow converts an AggregatedStats struct to a CSV row.
func getAggregatedRow(p *output.AggregatedStats) []string {
	return []string{
		p.SteamID,
		p.Name,
		p.Tier,
		strconv.Itoa(p.GamesCount),
		formatFloat(p.FinalRating),
		formatFloat(p.HLTVRating),
		strconv.Itoa(p.RoundsPlayed),
		strconv.Itoa(p.RoundsWon),
		strconv.Itoa(p.RoundsLost),
		strconv.Itoa(p.Kills),
		strconv.Itoa(p.Assists),
		strconv.Itoa(p.Deaths),
		strconv.Itoa(p.Damage),
		formatFloat(p.ADR),
		formatFloat(p.KPR),
		formatFloat(p.DPR),
		formatFloat(p.KAST),
		formatFloat(p.Survival),
		strconv.Itoa(p.OpeningKills),
		strconv.Itoa(p.OpeningDeaths),
		strconv.Itoa(p.OpeningAttempts),
		strconv.Itoa(p.OpeningSuccesses),
		formatFloat(p.OpeningKillsPerRound),
		formatFloat(p.OpeningDeathsPerRound),
		formatFloat(p.OpeningAttemptsPct),
		formatFloat(p.OpeningSuccessPct),
		strconv.Itoa(p.RoundsWonAfterOpening),
		formatFloat(p.WinPctAfterOpeningKill),
		formatFloat(p.EcoKillValue),
		formatFloat(p.EcoDeathValue),
		formatFloat(p.EconImpact),
		formatFloat(p.RoundImpact),
		formatFloat(p.RoundSwing),
		strconv.Itoa(p.ClutchRounds),
		strconv.Itoa(p.ClutchWins),
		formatFloat(p.ClutchPointsPerRound),
		strconv.Itoa(p.Clutch1v1Attempts),
		strconv.Itoa(p.Clutch1v1Wins),
		formatFloat(p.Clutch1v1WinPct),
		strconv.Itoa(p.TradeKills),
		formatFloat(p.TradeKillsPerRound),
		formatFloat(p.TradeKillsPct),
		strconv.Itoa(p.FastTrades),
		strconv.Itoa(p.TradedDeaths),
		formatFloat(p.TradedDeathsPerRound),
		formatFloat(p.TradedDeathsPct),
		strconv.Itoa(p.TradeDenials),
		strconv.Itoa(p.SavedByTeammate),
		formatFloat(p.SavedByTeammatePerRound),
		strconv.Itoa(p.SavedTeammate),
		formatFloat(p.SavedTeammatePerRound),
		strconv.Itoa(p.OpeningDeathsTraded),
		formatFloat(p.OpeningDeathsTradedPct),
		strconv.Itoa(p.AWPKills),
		formatFloat(p.AWPKillsPerRound),
		formatFloat(p.AWPKillsPct),
		strconv.Itoa(p.RoundsWithAWPKill),
		formatFloat(p.RoundsWithAWPKillPct),
		strconv.Itoa(p.AWPMultiKillRounds),
		formatFloat(p.AWPMultiKillRoundsPerRound),
		strconv.Itoa(p.AWPOpeningKills),
		formatFloat(p.AWPOpeningKillsPerRound),
		strconv.Itoa(p.AWPDeaths),
		strconv.Itoa(p.AWPDeathsNoKill),
		strconv.Itoa(p.MultiKills.OneK),
		strconv.Itoa(p.MultiKills.TwoK),
		strconv.Itoa(p.MultiKills.ThreeK),
		strconv.Itoa(p.MultiKills.FourK),
		strconv.Itoa(p.MultiKills.FiveK),
		strconv.Itoa(p.RoundsWithKill),
		formatFloat(p.RoundsWithKillPct),
		strconv.Itoa(p.RoundsWithMultiKill),
		formatFloat(p.RoundsWithMultiKillPct),
		strconv.Itoa(p.KillsInWonRounds),
		formatFloat(p.KillsPerRoundWin),
		strconv.Itoa(p.DamageInWonRounds),
		formatFloat(p.DamagePerRoundWin),
		strconv.Itoa(p.PerfectKills),
		formatFloat(p.DamagePerKill),
		strconv.Itoa(p.KnifeKills),
		strconv.Itoa(p.PistolVsRifleKills),
		strconv.Itoa(p.SupportRounds),
		formatFloat(p.SupportRoundsPct),
		strconv.Itoa(p.AssistedKills),
		formatFloat(p.AssistedKillsPct),
		formatFloat(p.AssistsPerRound),
		strconv.Itoa(p.AttackRounds),
		formatFloat(p.AttacksPerRound),
		formatFloat(p.TimeAlivePerRound),
		strconv.Itoa(p.LastAliveRounds),
		formatFloat(p.LastAlivePct),
		strconv.Itoa(p.SavesOnLoss),
		formatFloat(p.SavesPerRoundLoss),
		strconv.Itoa(p.UtilityDamage),
		formatFloat(p.UtilityDamagePerRound),
		strconv.Itoa(p.UtilityKills),
		formatFloat(p.UtilityKillsPer100Rounds),
		strconv.Itoa(p.FlashesThrown),
		formatFloat(p.FlashesThrownPerRound),
		strconv.Itoa(p.FlashAssists),
		formatFloat(p.FlashAssistsPerRound),
		formatFloat(p.EnemyFlashDurationPerRound),
		strconv.Itoa(p.TeamFlashCount),
		formatFloat(p.TeamFlashDurationPerRound),
		strconv.Itoa(p.ExitFrags),
		strconv.Itoa(p.EarlyDeaths),
		strconv.Itoa(p.LowBuyKills),
		formatFloat(p.LowBuyKillsPct),
		strconv.Itoa(p.DisadvantagedBuyKills),
		formatFloat(p.DisadvantagedBuyKillsPct),
		strconv.Itoa(p.PistolRoundsPlayed),
		strconv.Itoa(p.PistolRoundKills),
		strconv.Itoa(p.PistolRoundDeaths),
		strconv.Itoa(p.PistolRoundDamage),
		strconv.Itoa(p.PistolRoundsWon),
		strconv.Itoa(p.PistolRoundSurvivals),
		strconv.Itoa(p.PistolRoundMultiKills),
		formatFloat(p.PistolRoundRating),
		strconv.Itoa(p.TRoundsPlayed),
		strconv.Itoa(p.TKills),
		strconv.Itoa(p.TDeaths),
		strconv.Itoa(p.TDamage),
		strconv.Itoa(p.TSurvivals),
		strconv.Itoa(p.TRoundsWithMultiKill),
		formatFloat(p.TEcoKillValue),
		formatFloat(p.TRoundSwing),
		formatFloat(p.TKAST),
		strconv.Itoa(p.TClutchRounds),
		strconv.Itoa(p.TClutchWins),
		formatFloat(p.TRating),
		formatFloat(p.TEcoRating),
		strconv.Itoa(p.CTRoundsPlayed),
		strconv.Itoa(p.CTKills),
		strconv.Itoa(p.CTDeaths),
		strconv.Itoa(p.CTDamage),
		strconv.Itoa(p.CTSurvivals),
		strconv.Itoa(p.CTRoundsWithMultiKill),
		formatFloat(p.CTEcoKillValue),
		formatFloat(p.CTRoundSwing),
		formatFloat(p.CTKAST),
		strconv.Itoa(p.CTClutchRounds),
		strconv.Itoa(p.CTClutchWins),
		formatFloat(p.CTRating),
		formatFloat(p.CTEcoRating),
		getMapRating(p, "de_ancient"),
		getMapGames(p, "de_ancient"),
		getMapRating(p, "de_anubis"),
		getMapGames(p, "de_anubis"),
		getMapRating(p, "de_dust2"),
		getMapGames(p, "de_dust2"),
		getMapRating(p, "de_inferno"),
		getMapGames(p, "de_inferno"),
		getMapRating(p, "de_mirage"),
		getMapGames(p, "de_mirage"),
		getMapRating(p, "de_nuke"),
		getMapGames(p, "de_nuke"),
		getMapRating(p, "de_overpass"),
		getMapGames(p, "de_overpass"),
	}
}

// getMapRating returns the player's rating for a specific map, or empty string if not played.
func getMapRating(p *output.AggregatedStats, mapName string) string {
	if p.MapRatings == nil {
		return ""
	}
	if rating, ok := p.MapRatings[mapName]; ok {
		return formatFloat(rating)
	}
	return ""
}

// getMapGames returns the number of games played on a specific map, or empty string if none.
func getMapGames(p *output.AggregatedStats, mapName string) string {
	if p.MapGamesPlayed == nil {
		return ""
	}
	if games, ok := p.MapGamesPlayed[mapName]; ok {
		return strconv.Itoa(games)
	}
	return ""
}

// formatFloat converts a float64 to a string with 3 decimal places.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
