// Package utility is the utility family: flashes landed, flash assists and grenades thrown.
package utility

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc writes each player's row.Utility.
func Calc(r *round.Round, out *row.Round) {
	in := map[uint64]bool{} // the players with a row so far: one who joins mid-round has none before
	for id := range r.OpenMembers {
		in[id] = true
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Dropped {
			continue
		}
		if e.Join != nil {
			in[e.Join.Steam64] = true
		}
		switch f := e.Fact.(type) {
		case fact.Flashed:
			if !e.Flash {
				continue
			}
			// Duration is exact: CSC multiplied and divided it by the tick rate (Q26).
			if p := out.Players[f.Attacker.Steam64]; p != nil {
				p.Ef++
				p.EnemyFlashTime += f.Duration
			}
			if p := out.Players[f.Player.Steam64]; p != nil {
				p.MostRecentFlashVal, p.MostRecentFlasher = e.Blind.Until, e.Blind.Flasher
			}
		case fact.Kill:
			// A flash assist: the game marks it, or the victim was blind from the assister's teammate.
			if !e.FlashAssisted {
				continue
			}
			if p := out.Players[e.FlashAssister]; p != nil {
				p.FAss++
			}
		case fact.GrenadeThrow:
			if f.Weapon == nil || f.Thrower == nil {
				continue
			}
			p := out.Players[f.Thrower.Steam64]
			if p == nil || !in[f.Thrower.Steam64] {
				continue
			}
			switch *f.Weapon {
			case fact.EqHE:
				p.NadesThrown++
			case fact.EqSmoke:
				p.SmokeThrown++
			case fact.EqFlash:
				p.FlashThrown++
			case fact.EqMolotov, fact.EqIncendiary:
				p.FiresThrown++
			}
		}
	}
}
