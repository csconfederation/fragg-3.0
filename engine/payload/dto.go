package payload

import (
	"maps"
	"slices"
	"strconv"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Payload is what the worker sends Stats (D7, amended 2026-10-25): the stats Stats stores, named
// after its columns, and nothing else. Stats imports it field by field. Steam64s are decimal
// strings.
type Payload struct {
	SchemaVersion int    `json:"schemaVersion"` // 2; the oracle's payload has no version
	FullStatsOK   bool   `json:"fullStatsOK"`   // false: a partial demo, teams and score only (D9)
	MapName       string `json:"mapName"`
	TickRate      int    `json:"tickRate"`
	TotalRounds   int    `json:"totalRounds"`
	// HeaderMatchID is the match ID in the demo's server name, or null (S11-S14 demos, scrim and
	// preseason servers). Stats logs it when it disagrees with the request's ID, which wins (D11, D19).
	HeaderMatchID *string `json:"headerMatchID"`
	Teams         []Team  `json:"teams"`
	// Players holds each player's whole-match row (side 4) and side rows (2 T, 3 CT).
	Players []*row.Player `json:"players,omitempty"`
	Rounds  []Round       `json:"rounds,omitempty"`
}

// Team is a team's name, final score, and match stats (absent when score-only).
type Team struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	*row.Team
}

// Round is a committed round and its players' round rows.
type Round struct {
	*row.Round
	Players []*row.Player `json:"players"`
}

const schemaVersion = 2

func build(g *cscstats.Game, res *match.Result) *Payload {
	p := &Payload{SchemaVersion: schemaVersion, FullStatsOK: res.Outcome == match.Ended,
		MapName: res.Map, TickRate: g.TickRate, TotalRounds: res.TotalRounds}
	if res.HeaderMatchID != 0 {
		id := strconv.Itoa(res.HeaderMatchID)
		p.HeaderMatchID = &id
	}
	for _, name := range slices.Sorted(maps.Keys(res.Scores)) {
		t := Team{Name: name, Score: res.Scores[name]}
		if p.FullStatsOK {
			t.Team = teamRow(g.TotalTeamStats[name])
		}
		p.Teams = append(p.Teams, t)
	}
	if !p.FullStatsOK {
		return p
	}
	for _, steam := range slices.Sorted(maps.Keys(g.TotalPlayerStats)) {
		p.Players = append(p.Players, playerRow(g.TotalPlayerStats[steam]))
		for _, side := range []map[uint64]*cscstats.PlayerStats{g.TPlayerStats, g.CtPlayerStats} {
			if src := side[steam]; src != nil {
				p.Players = append(p.Players, playerRow(src))
			}
		}
	}
	for _, r := range g.Rounds {
		rr := Round{Round: RoundRow(r)}
		for _, steam := range slices.Sorted(maps.Keys(rr.Round.Players)) {
			rr.Players = append(rr.Players, rr.Round.Players[steam])
		}
		p.Rounds = append(p.Rounds, rr)
	}
	return p
}

// The rows come from the collectors and the fold (engine/cscstats) as they always have; the payload
// holds them as row types with the constants applied (constants.go).

// playerRow is a match or side row; a round's row is the same with the round's constants.
func playerRow(src *cscstats.PlayerStats) *row.Player {
	p := &row.Player{}
	copyRow(p, src)
	switch p.Side {
	case int(fact.SideT):
		setConstants(p, scopeSideT)
	case int(fact.SideCT):
		setConstants(p, scopeSideCT)
	}
	if p.SteamID == "0" {
		setConstants(p, scopeBot)
	}
	return p
}

// RoundRow is a committed round's row as the payload holds it, constants applied. cmd/calcdiff compares
// the calculators with it.
func RoundRow(src *cscstats.Round) *row.Round {
	r := &row.Round{Players: map[uint64]*row.Player{}}
	copyRow(r, src)
	setConstants(r, scopeRound)
	for steam, ps := range src.PlayerStats {
		p := &row.Player{}
		copyRow(p, ps)
		setConstants(p, scopeRoundPlayer)
		if p.SteamID == "0" {
			setConstants(p, scopeBot)
		}
		r.Players[steam] = p
	}
	return r
}

// RoundTeam is a team's row in a round, without the match row's constants. cmd/calcdiff compares the
// calculators with it.
func RoundTeam(src *cscstats.TeamStats) *row.Team {
	t := &row.Team{}
	copyRow(t, src)
	return t
}

// teamRow is nil for a team without stats.
func teamRow(src *cscstats.TeamStats) *row.Team {
	if src == nil {
		return nil
	}
	t := &row.Team{}
	copyRow(t, src)
	setConstants(t, scopeTeamMatch)
	return t
}
