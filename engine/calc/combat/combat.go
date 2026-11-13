// Package combat is the combat family: damage, kills, assists, deaths and supporter credit
// (design note section 3). Its CSC-origin fields follow the CSC collector's gates; the eight
// eco-origin ones (assistedKills, knifeKills, utilityKills, perfectKills, pistolVsRifleKills,
// awpMultiKillRounds, awpDeaths, roundsWithAWPKill) follow eco's, until D31 unifies them.
package combat

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// eco is the eco collector's per-round counters behind the eco-origin fields.
type eco struct {
	kills, damage, awpKills, assists, utilKills, perfect int
	knife, pistolVsRifle, lostAWP                        bool
}

// Calc writes each player's Combat.
func Calc(r *round.Round, out *row.Round) {
	rows := map[uint64]*row.Combat{}
	ecos := map[uint64]*eco{} // eco has rows for the round's humans, from the open or their join; bots share Steam64 0
	for id, m := range r.OpenMembers {
		rows[id] = &row.Combat{}
		if !m.IsBot {
			ecos[id] = &eco{}
		}
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil && rows[e.Join.Steam64] == nil {
			rows[e.Join.Steam64] = &row.Combat{}
		}
	}
	damage := map[uint64]map[uint64]int{} // each victim's damage by attacker, over the round

	for _, e := range r.Events[:r.Commit] {
		if e.Dropped {
			continue
		}
		if j := e.Join; j != nil && !j.IsBot && ecos[j.Steam64] == nil {
			ecos[j.Steam64] = &eco{} // a kill before the join was a scratch row's
		}
		switch f := e.Fact.(type) {
		case fact.Hurt:
			if e.Taken {
				rows[f.Player.Steam64].DamageTaken += f.HealthDamageTaken
			}
			if e.Dealt && f.Attacker != nil {
				if x := ecos[f.Attacker.Steam64]; x != nil {
					x.damage += f.HealthDamageTaken
				}
			}
			if e.Dealt {
				hurt(rows[f.Attacker.Steam64], f)
				if damage[f.Player.Steam64] == nil {
					damage[f.Player.Steam64] = map[uint64]int{}
				}
				damage[f.Player.Steam64][f.Attacker.Steam64] += f.HealthDamageTaken
			}
		case fact.Kill:
			if e.Expected {
				kill(rows, ecos, damage, e, f)
			}
		}
	}

	for id, c := range rows {
		p := out.Players[id]
		if p == nil {
			continue
		}
		if x := ecos[id]; x != nil {
			c.EcoKills, c.EcoDamage, c.EcoAWPKills = x.kills, x.damage, x.awpKills
			c.AssistedKills = x.assists
			c.UtilityKills = x.utilKills
			c.PerfectKills = x.perfect
			c.KnifeKills = b2i(x.knife)
			c.PistolVsRifleKills = b2i(x.pistolVsRifle)
			c.RoundsWithAWPKill = b2i(x.awpKills > 0)
			c.AWPMultiKillRounds = b2i(x.awpKills >= 2)
			c.AWPDeaths = b2i(x.lostAWP)
		}
		p.Combat = *c
	}
}

func hurt(a *row.Combat, f fact.Hurt) {
	w := -999
	if f.Weapon != nil {
		w = *f.Weapon
	}
	d := f.HealthDamageTaken
	a.Damage += d
	if w >= 500 && w <= fact.EqHE {
		a.UtilDmg += d
		if w == fact.EqHE {
			a.NadeDmg += d
		}
		if w == fact.EqMolotov || w == fact.EqIncendiary {
			a.InfernoDmg += d
		}
	}
}

func kill(rows map[uint64]*row.Combat, ecos map[uint64]*eco, damage map[uint64]map[uint64]int, e round.Event, f fact.Kill) {
	if e.Victim {
		v := rows[f.Victim.Steam64]
		v.Deaths++
		// Bots share one row, so a second bot death adds the whole list again.
		for supp, dmg := range damage[f.Victim.Steam64] {
			if e.Killer && supp == f.Killer.Steam64 {
				continue
			}
			s := rows[supp]
			s.SuppDamage += dmg
			if s.SuppDamage > 60 {
				s.SuppRounds = 1
			}
		}
		for _, a := range e.Avenged {
			rows[a.Steam].Eac++
		}
		if x := ecos[f.Victim.Steam64]; x != nil {
			for _, w := range f.Victim.Weapons {
				if w == fact.EqAWP {
					x.lostAWP = true
					break
				}
			}
		}
	}
	if e.Assist {
		a := rows[f.Assister.Steam64]
		a.Assists++
		a.Eac++
		a.SuppRounds = 1
		// A flasher who is also the damage assister has Eac once (D34); bots, sharing Steam64 0, count twice.
		if e.FlashAssisted && !f.AssistedFlash && (f.Assister.IsBot || e.FlashAssister != f.Assister.Steam64) {
			if fl := rows[e.FlashAssister]; fl != nil {
				fl.Eac++
				fl.SuppRounds = 1
			}
		}
		if x := ecos[f.Assister.Steam64]; x != nil {
			x.assists++
		}
	}
	if e.Enemy {
		k := rows[f.Killer.Steam64]
		k.Kills++
		if f.Weapon != nil && *f.Weapon == fact.EqAWP {
			k.AwpKills++
			if f.Killer.Side == fact.SideCT {
				k.CtAWP++
			}
		}
		if f.Headshot {
			k.Hs++
		}
	}
	if x := ecos[killerID(f)]; e.Enemy && x != nil {
		x.kills++
		if f.Headshot {
			x.perfect++
		}
		if f.Weapon != nil {
			w := *f.Weapon
			switch w {
			case fact.EqAWP:
				x.awpKills++
			case fact.EqKnife:
				x.knife = true
			case fact.EqHE, fact.EqMolotov, fact.EqIncendiary:
				x.utilKills++
			}
			if w >= fact.EqP2000 && w <= fact.EqRevolver && f.Victim.EquipValue > 3500 {
				x.pistolVsRifle = true
			}
		}
	}
}

// killerID is the killer's Steam64; an enemy kill has a killer.
func killerID(f fact.Kill) uint64 {
	if f.Killer == nil {
		return 0
	}
	return f.Killer.Steam64
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
