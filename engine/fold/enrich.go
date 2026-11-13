package fold

import (
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// Enrich is enrich on a round that may have been enriched: cmd/calcdiff reads the collectors' rows
// the way the fold leaves them. PR 18 deletes it with the collectors.
func Enrich(r *cscstats.Round) {
	r.WinTeamDmg = 0
	for _, t := range r.TeamStats {
		t.WinPoints, t.ImpactPoints, t.TWinPoints, t.CtWinPoints, t.TImpactPoints, t.CtImpactPoints = 0, 0, 0, 0, 0, 0
	}
	enrich(r)
}

// enrich computes a round's team win and impact points, its winning team's damage, and each
// winner's round win share; Stats stores the last two on the round rows.
func enrich(r *cscstats.Round) {
	for _, steam := range slices.Sorted(maps.Keys(r.PlayerStats)) {
		p := r.PlayerStats[steam]
		if p.RF == 1 {
			r.WinTeamDmg += p.Damage
		}
		team := r.TeamStats[p.TeamClanName]
		switch p.Side {
		case int(fact.SideT):
			team.TWinPoints += p.WinPoints
			team.TImpactPoints += p.ImpactPoints
		case int(fact.SideCT):
			team.CtWinPoints += p.WinPoints
			team.CtImpactPoints += p.ImpactPoints
		}
		team.WinPoints += p.WinPoints
		team.ImpactPoints += p.ImpactPoints
	}
	for _, p := range r.PlayerStats {
		if p.RF == 0 {
			continue
		}
		share := 100.0
		if r.EndDueToBombEvent {
			share = 70
		}
		p.Rws = share * (float64(p.Damage) / float64(r.WinTeamDmg))
		steam, _ := strconv.ParseUint(p.SteamID, 10, 64)
		if r.EndDueToBombEvent && (p.Side == int(fact.SideT) && r.Planter == steam || p.Side == int(fact.SideCT) && r.Defuser == steam) {
			p.Rws += 30
		}
		if math.IsNaN(p.Rws) {
			p.Rws = 0
		}
	}
}
