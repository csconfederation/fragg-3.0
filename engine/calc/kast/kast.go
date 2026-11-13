// Package kast is the KAST family: the rounds a player got a kill, an assist, a trade of their
// death, or survived (design note section 3). CSC's rule counts enemy kills; eco's, kept as
// EcoKastRounds until D31 unifies the gates, counts EcoKill ones and only the round's humans.
package kast

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc writes each player's KAST.
func Calc(r *round.Round, out *row.Round) {
	csc := map[uint64]bool{}  // the row got a kill, assist or trade
	died := map[uint64]bool{} // the row has a counted death
	ecoK := map[uint64]bool{} // eco rows, by the same test; a row exists from the open or the join
	for id, m := range r.OpenMembers {
		if !m.IsBot {
			ecoK[id] = false
		}
	}
	ecoDied := map[uint64]bool{}

	for _, e := range r.Events[:r.Commit] {
		if e.Dropped {
			continue
		}
		if j := e.Join; j != nil && !j.IsBot {
			if _, ok := ecoK[j.Steam64]; !ok {
				ecoK[j.Steam64] = false
			}
		}
		f, ok := e.Fact.(fact.Kill)
		if !ok || !e.Expected {
			continue
		}
		if e.Victim {
			died[f.Victim.Steam64] = true
			ecoDied[f.Victim.Steam64] = true
		}
		if e.Enemy {
			csc[f.Killer.Steam64] = true
			for _, a := range e.Avenged {
				csc[a.Steam] = true
			}
		}
		if e.Assist {
			csc[f.Assister.Steam64] = true
			if _, ok := ecoK[f.Assister.Steam64]; ok {
				ecoK[f.Assister.Steam64] = true
			}
		}
		if e.FlashAssisted { // D35: a flash assist is an assist
			csc[e.FlashAssister] = true
			if _, ok := ecoK[e.FlashAssister]; ok {
				ecoK[e.FlashAssister] = true
			}
		}
		if e.EcoKill {
			if _, ok := ecoK[f.Killer.Steam64]; ok {
				ecoK[f.Killer.Steam64] = true
			}
			for _, a := range e.Avenged {
				if _, ok := ecoK[a.Steam]; ok {
					ecoK[a.Steam] = true
				}
			}
		}
	}

	for id, p := range out.Players {
		if csc[id] || !died[id] {
			p.KastRounds = 1
		}
		if got, human := ecoK[id]; human && (got || !ecoDied[id]) {
			p.EcoKastRounds = 1
		}
	}
}
