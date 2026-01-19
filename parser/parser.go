package parser

import (
	"eco-rating/model"
	"eco-rating/output"
	"eco-rating/rating"
	"io"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
)

type DemoParser struct {
	parser demoinfocs.Parser
	state  *MatchState
	logger *Logger
}

// NewDemoParser creates a new demo parser with logging disabled by default
func NewDemoParser(r io.Reader) *DemoParser {
	return NewDemoParserWithLogging(r, false)
}

// NewDemoParserWithLogging creates a new demo parser with optional logging
func NewDemoParserWithLogging(r io.Reader, enableLogging bool) *DemoParser {
	p := demoinfocs.NewParser(r)
	state := NewMatchState()

	dp := &DemoParser{
		parser: p,
		state:  state,
		logger: NewLogger(enableLogging),
	}

	dp.registerHandlers()
	return dp
}

// SetLogging enables or disables logging
func (d *DemoParser) SetLogging(enabled bool) {
	d.logger.SetEnabled(enabled)
}

// SetPlayerFilter sets the list of players to filter logs for
// Only events involving these players will be logged
// Pass empty slice to clear filter and log all events
func (d *DemoParser) SetPlayerFilter(players []string) {
	d.logger.SetPlayerFilter(players)
}

// AddPlayerFilter adds a player to the filter list
func (d *DemoParser) AddPlayerFilter(player string) {
	d.logger.AddPlayerFilter(player)
}

// ClearPlayerFilter clears the player filter (logs all events)
func (d *DemoParser) ClearPlayerFilter() {
	d.logger.ClearPlayerFilter()
}

