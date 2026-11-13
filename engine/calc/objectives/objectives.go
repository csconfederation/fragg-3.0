// Package objectives is the objectives family: kill points, exit frags, the bomb's round fields and
// the impact, win and round-win-share points built from them (design note section 3). Calc is tier 1,
// reading only the round; Finish is tier 2, reading the combat rows and the clutch slots, and runs
// last. The kill value is CSC's (Q7, Q19), exactly in its float order: one ULP would show in mip,
// which compares with ==.
package objectives

import (
	"maps"
	"math"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

var multiKillBonus = [...]float64{0, 0, 0.3, 0.7, 1.2, 2}
var clutchBonus = [...]float64{0, 0.2, 0.6, 1.2, 2, 3}

// Calc writes each player's killPoints and exitFrags, the bonus part of impactPoints (+0.15 per
// assisted kill, +0.5 per defuse or explosion, in fact order), and the round's bomb fields.
func Calc(r *round.Round, out *row.Round) {
	ecos := map[uint64]bool{} // eco has rows for the round's humans, from the open or their join
	for id, m := range r.OpenMembers {
		if !m.IsBot {
			ecos[id] = true
		}
	}
	wins := r.WinList()
	won := 0 // the side of the latest Win in effect: none before the first (Q20)
	next := 0

	for i, e := range r.Events[:r.Commit] {
		for next < len(wins) && wins[next].At <= i {
			won = int(wins[next].Side)
			next++
		}
		if e.Dropped {
			continue
		}
		if j := e.Join; j != nil && !j.IsBot {
			ecos[j.Steam64] = true // a kill before the join was a scratch row's
		}
		switch f := e.Fact.(type) {
		case fact.Kill:
			if !e.Expected {
				continue
			}
			if e.Enemy {
				if p := out.Players[f.Killer.Steam64]; p != nil {
					var assister *row.Player
					if e.Assist {
						assister = out.Players[f.Assister.Steam64]
					}
					p.KillPoints += killValue(e, f, won, assister)
				}
			}
			if e.EcoKill && e.Before.Phase == round.Decided && ecos[f.Killer.Steam64] {
				if p := out.Players[f.Killer.Steam64]; p != nil {
					p.ExitFrags++
				}
			}
		case fact.BombPlanted:
			if e.Before.Phase != round.Decided {
				out.Planter, out.BombStartTick = f.Player.Steam64, f.Tick
			}
		case fact.BombDefused:
			if e.Before.Phase != round.Decided {
				out.EndDueToBombEvent, out.Defuser = true, f.Player.Steam64
				if p := out.Players[f.Player.Steam64]; p != nil {
					p.ImpactPoints += 0.5
				}
			}
		case fact.BombExplode:
			if e.Before.Phase != round.Decided {
				out.EndDueToBombEvent = true
				if p := out.Players[out.Planter]; out.Planter != 0 && p != nil {
					p.ImpactPoints += 0.5
				}
			}
		}
	}
}

// killValue is an enemy kill's points, and gives its assister's +0.15.
func killValue(e round.Event, f fact.Kill, won int, assister *row.Player) float64 {
	killer := f.Killer.Side
	value, multiplier := 1.0, 1.0
	switch e.Before.Phase {
	case round.PrePlant:
		if killer == fact.SideT {
			value = 1.2
		} else if killer == fact.SideCT {
			value = 1
		}
	case round.PostPlant:
		if killer == fact.SideT {
			value = 1
		} else if killer == fact.SideCT {
			value = 1.2
		}
	case round.Decided:
		switch won {
		case 2:
			if killer == fact.SideT {
				value = 0.8
			}
			if killer == fact.SideCT {
				value = 0.6
			}
		case 3:
			if killer == fact.SideT {
				value = 0.5
			}
			if killer == fact.SideCT {
				if e.Before.TMoney {
					value = 0.6
				} else {
					value = 0.8
				}
			}
		}
	}
	if e.Before.Opening {
		if killer == fact.SideT {
			if e.Before.Phase == round.PrePlant {
				multiplier += 0.8
			} else {
				multiplier += 0.3
			}
		} else if killer == fact.SideCT {
			multiplier += 0.5
		}
	} else if len(e.Avenged) > 0 {
		multiplier += 0.3
	}
	if e.Assist && (f.AssistedFlash || e.FlashAssisted) {
		multiplier += 0.2
	}
	if e.Assist {
		value -= 0.15
		if assister != nil {
			assister.ImpactPoints += 0.15
		}
	}
	value *= multiplier

	ratio := float64(f.Victim.EquipValue) / float64(f.Killer.EquipValue)
	mod := 1.0
	if ratio > 4 {
		mod += 0.25
	} else if ratio > 2 {
		mod += 0.14
	} else if ratio < 0.25 {
		mod -= 0.25
	} else if ratio < 0.5 {
		mod -= 0.14
	}
	return value * mod
}

// Finish completes the points from the other families' rows: the clutch bonus, kill points, damage
// and the multi-kill bonus onto impactPoints, then win points, round wins and losses, mip and the
// round win share, and the teams' points and the winning team's damage. It runs after every tier 1
// calculator.
func Finish(r *round.Round, out *row.Round) {
	win, _ := r.WinAt(r.Commit)
	members := maps.Clone(r.OpenMembers)
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil {
			members[e.Join.Steam64] = *e.Join
		}
	}

	// The clutch bonus: the winning side's slot, T first (clutches.Calc's guard).
	for _, s := range []struct {
		side fact.Side
		c    round.Clutch
	}{{fact.SideT, r.End.TClutch}, {fact.SideCT, r.End.CTClutch}} {
		if win.Side != s.side || s.c.Steam == 0 {
			continue
		}
		if s.c.Size < 0 || s.c.Size >= len(clutchBonus) {
			break
		}
		if p := out.Players[s.c.Steam]; p != nil {
			p.ImpactPoints += clutchBonus[s.c.Size]
		}
		break
	}

	ids := slices.Sorted(maps.Keys(out.Players))
	highest := 0.0
	for _, id := range ids {
		p := out.Players[id]
		p.ImpactPoints += p.KillPoints
		p.ImpactPoints += float64(p.Damage) / float64(250)
		// Bots share one row, so it can hold more than five kills; they get the five-kill bonus.
		p.ImpactPoints += multiKillBonus[min(p.Kills, len(multiKillBonus)-1)]
		if p.ImpactPoints > highest {
			highest = p.ImpactPoints
		}
		if members[id].TeamID == int(win.Side) {
			p.WinPoints = p.ImpactPoints
			p.RF = 1
		} else {
			p.RA = 1
		}
	}
	mip := 0
	for _, id := range ids {
		if out.Players[id].ImpactPoints == highest {
			mip++
		}
	}
	for _, id := range ids {
		if p := out.Players[id]; p.ImpactPoints == highest {
			p.Mip = 1.0 / float64(mip)
		}
	}

	// The teams' points and the winners' damage, in Steam64 order.
	for _, id := range ids {
		p, m := out.Players[id], members[id]
		if p.RF == 1 {
			out.WinTeamDmg += p.Damage
		}
		t := out.Teams[r.SideOf(m.Clan)]
		if t == nil {
			continue
		}
		switch m.Side {
		case fact.SideT:
			t.TWinPoints += p.WinPoints
			t.TImpactPoints += p.ImpactPoints
		case fact.SideCT:
			t.CtWinPoints += p.WinPoints
			t.CtImpactPoints += p.ImpactPoints
		}
		t.WinPoints += p.WinPoints
		t.ImpactPoints += p.ImpactPoints
	}
	for _, id := range ids {
		p, m := out.Players[id], members[id]
		if p.RF == 0 {
			continue
		}
		share := 100.0
		if out.EndDueToBombEvent {
			share = 70
		}
		p.Rws = share * (float64(p.Damage) / float64(out.WinTeamDmg))
		if out.EndDueToBombEvent && (m.Side == fact.SideT && out.Planter == id || m.Side == fact.SideCT && out.Defuser == id) {
			p.Rws += 30
		}
		if math.IsNaN(p.Rws) {
			p.Rws = 0
		}
	}
}
