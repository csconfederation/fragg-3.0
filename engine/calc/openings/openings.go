// Package openings is the openings family: the round's opening kill and death, entries, multi-kill
// rounds, rounds with a kill, and the man-advantage kills and deaths; and each team's 4v5 and 5v4
// starts (design note section 3). The opening is the Deriver's (Before.Opening: no enemy kill yet),
// but the 4v5 start is the first death of any kind (Q11). Three fields are eco's, with its gates:
// awpOpeningKills, manAdvantageKills and manDisadvantageDeaths.
package openings

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// eco is the eco row's opening counters for the round.
type eco struct {
	awpOpening      bool
	advKills, disad int
}

// Calc writes each player's Openings and each team's TeamOpenings.
func Calc(r *round.Round, out *row.Round) {
	rows := map[uint64]*row.Openings{}
	ecos := map[uint64]*eco{} // eco has rows for the round's humans, from the open or their join
	members := map[uint64]match.Member{}
	kills := map[uint64]int{} // enemy kills by CSC row
	for id, m := range r.OpenMembers {
		rows[id], members[id] = &row.Openings{}, m
		if !m.IsBot {
			ecos[id] = &eco{}
		}
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil && rows[e.Join.Steam64] == nil {
			rows[e.Join.Steam64], members[e.Join.Steam64] = &row.Openings{}, *e.Join
		}
	}
	fourVFive := map[fact.Side]bool{}
	fiveVFour := map[fact.Side]bool{}

	for _, e := range r.Events[:r.Commit] {
		if e.Dropped {
			continue
		}
		if j := e.Join; j != nil && !j.IsBot && ecos[j.Steam64] == nil {
			ecos[j.Steam64] = &eco{} // a kill before the join was a scratch row's
		}
		f, ok := e.Fact.(fact.Kill)
		if !ok || !e.Expected {
			continue
		}
		if e.Victim && e.Before.Opening && r.InitT+r.InitCT == 10 {
			// The first death of any kind: both teams can end up 4v5 in a round (Q11).
			victim := r.SideOf(members[f.Victim.Steam64].Clan)
			fourVFive[victim] = true
			for _, s := range []fact.Side{fact.SideT, fact.SideCT} {
				if s != victim {
					fiveVFour[s] = true
				}
			}
		}
		if e.Enemy {
			k := rows[f.Killer.Steam64]
			k.Rwk = 1
			kills[f.Killer.Steam64]++
			if e.Before.Opening {
				k.Ok = 1
				rows[f.Victim.Steam64].Ol = 1
				if f.Killer.Side == fact.SideT && e.Before.Phase == round.PrePlant {
					k.Entries = 1
				}
			}
		}
		if e.EcoKill {
			if x := ecos[f.Killer.Steam64]; x != nil {
				if e.Before.Opening && e.Enemy && f.Weapon != nil && *f.Weapon == fact.EqAWP {
					x.awpOpening = true
				}
				ka, va := alive(e.Before, f.Killer.Side), alive(e.Before, f.Victim.Side)
				if ka <= va && alive(e.After, f.Killer.Side) > alive(e.After, f.Victim.Side) {
					x.advKills++
				}
			}
			if x := ecos[f.Victim.Steam64]; x != nil {
				va, ka := alive(e.Before, f.Victim.Side), alive(e.Before, f.Killer.Side)
				if va >= ka && alive(e.After, f.Victim.Side) < alive(e.After, f.Killer.Side) {
					x.disad++
				}
			}
		}
	}

	for id, o := range rows {
		p := out.Players[id]
		if p == nil {
			continue
		}
		switch kills[id] { // a bot row shares every bot's kills: above five has no bonus
		case 2:
			o.TwoK = 1
		case 3:
			o.ThreeK = 1
		case 4:
			o.FourK = 1
		case 5:
			o.FiveK = 1
		}
		if x := ecos[id]; x != nil {
			o.AWPOpeningKills = b2i(x.awpOpening)
			o.ManAdvantageKills, o.ManDisadvantageDeaths = x.advKills, x.disad
		}
		p.Openings = *o
	}

	// The winner's team takes a W for its start: its own 4v5 first, else its 5v4.
	if win, ok := r.WinAt(r.Commit); ok {
		if t := out.Teams[r.SideOf(win.Clan)]; t != nil {
			if fourVFive[r.SideOf(win.Clan)] {
				t.FourVFiveW = 1
			} else if fiveVFour[r.SideOf(win.Clan)] {
				t.FiveVFourW = 1
			}
		}
	}
	for s, t := range out.Teams {
		t.FourVFiveS, t.FiveVFourS = b2i(fourVFive[s]), b2i(fiveVFour[s])
	}
}

func alive(s round.State, side fact.Side) int {
	if side == fact.SideT {
		return s.TAlive
	}
	return s.CTAlive
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
