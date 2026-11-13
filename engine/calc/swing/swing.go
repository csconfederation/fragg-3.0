// Package swing is the swing family: each player's probability swing from the round's kills,
// bomb events and end, the kills' duel-adjusted value, and the time from first damage to kill.
// It runs the eco model's trackers (swing_tracker.go) over the round's events in order.
package swing

import (
	"maps"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	ratingswing "github.com/csconfederation/fragg-3.0/rating/swing"
)

// A row is a human member's running totals in the round; bots and non-members have none.
type member struct {
	side           fact.Side
	damage, kills  int
	flashAssists   int
	planted        bool
	defused        bool
	swing          float64
	lastDeathSwing float64
	ttkSum         float64
	ttkKills       int
	adjusted       float64
}

type state struct {
	r       *round.Round
	t       *SwingTracker
	rows    map[uint64]*member
	members map[uint64]bool // the round's rows, bots too: the deriver's membership
}

// Calc writes each player's row.Swing.
func Calc(r *round.Round, out *row.Round) {
	s := &state{r: r, t: NewSwingTrackerWithConfig(ratingswing.DefaultConfig()), rows: map[uint64]*member{},
		members: map[uint64]bool{}}
	s.open()
	wins := r.WinList()
	next := 0
	for i := 0; i <= r.Commit; i++ {
		for ; next < len(wins) && wins[next].At == i; next++ {
			s.win(wins[next])
		}
		if i == r.Commit {
			break
		}
		s.event(&r.Events[i])
	}
	for id, m := range s.rows {
		if p := out.Players[id]; p != nil {
			p.EcoProbabilitySwing, p.EcoAdjustedKills = m.swing, m.adjusted
			p.TimeToKillSum, p.KillsWithTTK = m.ttkSum, m.ttkKills
		}
	}
}

func (s *state) seconds(tick int) float64 {
	return float64(tick-s.r.Open.Tick) / float64(fact.TickRate)
}

// open is the oracle's BeginRound: the rows of the round's humans, alive counts and economy.
func (s *state) open() {
	r := s.r
	tCount, ctCount, tEquip, ctEquip := 0, 0, 0, 0
	for _, side := range []fact.Side{fact.SideT, fact.SideCT} {
		for _, p := range r.Roster.Members(r.Open, side) {
			switch p.Side {
			case fact.SideT:
				tCount++
				tEquip += p.EquipValue
			case fact.SideCT:
				ctCount++
				ctEquip += p.EquipValue
			}
			if !p.IsBot {
				s.rows[p.Steam64] = &member{side: p.Side}
			}
		}
	}
	for id := range r.OpenMembers {
		s.members[id] = true
	}
	s.t.ResetRound(min(r.InitT, 5), min(r.InitCT, 5), r.Map, r.Open.Tick)
	tAvg, ctAvg := 0.0, 0.0
	if tCount > 0 {
		tAvg = float64(tEquip) / float64(tCount)
	}
	if ctCount > 0 {
		ctAvg = float64(ctEquip) / float64(ctCount)
	}
	s.t.SetEconomyFromValues(tAvg, ctAvg)
}

// row is steam's row, or a scratch row nobody reads.
func (s *state) row(steam uint64) *member {
	if m, ok := s.rows[steam]; ok {
		return m
	}
	return &member{}
}

func (s *state) applyLedger(allocs []ratingswing.PlayerSwingAllocation) {
	for _, a := range allocs {
		if m, ok := s.rows[a.SteamID]; ok && a.Amount != 0 {
			m.swing += a.Amount
		}
	}
}

// context is the oracle's buildPlayerRoundContext over the members alive.
func (s *state) context(alive map[uint64]bool) map[uint64]ratingswing.PlayerRoundContext {
	players := map[uint64]ratingswing.PlayerRoundContext{}
	for id, m := range s.rows {
		positive, negative := 0.0, 0.0
		if m.swing > 0 {
			positive = m.swing
		} else if m.swing < 0 {
			negative = -m.swing
		}
		players[id] = ratingswing.PlayerRoundContext{Side: m.side, Alive: alive[id], Damage: m.damage, Kills: m.kills,
			FlashAssists: m.flashAssists, PositiveSwing: positive, NegativeSwing: negative,
			PlantedBomb: m.planted, DefusedBomb: m.defused}
	}
	return players
}

func aliveSet(ps []fact.Player) map[uint64]bool {
	alive := map[uint64]bool{}
	for _, p := range ps {
		alive[p.Steam64] = true
	}
	return alive
}

// win is the oracle's FinalizeRound: the residual swing from where the Win's RoundEnd left the round.
// The members alive are its snapshot's, less the deaths counted before the Win takes effect.
func (s *state) win(w round.Win) {
	e := s.r.Events[w.Event]
	re := e.Fact.(fact.RoundEnd)
	if re.Snapshot.Warmup {
		return
	}
	deaths, members := map[uint64]int{}, maps.Clone(s.members)
	for _, c := range s.r.Events[:w.At] {
		if c.Join != nil && !c.Dropped {
			members[c.Join.Steam64] = true
		}
		if k, ok := c.Fact.(fact.Kill); ok && c.Victim {
			deaths[round.Actor(k.Victim)]++
		}
	}
	alive := map[uint64]bool{}
	for _, side := range []fact.Side{fact.SideT, fact.SideCT} {
		for _, p := range e.Roster.Members(re.Snapshot, side) {
			if members[p.Steam64] && p.Alive && deaths[round.Actor(&p)] == 0 {
				alive[p.Steam64] = true
			}
		}
	}
	maxDamage := 0
	for _, m := range s.rows {
		maxDamage = max(maxDamage, m.damage)
	}
	s.applyLedger(s.t.ApplyRoundResidual(re.Winner, s.context(alive), maxDamage))
	s.t.FinalizeRound()
}

