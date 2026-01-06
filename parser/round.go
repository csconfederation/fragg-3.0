package parser

import (
	"eco-rating/model"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type MatchState struct {
	Players      map[uint64]*model.PlayerStats
	Round        map[uint64]*model.RoundStats
	RoundHasKill bool
	MatchStarted bool
	IsKnifeRound bool
	RoundNumber  int
	// Track recent kills for trade detection: killer SteamID -> (victim SteamID, tick)
	RecentKills map[uint64]recentKill
	// Round context tracking
	RoundStartTime float64 // Round start time in seconds
	CurrentSide    string  // "T" or "CT" for current perspective

	// Score tracking
	TeamScore      int     // Score for the team we're tracking (first T team)
	EnemyScore     int     // Score for the opposing team
	RoundDecided   bool    // Round outcome is already determined
	RoundDecidedAt float64 // Time when round was decided (seconds from round start)

	// Track recent teammate deaths for trade speed calculation
	RecentTeamDeaths map[uint64]float64 // SteamID -> death time (seconds from round start)
}

type recentKill struct {
	VictimID   uint64
	VictimTeam common.Team
	Tick       int
}

func NewMatchState() *MatchState {
	return &MatchState{
		Players:          make(map[uint64]*model.PlayerStats),
		Round:            make(map[uint64]*model.RoundStats),
		RecentKills:      make(map[uint64]recentKill),
		RecentTeamDeaths: make(map[uint64]float64),
	}
}

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

func (m *MatchState) ensureRound(p *common.Player) *model.RoundStats {
	id := p.SteamID64
	if _, ok := m.Round[id]; !ok {
		m.Round[id] = &model.RoundStats{}
	}
	return m.Round[id]
}
