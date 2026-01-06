package parser

import (
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
		}
		p.FinalRating = rating.ComputeFinalRating(p)

		// Log player summary
		d.logger.LogPlayerSummary(p.Name, p.Kills, p.Deaths, p.Damage, p.EcoKillValue, p.EcoDeathValue, p.FinalRating)
	}
}

func (d *DemoParser) ExportJSON(path string) error {
	return output.Export(d.state.Players, path)
}
