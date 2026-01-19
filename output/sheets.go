package output

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// SheetsClient handles Google Sheets operations
type SheetsClient struct {
	service       *sheets.Service
	spreadsheetID string
	sheetName     string
}

// NewSheetsClient creates a new Google Sheets client using service account credentials
func NewSheetsClient(credentialsJSON []byte, sheetURL, sheetName string) (*SheetsClient, error) {
	ctx := context.Background()

	// Parse credentials and create JWT config
	config, err := google.JWTConfigFromJSON(credentialsJSON, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create the Sheets service
	client := config.Client(ctx)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	// Extract spreadsheet ID from URL
	spreadsheetID, err := extractSpreadsheetID(sheetURL)
	if err != nil {
		return nil, err
	}

	return &SheetsClient{
		service:       srv,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

// getMapRating returns the rating for a specific map, or empty string if not played
func getMapRating(p *AggregatedStats, mapName string) interface{} {
	if p.MapRatings == nil {
		return ""
	}
	if rating, ok := p.MapRatings[mapName]; ok {
		return rating
	}
	return ""
}

// getMapGames returns the games played for a specific map, or empty string if not played
func getMapGames(p *AggregatedStats, mapName string) interface{} {
	if p.MapGamesPlayed == nil {
		return ""
	}
	if games, ok := p.MapGamesPlayed[mapName]; ok {
		return games
	}
	return ""
}

// extractSpreadsheetID extracts the spreadsheet ID from a Google Sheets URL
func extractSpreadsheetID(url string) (string, error) {
	// Match pattern: /spreadsheets/d/{spreadsheetId}/
	re := regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract spreadsheet ID from URL: %s", url)
	}
	return matches[1], nil
}

// UploadAggregatedStats uploads aggregated player stats to the Google Sheet
func (c *SheetsClient) UploadAggregatedStats(players map[string]*AggregatedStats) error {
	ctx := context.Background()

	// Build header row
	headers := []interface{}{
		"Steam ID", "Name", "Tier", "Final Rating", "Games", "Rounds Played", "Rounds Won", "Rounds Lost",
		"Kills", "Assists", "Deaths", "Damage", "Opening Kills",
		"ADR", "KPR", "DPR",
		"Perfect Kills", "Trade Denials", "Traded Deaths",
		"Rounds With Kill", "Rounds With Multi Kill", "Kills In Won Rounds", "Damage In Won Rounds",
		"AWP Kills", "AWP Kills Per Round", "Rounds With AWP Kill", "AWP Multi Kill Rounds", "AWP Opening Kills",
		"1K", "2K", "3K", "4K", "5K",
		"Round Impact", "Survival", "KAST", "Econ Impact",
		"Eco Kill Value", "Eco Death Value",
		"Round Swing", "Clutch Rounds", "Clutch Wins",
		"Saved By Teammate", "Saved Teammate", "Opening Deaths", "Opening Deaths Traded", "Support Rounds", "Assisted Kills",
		"Opening Attempts", "Opening Successes", "Rounds Won After Opening", "Attack Rounds",
		"Clutch 1v1 Attempts", "Clutch 1v1 Wins", "Time Alive Per Round", "Last Alive Rounds", "Saves On Loss",
		"Utility Damage", "Utility Kills", "Flashes Thrown", "Flash Assists",
		"Enemy Flash Duration Per Round", "Team Flash Count", "Team Flash Duration Per Round",
		"Exit Frags", "AWP Deaths", "AWP Deaths No Kill", "Knife Kills", "Pistol Vs Rifle Kills",
		"Trade Kills", "Fast Trades", "Early Deaths", "Low Buy Kills", "Low Buy Kills Pct", "Disadvantaged Buy Kills", "Disadvantaged Buy Kills Pct",
		"Pistol Rounds Played", "Pistol Round Kills", "Pistol Round Deaths", "Pistol Round Damage",
		"Pistol Rounds Won", "Pistol Round Survivals", "Pistol Round Multi Kills", "Pistol Round Rating",
		"T Rounds Played", "T Kills", "T Deaths", "T Damage", "T Survivals", "T Rounds With Multi Kill",
		"T Eco Kill Value", "T Round Swing", "T KAST", "T Clutch Rounds", "T Clutch Wins", "T Rating", "T Eco Rating",
		"CT Rounds Played", "CT Kills", "CT Deaths", "CT Damage", "CT Survivals", "CT Rounds With Multi Kill",
		"CT Eco Kill Value", "CT Round Swing", "CT KAST", "CT Clutch Rounds", "CT Clutch Wins", "CT Rating", "CT Eco Rating",
		"HLTV Rating",
		"Rounds With Kill Pct", "Kills Per Round Win", "Rounds With Multi Kill Pct", "Damage Per Round Win",
		"Saved By Teammate Per Round", "Traded Deaths Per Round", "Traded Deaths Pct", "Opening Deaths Traded Pct",
		"Assists Per Round", "Support Rounds Pct", "Saved Teammate Per Round", "Trade Kills Per Round", "Trade Kills Pct",
		"Assisted Kills Pct", "Damage Per Kill", "Opening Kills Per Round", "Opening Deaths Per Round",
		"Opening Attempts Pct", "Opening Success Pct", "Win Pct After Opening Kill", "Attacks Per Round",
		"Clutch Points Per Round", "Last Alive Pct", "Clutch 1v1 Win Pct", "Saves Per Round Loss",
		"AWP Kills Pct", "Rounds With AWP Kill Pct", "AWP Multi Kill Rounds Per Round", "AWP Opening Kills Per Round",
		"Utility Damage Per Round", "Utility Kills Per 100 Rounds", "Flashes Thrown Per Round", "Flash Assists Per Round",
		"Ancient Rating", "Ancient Games", "Anubis Rating", "Anubis Games", "Dust2 Rating", "Dust2 Games",
		"Inferno Rating", "Inferno Games", "Mirage Rating", "Mirage Games", "Nuke Rating", "Nuke Games", "Overpass Rating", "Overpass Games",
	}

	// Build data rows
	var rows [][]interface{}
	rows = append(rows, headers)

	// Tier priority order: premier > elite > challenger > contender > prospect > recruit
	tierOrder := map[string]int{
		"premier":    0,
		"elite":      1,
		"challenger": 2,
		"contender":  3,
		"prospect":   4,
		"recruit":    5,
	}

	// Collect all players into a slice for sorting
	playerList := make([]*AggregatedStats, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}

	// Sort by tier (ascending order = premier first), then by final rating descending within tier
	sort.Slice(playerList, func(i, j int) bool {
		tierI := tierOrder[playerList[i].Tier]
		tierJ := tierOrder[playerList[j].Tier]
		if tierI != tierJ {
			return tierI < tierJ
		}
		return playerList[i].FinalRating > playerList[j].FinalRating
	})

	for _, p := range playerList {
		row := []interface{}{
			p.SteamID, p.Name, p.Tier, p.FinalRating, p.GamesCount, p.RoundsPlayed, p.RoundsWon, p.RoundsLost,
			p.Kills, p.Assists, p.Deaths, p.Damage, p.OpeningKills,
			p.ADR, p.KPR, p.DPR,
			p.PerfectKills, p.TradeDenials, p.TradedDeaths,
			p.RoundsWithKill, p.RoundsWithMultiKill, p.KillsInWonRounds, p.DamageInWonRounds,
			p.AWPKills, p.AWPKillsPerRound, p.RoundsWithAWPKill, p.AWPMultiKillRounds, p.AWPOpeningKills,
			p.MultiKills.OneK, p.MultiKills.TwoK, p.MultiKills.ThreeK, p.MultiKills.FourK, p.MultiKills.FiveK,
			p.RoundImpact, p.Survival, p.KAST, p.EconImpact,
			p.EcoKillValue, p.EcoDeathValue,
			p.RoundSwing, p.ClutchRounds, p.ClutchWins,
			p.SavedByTeammate, p.SavedTeammate, p.OpeningDeaths, p.OpeningDeathsTraded, p.SupportRounds, p.AssistedKills,
			p.OpeningAttempts, p.OpeningSuccesses, p.RoundsWonAfterOpening, p.AttackRounds,
			p.Clutch1v1Attempts, p.Clutch1v1Wins, p.TimeAlivePerRound, p.LastAliveRounds, p.SavesOnLoss,
			p.UtilityDamage, p.UtilityKills, p.FlashesThrown, p.FlashAssists,
			p.EnemyFlashDurationPerRound, p.TeamFlashCount, p.TeamFlashDurationPerRound,
			p.ExitFrags, p.AWPDeaths, p.AWPDeathsNoKill, p.KnifeKills, p.PistolVsRifleKills,
			p.TradeKills, p.FastTrades, p.EarlyDeaths, p.LowBuyKills, p.LowBuyKillsPct, p.DisadvantagedBuyKills, p.DisadvantagedBuyKillsPct,
			p.PistolRoundsPlayed, p.PistolRoundKills, p.PistolRoundDeaths, p.PistolRoundDamage,
			p.PistolRoundsWon, p.PistolRoundSurvivals, p.PistolRoundMultiKills, p.PistolRoundRating,
			p.TRoundsPlayed, p.TKills, p.TDeaths, p.TDamage, p.TSurvivals, p.TRoundsWithMultiKill,
			p.TEcoKillValue, p.TRoundSwing, p.TKAST, p.TClutchRounds, p.TClutchWins, p.TRating, p.TEcoRating,
			p.CTRoundsPlayed, p.CTKills, p.CTDeaths, p.CTDamage, p.CTSurvivals, p.CTRoundsWithMultiKill,
			p.CTEcoKillValue, p.CTRoundSwing, p.CTKAST, p.CTClutchRounds, p.CTClutchWins, p.CTRating, p.CTEcoRating,
			p.HLTVRating,
			p.RoundsWithKillPct, p.KillsPerRoundWin, p.RoundsWithMultiKillPct, p.DamagePerRoundWin,
			p.SavedByTeammatePerRound, p.TradedDeathsPerRound, p.TradedDeathsPct, p.OpeningDeathsTradedPct,
			p.AssistsPerRound, p.SupportRoundsPct, p.SavedTeammatePerRound, p.TradeKillsPerRound, p.TradeKillsPct,
			p.AssistedKillsPct, p.DamagePerKill, p.OpeningKillsPerRound, p.OpeningDeathsPerRound,
			p.OpeningAttemptsPct, p.OpeningSuccessPct, p.WinPctAfterOpeningKill, p.AttacksPerRound,
			p.ClutchPointsPerRound, p.LastAlivePct, p.Clutch1v1WinPct, p.SavesPerRoundLoss,
			p.AWPKillsPct, p.RoundsWithAWPKillPct, p.AWPMultiKillRoundsPerRound, p.AWPOpeningKillsPerRound,
			p.UtilityDamagePerRound, p.UtilityKillsPer100Rounds, p.FlashesThrownPerRound, p.FlashAssistsPerRound,
			getMapRating(p, "de_ancient"), getMapGames(p, "de_ancient"),
			getMapRating(p, "de_anubis"), getMapGames(p, "de_anubis"),
			getMapRating(p, "de_dust2"), getMapGames(p, "de_dust2"),
			getMapRating(p, "de_inferno"), getMapGames(p, "de_inferno"),
			getMapRating(p, "de_mirage"), getMapGames(p, "de_mirage"),
			getMapRating(p, "de_nuke"), getMapGames(p, "de_nuke"),
			getMapRating(p, "de_overpass"), getMapGames(p, "de_overpass"),
		}
		rows = append(rows, row)
	}

	// Clear existing data in the sheet first
	clearRange := fmt.Sprintf("%s!A:ZZ", c.sheetName)
	_, err := c.service.Spreadsheets.Values.Clear(c.spreadsheetID, clearRange, &sheets.ClearValuesRequest{}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to clear sheet: %w", err)
	}

	// Write the data
	writeRange := fmt.Sprintf("%s!A1", c.sheetName)
	valueRange := &sheets.ValueRange{
		Values: rows,
	}

	_, err = c.service.Spreadsheets.Values.Update(c.spreadsheetID, writeRange, valueRange).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to write to sheet: %w", err)
	}

	return nil
}
