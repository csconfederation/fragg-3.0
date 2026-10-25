// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file contains the main DemoParser struct and its methods for parsing
// demo files, computing player statistics, and calculating ratings.
package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
	"github.com/csconfederation/fragg-3.0/rating/probability"
	"github.com/csconfederation/fragg-3.0/rating/swing"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

// DemoParser wraps the demoinfocs parser and manages match state and logging.
// It processes CS2 demo files and extracts comprehensive player statistics.
type DemoParser struct {
	parser            *demoinfocs.Parser
	state             *MatchState
	logger            ParserLogger
	collector         *probability.DataCollector
	kdprModifier      bool
	externalLifecycle bool
	lastWinner        int
}

// NewDemoParser creates a new DemoParser with logging disabled.
func NewDemoParser(r io.Reader) *DemoParser {
	return NewDemoParserWithOptions(r, false, false)
}

// NewDemoParserWithLogging creates a new DemoParser with configurable logging.
// The parser is initialized with event handlers but Parse() must be called to process.
func NewDemoParserWithLogging(r io.Reader, enableLogging bool) *DemoParser {
	return NewDemoParserWithOptions(r, enableLogging, false)
}

// NewDemoParserWithOptions creates a new DemoParser with configurable logging and KPR/DPR modifier.
func NewDemoParserWithOptions(r io.Reader, enableLogging bool, kdprModifier bool) *DemoParser {
	return NewDemoParserWithSwingConfig(r, enableLogging, kdprModifier, swing.DefaultConfig())
}

// NewDemoParserWithSwingConfig creates a DemoParser with full swing configuration.
func NewDemoParserWithSwingConfig(r io.Reader, enableLogging bool, kdprModifier bool, swingCfg swing.Config) *DemoParser {
	p := demoinfocs.NewParser(r,
		option.IgnorePacketEntitiesPanic(true),
		option.UserCmdParsing(option.UserCmdParsingDisabled))
	state := NewMatchStateWithConfig(swingCfg)

	dp := &DemoParser{
		parser:       p,
		state:        state,
		logger:       NewLogger(enableLogging),
		collector:    probability.NewDataCollector(),
		kdprModifier: kdprModifier,
	}

	dp.registerHandlers()
	return dp
}

// GetCollector returns the probability data collector for merging in cumulative mode.
func (d *DemoParser) GetCollector() *probability.DataCollector {
	return d.collector
}

// currentTime returns the current game time in seconds based on the current frame.
func (d *DemoParser) currentTime() float64 {
	return float64(d.parser.CurrentFrame()) / float64(rating.TickRate)
}

// timeInRound returns the elapsed time since the round started.
func (d *DemoParser) timeInRound() float64 {
	return d.currentTime() - d.state.RoundStartTime
}

// SetLogging enables or disables detailed parsing logs.
func (d *DemoParser) SetLogging(enabled bool) {
	d.logger.SetEnabled(enabled)
}

// SetPlayerFilter limits logging to events involving the specified players.
func (d *DemoParser) SetPlayerFilter(players []string) {
	d.logger.SetPlayerFilter(players)
}

// AddPlayerFilter adds a player to the logging filter.
func (d *DemoParser) AddPlayerFilter(player string) {
	d.logger.AddPlayerFilter(player)
}

// ClearPlayerFilter removes all player filters, logging all events.
func (d *DemoParser) ClearPlayerFilter() {
	d.logger.ClearPlayerFilter()
}

// Parse processes the entire demo file and computes all player statistics.
// After parsing, it calculates derived metrics (ADR, KPR, ratings, etc.)
// and the final eco-rating for each player.
// Returns an error if parsing fails. Truncated demos (ErrUnexpectedEndOfDemo)
// are handled gracefully — stats collected up to the truncation point are kept.
func (d *DemoParser) Parse(ctx context.Context) error {
	if err := d.parser.ParseToEnd(ctx); err != nil {
		if errors.Is(err, demoinfocs.ErrUnexpectedEndOfDemo) {
			log.Printf("Warning: demo truncated (unexpected EOF), using partial data")
		} else {
			return fmt.Errorf("failed to parse demo: %w", err)
		}
	}
	d.computeDerivedStats()
	return nil
}

// computeDerivedStats calculates all derived metrics for each player after parsing.
func (d *DemoParser) computeDerivedStats() {
	for _, p := range d.state.Players {
		eco.Derive(p, d.kdprModifier)
		d.logger.LogPlayerSummary(p.Name, p.Kills, p.Deaths, p.Damage, p.EcoKillValue, p.EcoDeathValue, p.FinalRating)
	}
}

// GetPlayers returns the map of all player statistics keyed by Steam ID.
func (d *DemoParser) GetPlayers() map[uint64]*model.PlayerStats {
	return d.state.Players
}

// GetRoundsCounted returns the number of rounds that were aggregated into the
// player stats after CSC round dedup (or after standalone RoundEnd folds).
func (d *DemoParser) GetRoundsCounted() int {
	return d.state.RoundsCounted
}

// GetMapName returns the name of the map played (e.g., "de_dust2").
func (d *DemoParser) GetMapName() string {
	return d.state.MapName
}

// GetLogs returns all captured log output from parsing.
func (d *DemoParser) GetLogs() string {
	return d.logger.GetOutput()
}
