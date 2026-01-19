// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package output provides functionality for aggregating player statistics.
// This file defines interfaces for the output components to enable
// dependency injection and easier unit testing.
package output

import (
	"eco-rating/model"
)

// StatsAggregatorInterface defines the contract for aggregating player statistics.
// Implementations can be mocked for testing purposes.
type StatsAggregatorInterface interface {
	// AddGame incorporates statistics from a single game into the aggregator.
	AddGame(players map[uint64]*model.PlayerStats, mapName string, tier string)

	// Finalize computes all derived statistics from accumulated raw values.
	Finalize()

	// GetResults returns the map of all aggregated player statistics.
	GetResults() map[string]*AggregatedStats
}

// Ensure Aggregator implements StatsAggregatorInterface at compile time.
var _ StatsAggregatorInterface = (*Aggregator)(nil)