func (s *state) event(e *round.Event) {
	if e.Dropped {
		return
	}
	if e.Join != nil {
		s.members[e.Join.Steam64] = true
		if _, ok := s.rows[e.Join.Steam64]; !ok && !e.Join.IsBot {
			s.rows[e.Join.Steam64] = &member{side: e.Join.Side}
		}
		return
	}
	switch f := e.Fact.(type) {
	case fact.RoundEnd, fact.TeamChange:
		return // not a stat fact: the alive counts stay
	case fact.Kill:
		s.kill(f, e)
	case fact.Hurt:
		if e.Dealt {
			s.row(f.Attacker.Steam64).damage += f.HealthDamageTaken
			s.t.RecordDamage(f.Attacker.Steam64, f.Player.Steam64, f.HealthDamageTaken, s.seconds(f.Tick))
		}
	case fact.Flashed:
		if f.Attacker != nil && f.Player != nil && f.Attacker.Side != f.Player.Side {
			s.t.RecordFlash(f.Attacker.Steam64, f.Player.Steam64, f.Duration, f.Tick)
		}
	case fact.BombPlanted:
		if f.Player != nil {
			s.row(f.Player.Steam64).planted = true
			s.applyLedger(s.t.RecordBombPlant(f.Player.Steam64, s.context(aliveSet(e.Alive)), f.Tick))
		}
	case fact.BombDefused:
		if f.Player != nil {
			s.row(f.Player.Steam64).defused = true
			s.applyLedger(s.t.RecordBombDefuse(f.Player.Steam64, s.context(aliveSet(e.Alive)), f.Tick))
		}
	}
	s.t.SetAlive(e.After.TAlive, e.After.CTAlive)
}

// kill is the oracle's handleKill, as far as swing reads it. A kill reaches swing's kill transition when
// it is an enemy kill between members (Enemy).
func (s *state) kill(f fact.Kill, e *round.Event) {
	if len(e.Avenged) > 0 { // a trade refunds the oldest avenged death's swing penalty
		oldest := e.Avenged[0].Steam
		if d, ok := s.rows[oldest]; ok && d.lastDeathSwing < 0 {
			var mates []uint64 // the round's humans on the trader's side
			for _, id := range slices.Sorted(maps.Keys(s.rows)) {
				if s.rows[id].side == f.Killer.Side && id != oldest {
					mates = append(mates, id)
				}
			}
			s.applyLedger(s.t.ApplyTradeReallocation(oldest, f.Killer.Side, d.lastDeathSwing, mates))
		}
	}
	swung := false
	if e.Enemy {
		swung = e.Victim
		attacker, victim := f.Killer, f.Victim
		killer, vr := s.row(attacker.Steam64), s.row(victim.Steam64)
		killer.kills++
		tick, t := f.Tick, s.seconds(f.Tick)
		if ttk := s.t.GetTimeToKill(attacker.Steam64, victim.Steam64, t); ttk >= 0 {
			killer.ttkSum += ttk
			killer.ttkKills++
		}
		if e.Victim {
			var losing []uint64
			for _, p := range e.Alive {
				if !p.IsBot && p.Side == victim.Side {
					losing = append(losing, p.Steam64)
				}
			}
			kr := s.t.RecordKill(attacker.Steam64, victim.Steam64, attacker.Side, victim.Side,
				float64(attacker.EquipValue), float64(victim.EquipValue), len(e.Avenged) > 0,
				e.Before.Phase == round.Decided, losing, tick)
			s.applyLedger(kr.Allocations)
			for _, a := range kr.Allocations {
				if a.SteamID == victim.Steam64 && a.Reason == ratingswing.SwingReasonVictimDeath {
					vr.lastDeathSwing = a.Amount
				}
			}
			if kr.SurvivalCreditPerPlayer > 0 {
				allocs := make([]ratingswing.PlayerSwingAllocation, 0, len(kr.SurvivalBeneficiaries))
				for _, id := range kr.SurvivalBeneficiaries {
					allocs = append(allocs, ratingswing.PlayerSwingAllocation{SteamID: id, Side: attacker.Side,
						Amount: kr.SurvivalCreditPerPlayer, Reason: ratingswing.SwingReasonSurvivalCredit})
				}
				s.applyLedger(allocs)
			}
			if kr.Swing.EcoMultiplier > 0 {
				killer.adjusted += kr.Swing.EcoMultiplier
			}
		}
	}
	if e.Victim && !swung { // a team kill, suicide or world death: the victim's advantage slots and damage go
		s.t.RecordDeath(f.Victim.Steam64, f.Victim.Side)
	}
	if e.FlashAssisted {
		s.row(e.FlashAssister).flashAssists++
	}
}
