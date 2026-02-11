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
	"eco-rating/rating/probability"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// MatchState holds all state information during demo parsing.
// It tracks players, current round stats, and various flags for game state.
type MatchState struct {
	Players        map[uint64]*model.PlayerStats
	Round          map[uint64]*model.RoundStats
	TradeDetector  *TradeDetector
	SwingTracker   *SwingTracker
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
	BombPlanted    bool

	// Round start state for swing calculation
	RoundStartState *probability.RoundState
}

// NewMatchState creates a new MatchState with initialized maps.
func NewMatchState() *MatchState {
	return &MatchState{
		Players:       make(map[uint64]*model.PlayerStats),
		Round:         make(map[uint64]*model.RoundStats),
		TradeDetector: NewTradeDetector(),
		SwingTracker:  NewSwingTracker(),
	}
}

// ensurePlayer returns the PlayerStats for a player, creating it if needed.
func (m *MatchState) ensurePlayer(p *common.Player) *model.PlayerStats {
	id := p.SteamID64
	if _, ok := m.Players[id]; !ok {
		m.Players[id] = &model.PlayerStats{
			SteamID:  fmt.Sprintf("%d", id),
			Name:     p.Name,
			TeamName: playerClanName(p),
		}
	}
	ps := m.Players[id]
	// Update team name if it wasn't available on first encounter
	if ps.TeamName == "" {
		ps.TeamName = playerClanName(p)
	}
	return ps
}

// playerClanName extracts the clan/team name from a player's team state.
func playerClanName(p *common.Player) string {
	if p.TeamState != nil {
		return p.TeamState.ClanName()
	}
	return ""
}

// ensureRound returns the RoundStats for a player in the current round, creating it if needed.
func (m *MatchState) ensureRound(p *common.Player) *model.RoundStats {
	id := p.SteamID64
	if _, ok := m.Round[id]; !ok {
		m.Round[id] = &model.RoundStats{}
	}
	return m.Round[id]
}

// ShouldSkipEvent returns true if the current event should be skipped
// (knife round or match not started).
func (m *MatchState) ShouldSkipEvent() bool {
	return m.IsKnifeRound || !m.MatchStarted
}

// CountAlivePlayers counts alive human players on each team from the given participants.
// Bots are excluded since their data is not meaningful for competitive probability.
// Counts are capped at 5 per side as a safety net (CS2 is 5v5).
func (m *MatchState) CountAlivePlayers(participants []*common.Player) (tAlive, ctAlive int) {
	for _, p := range participants {
		if p.IsBot || !p.IsAlive() {
			continue
		}
		if p.Team == common.TeamTerrorists {
			tAlive++
		} else if p.Team == common.TeamCounterTerrorists {
			ctAlive++
		}
	}
	if tAlive > 5 {
		tAlive = 5
	}
	if ctAlive > 5 {
		ctAlive = 5
	}
	return tAlive, ctAlive
}
