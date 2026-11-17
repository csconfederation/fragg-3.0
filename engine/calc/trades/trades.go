// Package trades is the trades family (design note section 3). Trades themselves are the
// Deriver's (Context.Avenged, D22). CSC's fields are trades and traded; the six eco-origin ones
// (tradeKills, fastTrades, tradeDenials, savedByTeammate, savedTeammate, openingDeathsTraded)
// follow eco's gates until D31 unifies them, and its trade denials, which come from nearby
// teammates and window expiry (checked at each kill and at the round's end, D41), from the Kill snapshots.
package trades

import (
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	"github.com/csconfederation/fragg-3.0/rating"
)

// eco is the eco row's trade flags and counts for the round.
type eco struct {
	kills, fast, denials                 int
	traded, savedBy, saved, openingDeath bool
}

// Calc writes each player's Trades.
func Calc(r *round.Round, out *row.Round) {
	rows := map[uint64]*row.Trades{}
	ecos := map[uint64]*eco{} // eco has rows for the round's humans, from the open or their join
	for id, m := range r.OpenMembers {
		rows[id] = &row.Trades{}
		if !m.IsBot {
			ecos[id] = &eco{}
		}
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil && rows[e.Join.Steam64] == nil {
			rows[e.Join.Steam64] = &row.Trades{}
		}
	}
	clock := func(tick int) float64 { return float64(tick-r.StartTick) / float64(fact.TickRate) }
	pending := map[uint64][]int{} // each killer's pending trades: the tick of the death, once per nearby teammate

	expire := func(tick int) { // a pass drops what's past the window; a killer is denied once per pass
		for id, ticks := range pending {
			kept, expired := ticks[:0], false
			for _, t := range ticks {
				if tick-t > rating.TradeWindowTicks {
					expired = true
				} else {
					kept = append(kept, t)
				}
			}
			if expired {
				if k := ecos[id]; k != nil {
					k.denials++
				}
			}
			if len(kept) > 0 {
				pending[id] = kept
			} else {
				delete(pending, id)
			}
		}
	}

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
		if e.Enemy && len(e.Avenged) > 0 {
			rows[f.Killer.Steam64].Trades++
			for _, a := range e.Avenged {
				rows[a.Steam].Traded = 1
			}
		}
		// Eco's kill handler runs for kills that count, unless both players are known and it's a
		// suicide or team kill. Until the victim and killer are both known, only the expiry runs.
		if e.BombIgnored || f.Killer != nil && f.Victim != nil && !e.Enemy {
			continue
		}
		known := f.Killer != nil && f.Victim != nil
		if known {
			for _, p := range f.Players {
				if !p.Listed || p.Side != f.Victim.Side || !p.Alive || p.Steam64 == f.Victim.Steam64 {
					continue
				}
				dx, dy := f.Victim.Pos[0]-p.Pos[0], f.Victim.Pos[1]-p.Pos[1]
				if math.Sqrt(dx*dx+dy*dy) < rating.TradeProximityUnits {
					pending[f.Killer.Steam64] = append(pending[f.Killer.Steam64], f.Tick)
				}
			}
		}
		if len(e.Avenged) > 0 {
			for _, a := range e.Avenged {
				if d := ecos[a.Steam]; d != nil {
					d.traded, d.savedBy = true, true
				}
			}
			if k := ecos[f.Killer.Steam64]; k != nil {
				k.saved = true
				k.denials++
			}
		}
		if known {
			delete(pending, f.Victim.Steam64) // the victim's kills can't be traded any more
		}
		expire(f.Tick)
		if !known {
			continue
		}
		if len(e.Avenged) > 0 {
			if k := ecos[f.Killer.Steam64]; k != nil {
				k.kills++
				if speed := clock(f.Tick) - clock(e.Avenged[0].Tick); speed < 2.0 {
					k.fast++
				}
			}
		}
		if e.Before.Opening && e.Enemy {
			if v := ecos[f.Victim.Steam64]; v != nil {
				v.openingDeath = true
			}
		}
	}

	expire(r.EndTick) // D41: a window that runs out at the round's end still denies

	for id, t := range rows {
		p := out.Players[id]
		if p == nil {
			continue
		}
		if x := ecos[id]; x != nil {
			t.TradeKills, t.FastTrades, t.TradeDenials = x.kills, x.fast, x.denials
			t.OpeningDeathsTraded = b2i(x.openingDeath && x.traded)
			t.SavedByTeammate, t.SavedTeammate = b2i(x.savedBy), b2i(x.saved)
		}
		p.Trades = *t
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
