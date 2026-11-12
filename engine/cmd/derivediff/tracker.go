package main

// The baseline's round.Tracker (ca9fe6b), verbatim except for the types, which engine/round still
// owns. derivediff drives it the old way and compares it with round.Deriver; PR 18 deletes it.

import (
	"fmt"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	. "github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/rating"
)

const (
	mr       = 12
	eqBomb   = 404
	tickRate = 64
)

// kill is an enemy kill, kept for trades.
type kill struct {
	Death
	team   int // the victim's team state
	traded bool
}

// Tracker computes the state over the lifecycle, before the collectors see each phase or fact.
type Tracker struct {
	s       State
	cur     *match.Round
	members map[uint64]bool    // the open round's rows: members at the open and joins
	deaths  map[uint64]int     // counted deaths this round
	kills   map[uint64][]*kill // each killer's enemy kills this round, in order, by actor
	blind   map[uint64]Blind   // each player's latest counting flash this round
}

// NewTracker returns the state before any round.
func NewTracker() *Tracker {
	return &Tracker{s: State{Phase: PrePlant}, members: map[uint64]bool{}, deaths: map[uint64]int{},
		kills: map[uint64][]*kill{}, blind: map[uint64]Blind{}}
}

// State is the current state.
func (t *Tracker) State() State { return t.s }

func (t *Tracker) Open(r *match.Round) {
	// Num wraps like CSC's int8 roundNum past round 127, as the gates always read it.
	t.s = State{Num: int(int8(r.Num)), Phase: PrePlant, Opening: true, TAlive: r.InitT, CTAlive: r.InitCT}
	t.cur = r
	t.members = map[uint64]bool{}
	for id := range r.Members {
		t.members[id] = true
	}
	t.deaths = map[uint64]int{}
	t.kills = map[uint64][]*kill{}
	t.blind = map[uint64]Blind{}
}

func (t *Tracker) Join(r *match.Round, steam uint64) {
	if r == t.cur {
		t.members[steam] = true
	}
}

// RoundEnd is every round end while the match is live.
func (t *Tracker) RoundEnd(f fact.RoundEnd) {
	if f.Winner == fact.SideT {
		t.s.TMoney = true
	}
}

func (t *Tracker) Win() { t.s.Phase = Decided }

// Fact applies f and returns its context. Nothing changes while the match isn't live.
func (t *Tracker) Fact(m *match.Match, f fact.Fact) (Context, error) {
	c := Context{Before: t.s}
	if m.Live() {
		if err := t.apply(m, f, &c); err != nil {
			return Context{}, err
		}
	}
	c.After = t.s
	switch f := f.(type) {
	case fact.Kill:
		if c.Expected {
			c.Alive = t.Alive(m, f.Snapshot)
		}
	case fact.BombPlanted:
		c.Alive = t.Alive(m, f.Snapshot)
	case fact.BombDefused:
		c.Alive = t.Alive(m, f.Snapshot)
	}
	return c, nil
}

