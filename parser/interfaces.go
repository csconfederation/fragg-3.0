// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file defines interfaces for the parser components to enable
// dependency injection and easier unit testing.
package parser

import (
	"eco-rating/model"
)

// DemoParserInterface defines the contract for demo parsing operations.
// Implementations can be mocked for testing purposes.
type DemoParserInterface interface {
	// Parse processes the demo file and computes all statistics.
	// Returns an error if parsing fails.
	Parse() error

	// GetPlayers returns the map of all player statistics keyed by Steam ID.
	GetPlayers() map[uint64]*model.PlayerStats

	// GetMapName returns the name of the map played.
	GetMapName() string

	// GetLogs returns all captured log output from parsing.
	GetLogs() string

	// SetLogging enables or disables detailed parsing logs.
	SetLogging(enabled bool)

	// SetPlayerFilter limits logging to events involving the specified players.
	SetPlayerFilter(players []string)
}

// MatchStateInterface defines the contract for match state management.
// This allows for easier testing of state-dependent logic.
type MatchStateInterface interface {
	// EnsurePlayer returns the PlayerStats for a player, creating it if needed.
	EnsurePlayer(steamID uint64, name string) *model.PlayerStats

	// EnsureRound returns the RoundStats for a player in the current round.
	EnsureRound(steamID uint64) *model.RoundStats

	// GetPlayers returns all tracked players.
	GetPlayers() map[uint64]*model.PlayerStats

	// ResetRound resets the round state for a new round.
	ResetRound()
}

// LoggerInterface defines the contract for parsing event logging.
// Implementations can be mocked to verify logging behavior in tests.
type LoggerInterface interface {
	// SetEnabled enables or disables logging.
	SetEnabled(enabled bool)

	// SetPlayerFilter sets the list of player names to include in logging.
	SetPlayerFilter(players []string)

	// LogKill logs a kill event.
	LogKill(round int, killer, victim string, killerEquip, victimEquip int, killValue float64)

	// LogDeath logs a death event.
	LogDeath(round int, victim, killer string, victimEquip, killerEquip int, deathPenalty float64)

	// LogRoundStart logs the beginning of a new round.
	LogRoundStart(round int)

	// LogRoundEnd logs the end of a round.
	LogRoundEnd(round int)

	// LogTrade logs a trade kill event.
	LogTrade(round int, trader, tradedPlayer, originalKiller string)

	// LogOpeningKill logs the first kill of a round.
	LogOpeningKill(round int, killer, victim string)

	// LogMultiKill logs a multi-kill round.
	LogMultiKill(round int, player string, kills int)

	// LogPlayerSummary logs end-of-game statistics for a player.
	LogPlayerSummary(name string, kills, deaths, damage int, ecoKillValue, ecoDeathValue, finalRating float64)

	// LogBombPlant logs a bomb plant event.
	LogBombPlant(round int, planter string)

	// LogBombDefuse logs a bomb defuse event.
	LogBombDefuse(round int, defuser string)

	// LogKnifeRound logs detection of a knife round.
	LogKnifeRound()

	// GetOutput returns all captured log output as a string.
	GetOutput() string
}

// Ensure DemoParser implements DemoParserInterface at compile time.
var _ DemoParserInterface = (*DemoParser)(nil)

// Ensure Logger implements LoggerInterface at compile time.
var _ LoggerInterface = (*Logger)(nil)
