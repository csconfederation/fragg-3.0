// Package survival is the survival family: when and in what order a player died, who saved.
package survival

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	"maps"
)

// earlyDeath is eco's cutoff, in seconds into the round.
const earlyDeath = 30.0

// Calc writes each player's row.Survival and each team's row.TeamSurvival.
func Calc(r *round.Round, out *row.Round) {
	died := map[uint64]bool{}
	joined := map[fact.Side]int{} // players who joined each side mid-round (D33)
	members := map[uint64]match.Member{}
	maps.Copy(members, r.OpenMembers)
	for _, e := range r.Events[:r.Commit] {
		if j := e.Join; j != nil {
			members[j.Steam64] = *j
			joined[j.Side]++
		}
		k, ok := e.Fact.(fact.Kill)
		if !ok || e.Dropped || !e.Victim {
			continue
		}
		p := out.Players[k.Victim.Steam64]
		if p == nil {
			continue
		}
		died[k.Victim.Steam64] = true
		p.DeathTick = k.Tick
		p.TicksAlive += k.Tick - r.StartTick
		switch k.Victim.Side {
		case fact.SideT:
			p.DeathPlacement = float64(r.InitT + joined[fact.SideT] - e.After.TAlive)
		case fact.SideCT:
			p.DeathPlacement = float64(r.InitCT + joined[fact.SideCT] - e.After.CTAlive)
		}
		if id := k.Victim.Steam64; id != 0 { // eco has no bot rows
			p.EcoDeathTime = seconds(r, k.Tick)
		}
	}
	win, won := r.WinAt(r.Commit)
	left := r.Disconnected()
	for id, p := range out.Players {
		if !died[id] && left[id] { // D44: a disconnect while alive is not surviving
			p.LeftAlive = 1
		} else if !died[id] {
			if m := members[id]; m.TeamID != int(win.Side) {
				p.Saves = 1
				if t := out.Teams[r.SideOf(m.Clan)]; t != nil {
					t.Saves = 1
				}
			}
		}
		if id == 0 {
			continue
		}
		if died[id] {
			p.EcoTimeAlive = p.EcoDeathTime
			if p.EcoDeathTime < earlyDeath {
				p.EarlyDeaths = 1
			}
		} else if won {
			p.EcoTimeAlive = seconds(r, win.Tick)
		}
	}
}

func seconds(r *round.Round, tick int) float64 {
	return float64(tick-r.StartTick) / float64(fact.TickRate)
}