// Alive is the open round's members alive in s, T then CT.
func (t *Tracker) Alive(m *match.Match, s fact.Snapshot) []fact.Player {
	var out []fact.Player
	for _, side := range []fact.Side{fact.SideT, fact.SideCT} {
		for _, p := range m.Members(s, side) {
			if t.members[p.Steam64] && t.alive(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func (t *Tracker) apply(m *match.Match, f fact.Fact, c *Context) error {
	decided := t.s.Phase == Decided
	switch f := f.(type) {
	case fact.Kill:
		return t.kill(m, f, c)
	case fact.Hurt:
		w := -999
		if f.Weapon != nil {
			w = *f.Weapon
		}
		c.BombIgnored = t.bombIgnored(w)
		c.Enemy = f.Attacker != nil && f.Player != nil && f.Attacker.Side != f.Player.Side
	case fact.Flashed:
		t.flashed(f, c)
	case fact.BombPlanted:
		if !decided {
			if f.Player == nil {
				return fmt.Errorf("bomb planted by nobody")
			}
			t.s.Phase, t.s.TMoney = PostPlant, true
		}
	case fact.BombDefused:
		if !decided {
			if f.Player == nil {
				return fmt.Errorf("bomb defused by nobody")
			}
			t.s.Phase, t.s.BombDecided = Decided, true
		}
	case fact.BombExplode:
		if !decided {
			t.s.Phase, t.s.BombDecided = Decided, true
		}
	case fact.Disconnect:
		// Only the round's members, as Alive: a mid-round reconnect can be relisted alive without
		// joining the round (S19 Combine mid7214).
		t.s.TAlive, t.s.CTAlive = 0, 0
		for _, p := range m.Members(f.Snapshot, fact.SideT) {
			if t.members[p.Steam64] && t.alive(p) {
				t.s.TAlive++
			}
		}
		for _, p := range m.Members(f.Snapshot, fact.SideCT) {
			if t.members[p.Steam64] && t.alive(p) {
				t.s.CTAlive++
			}
		}
	}
	return nil
}

func isRoundFinalInHalf(round int) bool {
	return round%mr == 0 || (round > mr*2 && round%3 == 0)
}

// bombIgnored is the filter on bomb kills and damage in a half's final round, where a bomb can
// explode after the round was won by elimination. D10: the bomb still counts when it ends the
// round, or when the round is undecided (v5 delivered the bomb's kills before the explosion).
func (t *Tracker) bombIgnored(weapon int) bool {
	return weapon == eqBomb && isRoundFinalInHalf(t.s.Num) && t.s.Phase == Decided && !t.s.BombDecided
}

func (t *Tracker) kill(m *match.Match, f fact.Kill, c *Context) error {
	// The kill belongs to the open round.
	if t.s.Num != f.RoundsPlayed+1 && (t.s.Num != f.RoundsPlayed || t.s.Phase != Decided) {
		return nil
	}
	c.Expected = true
	if f.Weapon == nil {
		return fmt.Errorf("kill without a weapon at tick %d", f.Tick) // the oracle reads e.Weapon.Type
	}
	c.BombIgnored = t.bombIgnored(*f.Weapon)
	c.Victim = f.Victim != nil && t.members[f.Victim.Steam64] && !c.BombIgnored
	c.Killer = f.Killer != nil && t.members[f.Killer.Steam64] && !c.BombIgnored
	if c.Victim {
		t.deaths[actor(f.Victim)]++
		switch f.Victim.Side {
		case fact.SideT:
			t.s.TAlive--
		case fact.SideCT:
			t.s.CTAlive--
		}
		// Clutches (corpus finding 6): a bomb death at the explosion can't start one, and the
		// clutcher is the side's member with no death this round, not the entity's IsAlive(),
		// which lags the kill.
		if t.s.Phase != Decided && *f.Weapon != eqBomb {
			if t.s.TAlive == 1 && t.s.TClutch.Size == 0 {
				t.s.TClutch = Clutch{Size: t.s.CTAlive, Steam: t.survivor(m.Members(f.Snapshot, fact.SideT))}
			}
			if t.s.CTAlive == 1 && t.s.CTClutch.Size == 0 {
				t.s.CTClutch = Clutch{Size: t.s.TAlive, Steam: t.survivor(m.Members(f.Snapshot, fact.SideCT))}
			}
		}
	}
	if c.Killer && c.Victim {
		enemy, err := Enemies(f.Killer, f.Victim)
		if err != nil {
			return err
		}
		c.Enemy = enemy
	}
	if c.Enemy {
		t.s.Opening = false
		t.trade(f, c)
	}
	if c.Victim && f.Assister != nil && t.members[f.Assister.Steam64] {
		assist, err := Enemies(f.Assister, f.Victim)
		if err != nil {
			return err
		}
		c.Assist = assist
	}
	if c.Assist {
		if f.AssistedFlash {
			c.FlashAssister, c.FlashAssisted = f.Assister.Steam64, true
		} else if b := t.blind[f.Victim.Steam64]; float64(f.Tick) < b.Until {
			c.FlashAssister, c.FlashAssisted = b.Flasher, true
		}
	}
	return nil
}

// flashed is CSC's flash filter and its latest-flash state (the longest blind wins).
func (t *Tracker) flashed(f fact.Flashed, c *Context) {
	if f.Player == nil || f.Attacker == nil {
		return
	}
	blind := f.Duration * tickRate
	v, a := f.Player, f.Attacker
	// Both must be round members (rows in every collector): a mid-round reconnect can be relisted
	// without joining the round (S19 Combine mid7214), and the oracle dereferences nil for it.
	if !t.members[v.Steam64] || !t.members[a.Steam64] {
		return
	}
	if a.Side == v.Side || blind <= tickRate || !v.Alive || v.FlashDuration >= blind/tickRate+1 {
		return
	}
	c.Flash = true
	b := t.blind[v.Steam64]
	if until := float64(f.Tick) + blind; until > b.Until {
		b = Blind{Flasher: a.Steam64, Until: until}
		t.blind[v.Steam64] = b
	}
	c.Blind = b
}

// trade is an enemy kill's trades (D22): the victim's untraded enemy kills of the killer's
// teammates within the window, then the kill itself becomes tradeable.
func (t *Tracker) trade(f fact.Kill, c *Context) {
	for _, k := range t.kills[actor(f.Victim)] {
		if !k.traded && k.team == *f.Killer.TeamStateID && f.Tick-k.Tick <= rating.TradeWindowTicks {
			k.traded = true
			c.Avenged = append(c.Avenged, k.Death)
		}
	}
	t.kills[actor(f.Killer)] = append(t.kills[actor(f.Killer)],
		&kill{Death: Death{Steam: f.Victim.Steam64, Tick: f.Tick}, team: *f.Victim.TeamStateID})
}

// actor tells players apart for trades: bots share Steam64 0, so they go by user ID.
func actor(p *fact.Player) uint64 {
	if p.IsBot {
		return 1<<63 | uint64(p.UserID)
	}
	return p.Steam64
}

// alive is p alive at the start of the tick with no death this round: entity state is applied
// after the tick's events, so it still shows earlier victims of the tick alive. Deaths go by actor:
// keyed by Steam64, one bot's death made every bot (Steam64 0) dead at a disconnect recount.
func (t *Tracker) alive(p fact.Player) bool {
	return p.Alive && t.deaths[actor(&p)] == 0
}

// survivor is the last alive member with a round row.
func (t *Tracker) survivor(members []fact.Player) uint64 {
	var id uint64
	for _, p := range members {
		if t.members[p.Steam64] && t.alive(p) {
			id = p.Steam64
		}
	}
	return id
}

// IsMember and AliveP expose the row test and alive test to the driver's residual.
func (t *Tracker) IsMember(steam uint64) bool { return t.members[steam] }
func (t *Tracker) AliveP(p fact.Player) bool  { return t.alive(p) }

// endsBombWave is the baseline's match.endsBombWave: what ended an explosion Win's wait.
func endsBombWave(f fact.Fact) bool {
	switch f := f.(type) {
	case fact.Kill, fact.Hurt, fact.Flashed:
		return !bombFact(f)
	case fact.PlayerInfo:
		return f.Found
	case fact.Disconnect, fact.TeamChange:
		return true
	}
	return false
}

func bombFact(f fact.Fact) bool {
	var w *int
	switch f := f.(type) {
	case fact.Hurt:
		w = f.Weapon
	case fact.Kill:
		w = f.Weapon
	}
	return w != nil && *w == eqBomb
}
