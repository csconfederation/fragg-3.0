package cscstats

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/rating"
)

// Buy is a team's buy in a round (Core-planning#590).
type Buy uint8

const (
	BuyUnknown Buy = iota // the round committed without a freeze-end open, so before the buy
	BuyPistol             // rounds 1 and 13 (D13)
	BuyEco                // average equipment per player under $1,500 at freeze end
	BuyForce              // $1,500-3,499
	BuyFull               // $3,500 and up
)

const (
	ecoMax   = 1500
	forceMax = 3500
)

// buy classifies side's buy from the round's open snapshot.
func buy(m *match.Match, r *match.Round, s fact.Snapshot, side fact.Side) Buy {
	if r.Num == rating.FirstHalfPistolRound || r.Num == rating.SecondHalfPistolRound {
		return BuyPistol
	}
	members := m.Members(s, side)
	if !r.FreezeEnd || len(members) == 0 {
		return BuyUnknown
	}
	equip := 0
	for _, p := range members {
		equip += p.EquipValue
	}
	switch avg := equip / len(members); {
	case avg < ecoMax:
		return BuyEco
	case avg < forceMax:
		return BuyForce
	}
	return BuyFull
}

// countBuy marks the round's buy on its team row; Commit marks the winner's.
func (t *teamStats) countBuy(opponent Buy) {
	switch t.Buy {
	case BuyEco:
		t.EcoBuys = 1
	case BuyForce:
		t.ForceBuys = 1
	case BuyFull:
		t.FullBuys = 1
		if opponent == BuyEco {
			t.AntiEcos = 1
		}
	}
}
