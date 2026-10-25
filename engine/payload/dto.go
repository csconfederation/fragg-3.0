package payload

import (
	"maps"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/match"
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
	Teams         []Team `json:"teams"`
	// Players holds each player's whole-match row (side 4) and side rows (2 T, 3 CT).
	Players []*cscstats.PlayerStats `json:"players,omitempty"`
	Rounds  []Round                 `json:"rounds,omitempty"`
}

// Team is a team's name, final score, and match stats (absent when score-only).
type Team struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	*cscstats.TeamStats
}

// Round is a committed round and its players' round rows.
type Round struct {
	*cscstats.Round
	Players []*cscstats.PlayerStats `json:"players"`
}

const schemaVersion = 2

func build(g *cscstats.Game, res *match.Result) *Payload {
	p := &Payload{SchemaVersion: schemaVersion, FullStatsOK: res.Outcome == match.Ended,
		MapName: res.Map, TickRate: g.TickRate, TotalRounds: res.TotalRounds}
	for _, name := range slices.Sorted(maps.Keys(res.Scores)) {
		t := Team{Name: name, Score: res.Scores[name]}
		if p.FullStatsOK {
			t.TeamStats = g.TotalTeamStats[name]
		}
		p.Teams = append(p.Teams, t)
	}
	if !p.FullStatsOK {
		return p
	}
	for _, steam := range slices.Sorted(maps.Keys(g.TotalPlayerStats)) {
		p.Players = append(p.Players, g.TotalPlayerStats[steam])
		for _, side := range []map[uint64]*cscstats.PlayerStats{g.TPlayerStats, g.CtPlayerStats} {
			if row := side[steam]; row != nil {
				p.Players = append(p.Players, row)
			}
		}
	}
	for _, r := range g.Rounds {
		rr := Round{Round: r}
		for _, steam := range slices.Sorted(maps.Keys(r.PlayerStats)) {
			rr.Players = append(rr.Players, r.PlayerStats[steam])
		}
		p.Rounds = append(p.Rounds, rr)
	}
	return p
}