func (d *DemoParser) Parse() {
	_ = d.parser.ParseToEnd()

	// Compute final stats for all players
	for _, p := range d.state.Players {
		if p.RoundsPlayed > 0 {
			rounds := float64(p.RoundsPlayed)
			// Calculate per-round stats
			p.ADR = float64(p.Damage) / rounds
			p.KPR = float64(p.Kills) / rounds
			p.DPR = float64(p.Deaths) / rounds
			// Convert KAST count to percentage
			p.KAST = p.KAST / rounds
			// Convert Survival count to percentage
			p.Survival = p.Survival / rounds

			// Calculate AWP kills per round
			if p.RoundsPlayed > 0 {
				p.AWPKillsPerRound = float64(p.AWPKills) / rounds
			}

			// Calculate HLTV Rating 1.0
			// Formula: (KillRating + 0.7*SurvivalRating + RoundsWithMultiKillRating) / 2.7
			// Where each component is normalized against average values
			avgKPR := 0.679 // Average kills per round
			avgSPR := 0.317 // Average (Survived - Deaths) / Rounds
			avgRMK := 1.277 // Average RMK points per round (weighted multi-kills)

			killRating := p.KPR / avgKPR
			// SurvivalRating: (Rounds Survived - Deaths) / Rounds, normalized
			survived := p.Survival * rounds // p.Survival is already survival rate, convert back to count
			survivalRating := ((survived - float64(p.Deaths)) / rounds) / avgSPR
			// RMK: weighted multi-kill points (1K=1, 2K=4, 3K=9, 4K=16, 5K=25)
			rmkPoints := float64(p.MultiKillsRaw[1]*1 + p.MultiKillsRaw[2]*4 + p.MultiKillsRaw[3]*9 + p.MultiKillsRaw[4]*16 + p.MultiKillsRaw[5]*25)
			rmkRating := (rmkPoints / rounds) / avgRMK

			p.HLTVRating = (killRating + 0.7*survivalRating + rmkRating) / 2.7

			// Calculate Pistol Round Rating (HLTV 1.0 for pistol rounds only)
			if p.PistolRoundsPlayed > 0 {
				pistolRounds := float64(p.PistolRoundsPlayed)
				pistolKPR := float64(p.PistolRoundKills) / pistolRounds
				// SurvivalRating: (Survived - Deaths) / Rounds
				pistolSurvivalRating := ((float64(p.PistolRoundSurvivals) - float64(p.PistolRoundDeaths)) / pistolRounds) / avgSPR
				// RMK: approximate using multi-kill rounds * 4 (average weight for 2K)
				pistolRMKPoints := float64(p.PistolRoundMultiKills) * 4.0
				pistolRMKRating := (pistolRMKPoints / pistolRounds) / avgRMK

				pistolKillRating := pistolKPR / avgKPR
				p.PistolRoundRating = (pistolKillRating + 0.7*pistolSurvivalRating + pistolRMKRating) / 2.7
			}

			// Calculate T-side HLTV Rating 1.0
			if p.TRoundsPlayed > 0 {
				tRounds := float64(p.TRoundsPlayed)
				tKPR := float64(p.TKills) / tRounds
				// SurvivalRating: (Survived - Deaths) / Rounds
				tSurvivalRating := ((float64(p.TSurvivals) - float64(p.TDeaths)) / tRounds) / avgSPR
				// RMK: weighted multi-kill points
				tRMKPoints := float64(p.TMultiKills[1]*1 + p.TMultiKills[2]*4 + p.TMultiKills[3]*9 + p.TMultiKills[4]*16 + p.TMultiKills[5]*25)
				tRMKRating := (tRMKPoints / tRounds) / avgRMK

				tKillRating := tKPR / avgKPR
				p.TRating = (tKillRating + 0.7*tSurvivalRating + tRMKRating) / 2.7
			}

			// Calculate CT-side HLTV Rating 1.0
			if p.CTRoundsPlayed > 0 {
				ctRounds := float64(p.CTRoundsPlayed)
				ctKPR := float64(p.CTKills) / ctRounds
				// SurvivalRating: (Survived - Deaths) / Rounds
				ctSurvivalRating := ((float64(p.CTSurvivals) - float64(p.CTDeaths)) / ctRounds) / avgSPR
				// RMK: weighted multi-kill points
				ctRMKPoints := float64(p.CTMultiKills[1]*1 + p.CTMultiKills[2]*4 + p.CTMultiKills[3]*9 + p.CTMultiKills[4]*16 + p.CTMultiKills[5]*25)
				ctRMKRating := (ctRMKPoints / ctRounds) / avgRMK

				ctKillRating := ctKPR / avgKPR
				p.CTRating = (ctKillRating + 0.7*ctSurvivalRating + ctRMKRating) / 2.7
			}

			// Calculate all per-round and percentage stats
			p.TimeAlivePerRound = p.TotalTimeAlive / rounds
			p.EnemyFlashDurationPerRound = p.EnemyFlashDuration / rounds
			p.TeamFlashDurationPerRound = p.TeamFlashDuration / rounds
			p.RoundsWithKillPct = float64(p.RoundsWithKill) / rounds
			p.RoundsWithMultiKillPct = float64(p.RoundsWithMultiKill) / rounds
			p.SavedByTeammatePerRound = float64(p.SavedByTeammate) / rounds
			p.TradedDeathsPerRound = float64(p.TradedDeaths) / rounds
			p.AssistsPerRound = float64(p.Assists) / rounds
			p.SupportRoundsPct = float64(p.SupportRounds) / rounds
			p.SavedTeammatePerRound = float64(p.SavedTeammate) / rounds
			p.TradeKillsPerRound = float64(p.TradeKills) / rounds
			p.OpeningKillsPerRound = float64(p.OpeningKills) / rounds
			p.OpeningDeathsPerRound = float64(p.OpeningDeaths) / rounds
			p.OpeningAttemptsPct = float64(p.OpeningAttempts) / rounds
			p.AttacksPerRound = float64(p.AttackRounds) / rounds
			p.ClutchPointsPerRound = float64(p.ClutchWins) / rounds
			p.LastAlivePct = float64(p.LastAliveRounds) / rounds
			p.RoundsWithAWPKillPct = float64(p.RoundsWithAWPKill) / rounds
			p.AWPMultiKillRoundsPerRound = float64(p.AWPMultiKillRounds) / rounds
			p.AWPOpeningKillsPerRound = float64(p.AWPOpeningKills) / rounds
			p.UtilityDamagePerRound = float64(p.UtilityDamage) / rounds
			p.UtilityKillsPer100Rounds = float64(p.UtilityKills) * 100 / rounds
			p.FlashesThrownPerRound = float64(p.FlashesThrown) / rounds
			p.FlashAssistsPerRound = float64(p.FlashAssists) / rounds
		}

		// Stats that depend on RoundsWon
		if p.RoundsWon > 0 {
			p.KillsPerRoundWin = float64(p.KillsInWonRounds) / float64(p.RoundsWon)
			p.DamagePerRoundWin = float64(p.DamageInWonRounds) / float64(p.RoundsWon)
		}

		// Stats that depend on RoundsLost
		if p.RoundsLost > 0 {
			p.SavesPerRoundLoss = float64(p.SavesOnLoss) / float64(p.RoundsLost)
		}

		// Stats that depend on Deaths
		if p.Deaths > 0 {
			p.TradedDeathsPct = float64(p.TradedDeaths) / float64(p.Deaths)
		}

		// Stats that depend on OpeningDeaths
		if p.OpeningDeaths > 0 {
			p.OpeningDeathsTradedPct = float64(p.OpeningDeathsTraded) / float64(p.OpeningDeaths)
		}

		// Stats that depend on Kills
		if p.Kills > 0 {
			p.TradeKillsPct = float64(p.TradeKills) / float64(p.Kills)
			p.AssistedKillsPct = float64(p.AssistedKills) / float64(p.Kills)
			p.DamagePerKill = float64(p.Damage) / float64(p.Kills)
			p.AWPKillsPct = float64(p.AWPKills) / float64(p.Kills)
			p.LowBuyKillsPct = float64(p.LowBuyKills) / float64(p.Kills)
			p.DisadvantagedBuyKillsPct = float64(p.DisadvantagedBuyKills) / float64(p.Kills)
		}

		// Stats that depend on OpeningAttempts
		if p.OpeningAttempts > 0 {
			p.OpeningSuccessPct = float64(p.OpeningSuccesses) / float64(p.OpeningAttempts)
		}

		// Stats that depend on OpeningKills
		if p.OpeningKills > 0 {
			p.WinPctAfterOpeningKill = float64(p.RoundsWonAfterOpening) / float64(p.OpeningKills)
		}

		// Stats that depend on Clutch1v1Attempts
		if p.Clutch1v1Attempts > 0 {
			p.Clutch1v1WinPct = float64(p.Clutch1v1Wins) / float64(p.Clutch1v1Attempts)
		}

		// Populate MultiKills struct from raw array
		p.MultiKills.OneK = p.MultiKillsRaw[1]
		p.MultiKills.TwoK = p.MultiKillsRaw[2]
		p.MultiKills.ThreeK = p.MultiKillsRaw[3]
		p.MultiKills.FourK = p.MultiKillsRaw[4]
		p.MultiKills.FiveK = p.MultiKillsRaw[5]

		p.FinalRating = rating.ComputeFinalRating(p)

		// Calculate per-side Eco Ratings
		if p.TRoundsPlayed > 0 {
			p.TEcoRating = rating.ComputeSideRating(
				p.TRoundsPlayed, p.TKills, p.TDeaths, p.TDamage, p.TEcoKillValue,
				p.TRoundSwing, p.TKAST, p.TMultiKills, p.TClutchRounds, p.TClutchWins)
		}
		if p.CTRoundsPlayed > 0 {
			p.CTEcoRating = rating.ComputeSideRating(
				p.CTRoundsPlayed, p.CTKills, p.CTDeaths, p.CTDamage, p.CTEcoKillValue,
				p.CTRoundSwing, p.CTKAST, p.CTMultiKills, p.CTClutchRounds, p.CTClutchWins)
		}

		// Log player summary
		d.logger.LogPlayerSummary(p.Name, p.Kills, p.Deaths, p.Damage, p.EcoKillValue, p.EcoDeathValue, p.FinalRating)
	}
}

func (d *DemoParser) ExportJSON(path string) error {
	return output.Export(d.state.Players, path)
}

// GetPlayers returns the parsed player stats map
func (d *DemoParser) GetPlayers() map[uint64]*model.PlayerStats {
	return d.state.Players
}

// GetMapName returns the map name captured from the demo
func (d *DemoParser) GetMapName() string {
	return d.state.MapName
}

// GetLogs returns the buffered log output from parsing
func (d *DemoParser) GetLogs() string {
	return d.logger.GetOutput()
}
