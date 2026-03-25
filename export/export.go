// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package export defines interfaces and implementations for exporting player
// statistics to various formats (CSV, JSON, etc.). It supports both single-game
// exports and aggregated multi-game statistics.
package export

import (
	"github.com/ethsmith/eco-rating/model"
	"github.com/ethsmith/eco-rating/output"
)

// ExportOption defines the interface for exporting player statistics.
// Implementations can export to different formats (CSV, JSON, database, etc.).
type ExportOption interface {
	// Export writes single-game player statistics to the output destination.
	Export(players map[uint64]*model.PlayerStats) error

	// ExportAggregated writes aggregated multi-game statistics to the output destination.
	ExportAggregated(players map[string]*output.AggregatedStats) error
}
