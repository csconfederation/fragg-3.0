// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file defines the MatchState struct which tracks the current state of a
// match during parsing, including player stats, round stats, and trade detection.
package parser

import (
	"eco-rating/model"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// MatchState holds all state information during demo parsing.
// It tracks players, current round stats, and various flags for game state.
type MatchState struct {
	Players        map[uint64]*model.PlayerStats
	Round          map[uint64]*model.RoundStats
	TradeDetector  *TradeDetector
	RoundHasKill   bool
	MatchStarted   bool
	IsKnifeRound   bool
	IsPistolRound  bool
	RoundNumber    int
	MapName        string
	RoundStartTime float64
	CurrentSide    string
	TeamScore      int
	EnemyScore     int
	RoundDecided   bool
	RoundDecidedAt float64
}

// NewMatchState creates a new MatchState with initialized maps.
func NewMatchState() *MatchState {
	return &MatchState{
		Players:       make(map[uint64]*model.PlayerStats),
		Round:         make(map[uint64]*model.RoundStats),
		TradeDetector: NewTradeDetector(),
	}
}

// ensurePlayer returns the PlayerStats for a player, creating it if needed.
func (m *MatchState) ensurePlayer(p *common.Player) *model.PlayerStats {
	id := p.SteamID64
	if _, ok := m.Players[id]; !ok {
		m.Players[id] = &model.PlayerStats{
			SteamID: fmt.Sprintf("%d", id),
			Name:    p.Name,
		}
	}
	return m.Players[id]
}

// ensureRound returns the RoundStats for a player in the current round, creating it if needed.
func (m *MatchState) ensureRound(p *common.Player) *model.RoundStats {
	id := p.SteamID64
	if _, ok := m.Round[id]; !ok {
		m.Round[id] = &model.RoundStats{}
	}
	return m.Round[id]
}
