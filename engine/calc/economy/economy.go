// Package economy is the economy family: each team's buy and the value of each kill and death by
// the equipment of the players in it.
package economy

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	"github.com/csconfederation/fragg-3.0/rating"
)

// A team's buy by its average equipment per player at freeze end.
const (
	ecoMax   = 1500
	forceMax = 3500
)

// Calc writes each player's row.Economy and each team's row.TeamEconomy.
func Calc(r *round.Round, out *row.Round) {
	t, ct := buy(r, fact.SideT), buy(r, fact.SideCT)
	for side, b := range map[fact.Side][2]row.Buy{fact.SideT: {t, ct}, fact.SideCT: {ct, t}} {
		tm := out.Teams[side]
		tm.Buy = b[0]
		if r.Num == fact.FirstPistolRound || r.Num == fact.SecondPistolRound {
			tm.Pistols = 1
		}
		switch b[0] {
		case row.BuyEco:
			tm.EcoBuys = 1
		case row.BuyForce:
			tm.ForceBuys = 1
		case row.BuyFull:
			tm.FullBuys = 1
			if b[1] == row.BuyEco {
				tm.AntiEcos = 1
			}
		}
	}
	// The winner's team takes the round as won in every count it has.
	if win, ok := r.WinAt(r.Commit); ok {
		if tm := out.Teams[r.SideOf(win.Clan)]; tm != nil {
			tm.PistolsW = tm.Pistols
			tm.EcoBuysW, tm.ForceBuysW, tm.FullBuysW, tm.AntiEcosW = tm.EcoBuys, tm.ForceBuys, tm.FullBuys, tm.AntiEcos
		}
	}

	in := map[uint64]bool{} // the players with a row so far: one who joins mid-round has none before
	for id := range r.OpenMembers {
		in[id] = true
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil && !e.Dropped {
			in[e.Join.Steam64] = true
		}
		k, ok := e.Fact.(fact.Kill)
		if !ok || e.Dropped || !e.Enemy {
			continue
		}
		// Eco has no bot rows: a bot's share of the kill or death goes nowhere.
		attacker, victim := float64(k.Killer.EquipValue), float64(k.Victim.EquipValue)
		if id := k.Killer.Steam64; id != 0 && in[id] && out.Players[id] != nil {
			p := out.Players[id]
			value := rating.EcoKillValue(attacker, victim)
			p.EcoKillValue += value
			if value < 1.0 {
				p.KillsVsLowerBuy++
			}
			if value <= 0.85 {
				p.KillsVsMuchLowerBuy++
			}
		}
		if id := k.Victim.Steam64; id != 0 && in[id] && out.Players[id] != nil {
			out.Players[id].EcoDeathValue += rating.EcoDeathPenalty(victim, attacker)
		}
	}
}

// buy classifies side's buy from the round's open snapshot.
func buy(r *round.Round, side fact.Side) row.Buy {
	if r.Num == fact.FirstPistolRound || r.Num == fact.SecondPistolRound {
		return row.BuyPistol
	}
	members := r.Roster.Members(r.Open, side)
	if !r.FreezeEnd || len(members) == 0 {
		return row.BuyUnknown
	}
	equip := 0
	for _, p := range members {
		equip += p.EquipValue
	}
	switch avg := equip / len(members); {
	case avg < ecoMax:
		return row.BuyEco
	case avg < forceMax:
		return row.BuyForce
	}
	return row.BuyFull
}
