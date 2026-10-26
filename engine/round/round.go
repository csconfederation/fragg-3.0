// Package round owns the round state the stat collectors share (Core-planning#591): roster,
// alive counts, phase, opening and clutch slots, and which facts count. For each fact it returns a
// Context; collectors read it and never change the state. Step 1 ports CSC's rules unchanged.
package round

import (
	"fmt"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/rating"
)

const (
	mr       = 12
	eqBomb   = 404
	tickRate = 64
)

// Phase is where the round stands. Exactly one holds at a time.
type Phase uint8

const (
	PrePlant  Phase = iota
	PostPlant       // the bomb is planted and the round undecided
	Decided         // won by elimination, time, defuse or explosion
)

// Clutch is a side's clutch slot: the last player alive and the enemies left when they became it.
type Clutch struct {
	Size  int
	Steam uint64
}

// State is the shared round state.
type State struct {
	Num         int // the open round's number (CSC's int8 roundNum); 0 before the first
	Phase       Phase
	BombDecided bool // a defuse or explosion decided the round
	TMoney      bool // the T side earns the bomb or win money this round
	TAlive      int
	CTAlive     int
	Opening     bool // no enemy kill yet this round
	TClutch     Clutch
	CTClutch    Clutch
}

// Context is one fact's view of the state: before the fact and after it.
type Context struct {
	Before, After State
	// Hurts: Enemy is damage from the player's enemy, by side; damage taken counts only that (#591).
	// Kills: Expected is the kill belonging to the open round. Victim and Killer are each player
	// having a round row with the kill counting (D10); Enemy is both, on opposite teams. Assist is
	// the assister having a row and being the counted victim's enemy (CSC's rule).
	Expected, Victim, Killer, Enemy, Assist bool
	// BombIgnored is a bomb kill or hurt that doesn't count (D10).
	BombIgnored bool
	// Alive is the members alive after a kill or bomb event, bots included (see Alive).
	Alive []fact.Player
	// Flash is a blind that counts (CSC's): an enemy's flash over 1 s on an alive player, past the
	// overlap guard (the player isn't already blind for longer). Blind is the player's flash state
	// after it.
	Flash bool
	Blind Blind
	// FlashAssister has a kill's flash assist (CSC's), when FlashAssisted: an assist the game marks
	// as a flash goes to the assister; any other assist, to the victim's latest flasher while the
	// victim is still blind.
	FlashAssister uint64
	FlashAssisted bool
	// Avenged is the deaths an enemy kill trades, oldest first (D22): the victim's own enemy kills
	// of the killer's teammates in the last 5 s, inclusive, each traded once. A kill avenging any is
	// one trade kill.
	Avenged []Death
}

// Blind is a player's latest counting flash: who threw it and the tick it wears off.
type Blind struct {
	Flasher uint64
	Until   float64
}

// Death is a counted enemy kill's victim.
type Death struct {
	Steam uint64
	Tick  int
}

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

// New returns the state before any round.
func New() *Tracker {
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
		t.deaths[f.Victim.Steam64]++
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
				t.s.TClutch = Clutch{t.s.CTAlive, t.survivor(m.Members(f.Snapshot, fact.SideT))}
			}
			if t.s.CTAlive == 1 && t.s.CTClutch.Size == 0 {
				t.s.CTClutch = Clutch{t.s.TAlive, t.survivor(m.Members(f.Snapshot, fact.SideCT))}
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
	if a.Side == v.Side || blind <= tickRate || !v.Alive || v.FlashDuration >= blind/tickRate+1 {
		return
	}
	c.Flash = true
	b := t.blind[v.Steam64]
	if until := float64(f.Tick) + blind; until > b.Until {
		b = Blind{a.Steam64, until}
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
		&kill{Death: Death{f.Victim.Steam64, f.Tick}, team: *f.Victim.TeamStateID})
}

// actor tells players apart for trades: bots share Steam64 0, so they go by user ID.
func actor(p *fact.Player) uint64 {
	if p.IsBot {
		return 1<<63 | uint64(p.UserID)
	}
	return p.Steam64
}

// Enemies is the oracle's TeamState.ID() comparison, which dereferences both team states.
func Enemies(a, b *fact.Player) (bool, error) {
	if a.TeamStateID == nil || b.TeamStateID == nil {
		return false, fmt.Errorf("no team state for %d or %d", a.Steam64, b.Steam64)
	}
	return *a.TeamStateID != *b.TeamStateID, nil
}

// alive is p alive at the start of the tick with no death this round: entity state is applied
// after the tick's events, so it still shows earlier victims of the tick alive.
func (t *Tracker) alive(p fact.Player) bool {
	return p.Alive && t.deaths[p.Steam64] == 0
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
