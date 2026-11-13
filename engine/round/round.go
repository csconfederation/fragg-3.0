// Package round owns the round state the stat collectors share (Core-planning#591): roster,
// alive counts, phase, opening and clutch slots, and which facts count. The Deriver computes a
// Context for each fact, buffers the live ones, and hands each committed round to the collectors;
// they read the contexts and never change the state. It ports CSC's rules unchanged.
package round

import (
	"fmt"
	"maps"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/rating"
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
	Num         int // the open round's number; 0 before the first
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
	// Hurts: Dealt is CSC's damage dealt (both players have rows, on different sides); Taken is its
	// damage taken (the victim has a row, an enemy's, not bomb-ignored). EcoDamage is eco's gate:
	// not warmup, an enemy's, not bomb-ignored, with no membership test. Kills: EcoKill is eco's
	// killer credit: expected, not bomb-ignored, both players known, no suicide or team kill. The
	// eco flags go when D31 unifies the gates.
	Dealt, Taken       bool
	EcoKill, EcoDamage bool
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

// Round is one committed round as the deriver saw it, with everything the collectors read. It
// never refers to the lifecycle's mutable round.
type Round struct {
	// Key identifies the lifecycle's round for the caller; nothing reads through it.
	Key                           *match.Round
	Num, StartTick, InitT, InitCT int
	Pistol, FreezeEnd             bool
	TClan, CTClan, Map            string // Map: ServerInfo's map at the open
	Open                          fact.Snapshot
	Roster                        match.Roster
	OpenMembers                   map[uint64]match.Member // r.Members at the open, before any join
	EndTick                       int
	// Events are the round's live events, then the tail after the commit; joins are Events with Join.
	Events []Event
	// Commit is the index of the first event after the commit.
	Commit int
	// Wins are the round's Wins, one per RoundEnd with Win set, placed by the rule of design note
	// section 6: the index of the event the Win takes effect before. A Win whose RoundEnd is at or
	// after Commit is a tail's, and calculators skip it.
	Wins []int
	End  State // at the commit
	// committed is set by Commit; the deriver hands over only committed rounds.
	committed bool
}

// Win is a Win: the winning side and clan, the RoundEnd's event index and tick, and At, the index
// of the event it takes effect before.
type Win struct {
	Side      fact.Side
	Clan      string
	Event, At int
	Tick      int
}

// WinList is the round's Wins in order, a tail's included.
func (r *Round) WinList() []Win {
	var ws []Win
	for i, e := range r.Events {
		if re, ok := e.Fact.(fact.RoundEnd); ok && e.Win && len(ws) < len(r.Wins) {
			ws = append(ws, Win{Side: re.Winner, Clan: e.Winner, Event: i, At: r.Wins[len(ws)], Tick: re.Snapshot.Tick})
		}
	}
	return ws
}

// WinAt is the latest Win in effect when event i runs, and false before the first.
func (r *Round) WinAt(i int) (Win, bool) {
	var w Win
	var ok bool
	for _, c := range r.WinList() {
		if c.At > i {
			break
		}
		w, ok = c, true
	}
	return w, ok
}

// Disconnected is who left before the commit (D44). Combined with "no counted death" it is a player
// who disconnected alive, who did not survive the round.
func (r *Round) Disconnected() map[uint64]bool {
	left := map[uint64]bool{}
	for _, e := range r.Events[:r.Commit] {
		if d, ok := e.Fact.(fact.Disconnect); ok && !e.Dropped {
			left[d.Steam64] = true
		}
	}
	return left
}

// SideOf is the side of one of the round's two clans, 0 for any other.
func (r *Round) SideOf(clan string) fact.Side {
	switch clan {
	case r.TClan:
		return fact.SideT
	case r.CTClan:
		return fact.SideCT
	}
	return 0
}

// Event is a buffered fact with its context.
type Event struct {
	match.Event
	Context
}

// Handover receives each committed round once, in commit order.
type Handover interface{ Round(r *Round) error }

// Deriver is the lifecycle's stats (match.Stats). It computes each fact's Context in the oracle's
// order, returns the errors the old collectors returned, at the same fact, and buffers the live
// events of the open round. At the next open, or Close, it hands a committed round to the Handover.
type Deriver struct {
	out       Handover
	s         State
	cur       *Round
	members   map[uint64]match.Member // the open round's rows: members at the open and joins
	deaths    map[uint64]int          // counted deaths this round, by actor
	cscDeaths map[uint64]int          // the same by Steam64, as CSC's rows count them
	kills     map[uint64][]*kill      // each killer's enemy kills this round, in order, by actor
	blind     map[uint64]Blind        // each player's latest counting flash this round
	mapName   string
	opened    bool // a round has opened: the eco pipeline is active
	winner    fact.Side
}

// NewDeriver returns the state before any round.
func NewDeriver(out Handover) *Deriver {
	return &Deriver{out: out, s: State{Phase: PrePlant}, members: map[uint64]match.Member{}, deaths: map[uint64]int{},
		cscDeaths: map[uint64]int{}, kills: map[uint64][]*kill{}, blind: map[uint64]Blind{}}
}

// State is the current state.
func (d *Deriver) State() State { return d.s }

func (d *Deriver) Open(r *match.Round, s fact.Snapshot, roster match.Roster) error {
	if err := d.handover(); err != nil {
		return err
	}
	d.s = State{Num: r.Num, Phase: PrePlant, Opening: true, TAlive: r.InitT, CTAlive: r.InitCT}
	d.members = maps.Clone(r.Members)
	d.deaths = map[uint64]int{}
	d.cscDeaths = map[uint64]int{}
	d.kills = map[uint64][]*kill{}
	d.blind = map[uint64]Blind{}
	d.opened, d.winner = true, 0
	d.cur = &Round{Key: r, Num: r.Num, StartTick: r.StartTick, InitT: r.InitT, InitCT: r.InitCT, Pistol: r.Pistol,
		FreezeEnd: r.FreezeEnd, TClan: r.TClan, CTClan: r.CTClan, Map: d.mapName, Open: s, Roster: roster,
		OpenMembers: maps.Clone(r.Members)}
	return nil
}

// Commit is the round's commit. CSC's saves check is the only one left that can fail here.
func (d *Deriver) Commit(r *match.Round, tick int) error {
	for _, id := range slices.Sorted(maps.Keys(d.members)) {
		m := d.members[id]
		if d.cscDeaths[id] == 0 && m.TeamID != int(d.winner) && m.Clan != d.cur.TClan && m.Clan != d.cur.CTClan {
			return fmt.Errorf("round %d: no team stats for %q", r.Num, m.Clan)
		}
	}
	d.cur.committed, d.cur.Commit, d.cur.EndTick, d.cur.End = true, len(d.cur.Events), tick, d.s
	return nil
}

// Close hands over the last round, once the lifecycle has finished.
func (d *Deriver) Close() error { return d.handover() }

func (d *Deriver) handover() error {
	r := d.cur
	d.cur = nil
	if r == nil || !r.committed {
		return nil
	}
	r.placeWins()
	return d.out.Round(r)
}

// placeWins fills Wins: each Win takes effect right after its RoundEnd, or, for an explosion, after
// the last bomb Kill or Hurt of its wave, the events up to the next RoundEnd, the commit (when the
// RoundEnd is before it) or the buffer's end. Dropped events are skipped and nothing is reordered.
func (r *Round) placeWins() {
	r.Wins = nil
	for i, e := range r.Events {
		re, ok := e.Fact.(fact.RoundEnd)
		if !ok || !e.Win {
			continue
		}
		at := i + 1
		if re.Reason == fact.ReasonTargetBombed {
			end := len(r.Events)
			if i < r.Commit {
				end = r.Commit
			}
			for j := i + 1; j < end; j++ {
				if _, ok := r.Events[j].Fact.(fact.RoundEnd); ok {
					break
				}
				if !r.Events[j].Dropped && bombFact(r.Events[j].Fact) {
					at = j + 1
				}
			}
		}
		r.Wins = append(r.Wins, at)
	}
}

// Event applies one fact: the checks the old pipeline made (round, then eco, then CSC), the state,
// and the context. Live events join the open round's buffer.
func (d *Deriver) Event(e match.Event) error {
	var c Context
	switch f := e.Fact.(type) {
	case nil: // Final
		return nil
	case fact.ServerInfo:
		if !e.Dropped {
			d.mapName = f.Map
		}
		return nil
	case fact.PlayerInfo:
		return nil
	case fact.TeamChange:
		if e.Join != nil {
			d.members[e.Join.Steam64] = *e.Join
		}
		c = Context{Before: d.s, After: d.s}
	case fact.RoundEnd:
		c.Before = d.s
		if f.Winner == fact.SideT {
			d.s.TMoney = true
		}
		c.After = d.s
		d.add(e, c)
		if e.Win {
			d.s.Phase, d.winner = Decided, f.Winner
		}
		return nil
	default:
		if e.Dropped {
			d.add(e, c)
			return nil
		}
		c.Before = d.s
		if e.Live {
			if err := d.apply(e, &c); err != nil {
				return err
			}
		}
		c.After = d.s
		switch f := f.(type) {
		case fact.Kill:
			if c.Expected {
				c.Alive = d.Alive(e.Roster, f.Snapshot)
			}
		case fact.BombPlanted:
			c.Alive = d.Alive(e.Roster, f.Snapshot)
		case fact.BombDefused:
			c.Alive = d.Alive(e.Roster, f.Snapshot)
		}
		if err := d.checks(e, c); err != nil {
			return err
		}
	}
	d.add(e, c)
	return nil
}

// add buffers a live event. A Hurt keeps only the snapshot fields the collectors read.
func (d *Deriver) add(e match.Event, c Context) {
	if !e.Live || d.cur == nil {
		return
	}
	if h, ok := e.Fact.(fact.Hurt); ok {
		h.Snapshot = fact.Snapshot{Tick: h.Tick, RoundsPlayed: h.RoundsPlayed, Warmup: h.Warmup}
		e.Fact = h
	}
	d.cur.Events = append(d.cur.Events, Event{e, c})
}

// checks are the eco and CSC steps' errors, after the round step passed (the old pipeline ran its
// collectors in that order).
func (d *Deriver) checks(e match.Event, c Context) error {
	if e.Dropped {
		return nil
	}
	switch f := e.Fact.(type) {
	case fact.BombPlanted:
		if d.opened && f.Player == nil {
			return fmt.Errorf("bomb planted by nobody at tick %d", f.Tick) // the oracle dereferences nil
		}
	case fact.BombDefused:
		if d.opened && f.Player == nil {
			return fmt.Errorf("bomb defused by nobody at tick %d", f.Tick)
		}
	}
	if !e.Live {
		return nil
	}
	switch f := e.Fact.(type) {
	case fact.GrenadeThrow:
		if f.Weapon == nil {
			return fmt.Errorf("grenade without a weapon at tick %d", f.Tick)
		}
	case fact.Kill:
		// A victim whose clan isn't one of the round's teams: a joiner (inferred unreachable).
		if c.Victim && c.Before.Opening && d.cur.InitT+d.cur.InitCT == 10 {
			if clan := d.members[f.Victim.Steam64].Clan; clan != d.cur.TClan && clan != d.cur.CTClan {
				return fmt.Errorf("round %d: no team stats for %q", d.s.Num, clan)
			}
		}
	}
	return nil
}

// bombFact reports whether f is the bomb's damage or kill.
func bombFact(f fact.Fact) bool {
	var w *int
	switch f := f.(type) {
	case fact.Hurt:
		w = f.Weapon
	case fact.Kill:
		w = f.Weapon
	}
	return w != nil && *w == fact.EqBomb
}

// Alive is the open round's members alive in s, T then CT.
func (d *Deriver) Alive(r match.Roster, s fact.Snapshot) []fact.Player {
	var out []fact.Player
	for _, side := range []fact.Side{fact.SideT, fact.SideCT} {
		for _, p := range r.Members(s, side) {
			if d.isMember(p.Steam64) && d.alive(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func (d *Deriver) apply(e match.Event, c *Context) error {
	decided := d.s.Phase == Decided
	switch f := e.Fact.(type) {
	case fact.Kill:
		return d.kill(f, e.Roster, c)
	case fact.Hurt:
		w := -999
		if f.Weapon != nil {
			w = *f.Weapon
		}
		c.BombIgnored = d.bombIgnored(w)
		c.Enemy = f.Attacker != nil && f.Player != nil && f.Attacker.Side != f.Player.Side
		member := f.Player != nil && d.isMember(f.Player.Steam64)
		c.Taken = member && !c.BombIgnored && c.Enemy
		c.Dealt = member && c.Enemy && d.isMember(f.Attacker.Steam64)
		c.EcoDamage = !f.Warmup && c.Enemy && !c.BombIgnored
	case fact.Flashed:
		d.flashed(f, c)
	case fact.BombPlanted:
		if !decided {
			if f.Player == nil {
				return fmt.Errorf("bomb planted by nobody")
			}
			d.s.Phase, d.s.TMoney = PostPlant, true
		}
	case fact.BombDefused:
		if !decided {
			if f.Player == nil {
				return fmt.Errorf("bomb defused by nobody")
			}
			d.s.Phase, d.s.BombDecided = Decided, true
		}
	case fact.BombExplode:
		if !decided {
			d.s.Phase, d.s.BombDecided = Decided, true
		}
	case fact.Disconnect:
		// Only the round's members, as Alive: a mid-round reconnect can be relisted alive without
		// joining the round (S19 Combine mid7214).
		d.s.TAlive, d.s.CTAlive = 0, 0
		for _, p := range e.Roster.Members(f.Snapshot, fact.SideT) {
			if d.isMember(p.Steam64) && d.alive(p) {
				d.s.TAlive++
			}
		}
		for _, p := range e.Roster.Members(f.Snapshot, fact.SideCT) {
			if d.isMember(p.Steam64) && d.alive(p) {
				d.s.CTAlive++
			}
		}
	}
	return nil
}

func isRoundFinalInHalf(round int) bool {
	return round%fact.MR == 0 || (round > fact.MR*2 && round%3 == 0)
}

// bombIgnored is the filter on bomb kills and damage in a half's final round, where a bomb can
// explode after the round was won by elimination. D10: the bomb still counts when it ends the
// round, or when the round is undecided (v5 delivered the bomb's kills before the explosion).
func (d *Deriver) bombIgnored(weapon int) bool {
	return weapon == fact.EqBomb && isRoundFinalInHalf(d.s.Num) && d.s.Phase == Decided && !d.s.BombDecided
}

func (d *Deriver) kill(f fact.Kill, roster match.Roster, c *Context) error {
	// The kill belongs to the open round.
	if d.s.Num != f.RoundsPlayed+1 && (d.s.Num != f.RoundsPlayed || d.s.Phase != Decided) {
		return nil
	}
	c.Expected = true
	c.BombIgnored = d.bombIgnored(*f.Weapon)
	if k, v := f.Killer, f.Victim; !c.BombIgnored && k != nil && v != nil {
		c.EcoKill = !(k.Steam64 == v.Steam64 && !k.IsBot || k.Side == v.Side)
	}
	c.Victim = f.Victim != nil && d.isMember(f.Victim.Steam64) && !c.BombIgnored
	c.Killer = f.Killer != nil && d.isMember(f.Killer.Steam64) && !c.BombIgnored
	if c.Victim {
		d.deaths[Actor(f.Victim)]++
		d.cscDeaths[f.Victim.Steam64]++
		switch f.Victim.Side {
		case fact.SideT:
			d.s.TAlive--
		case fact.SideCT:
			d.s.CTAlive--
		}
		// Clutches (corpus finding 6): a bomb death at the explosion can't start one, and the
		// clutcher is the side's member with no death this round, not the entity's IsAlive(),
		// which lags the kill.
		if d.s.Phase != Decided && *f.Weapon != fact.EqBomb {
			if d.s.TAlive == 1 && d.s.TClutch.Size == 0 {
				d.s.TClutch = Clutch{d.s.CTAlive, d.survivor(roster.Members(f.Snapshot, fact.SideT))}
			}
			if d.s.CTAlive == 1 && d.s.CTClutch.Size == 0 {
				d.s.CTClutch = Clutch{d.s.TAlive, d.survivor(roster.Members(f.Snapshot, fact.SideCT))}
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
		d.s.Opening = false
		d.trade(f, c)
	}
	if c.Victim && f.Assister != nil && d.isMember(f.Assister.Steam64) {
		assist, err := Enemies(f.Assister, f.Victim)
		if err != nil {
			return err
		}
		c.Assist = assist
	}
	if c.Assist {
		if f.AssistedFlash {
			c.FlashAssister, c.FlashAssisted = f.Assister.Steam64, true
		} else if b := d.blind[f.Victim.Steam64]; float64(f.Tick) < b.Until {
			c.FlashAssister, c.FlashAssisted = b.Flasher, true
		}
	}
	return nil
}

// flashed is CSC's flash filter and its latest-flash state (the longest blind wins).
func (d *Deriver) flashed(f fact.Flashed, c *Context) {
	if f.Player == nil || f.Attacker == nil {
		return
	}
	blind := f.Duration * fact.TickRate
	v, a := f.Player, f.Attacker
	// Both must be round members (rows in every collector): a mid-round reconnect can be relisted
	// without joining the round (S19 Combine mid7214), and the oracle dereferences nil for it.
	if !d.isMember(v.Steam64) || !d.isMember(a.Steam64) {
		return
	}
	if a.Side == v.Side || blind <= fact.TickRate || !v.Alive || v.FlashDuration >= blind/fact.TickRate+1 {
		return
	}
	c.Flash = true
	b := d.blind[v.Steam64]
	if until := float64(f.Tick) + blind; until > b.Until {
		b = Blind{a.Steam64, until}
		d.blind[v.Steam64] = b
	}
	c.Blind = b
}

// trade is an enemy kill's trades (D22): the victim's untraded enemy kills of the killer's
// teammates within the window, then the kill itself becomes tradeable.
func (d *Deriver) trade(f fact.Kill, c *Context) {
	for _, k := range d.kills[Actor(f.Victim)] {
		if !k.traded && k.team == *f.Killer.TeamStateID && f.Tick-k.Tick <= rating.TradeWindowTicks {
			k.traded = true
			c.Avenged = append(c.Avenged, k.Death)
		}
	}
	d.kills[Actor(f.Killer)] = append(d.kills[Actor(f.Killer)],
		&kill{Death: Death{f.Victim.Steam64, f.Tick}, team: *f.Victim.TeamStateID})
}

// Actor tells players apart for trades and deaths: bots share Steam64 0, so they go by user ID.
func Actor(p *fact.Player) uint64 {
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
// after the tick's events, so it still shows earlier victims of the tick alive. Deaths go by actor:
// keyed by Steam64, one bot's death made every bot (Steam64 0) dead at a disconnect recount.
func (d *Deriver) alive(p fact.Player) bool {
	return p.Alive && d.deaths[Actor(&p)] == 0
}

// survivor is the last alive member with a round row.
func (d *Deriver) survivor(members []fact.Player) uint64 {
	var id uint64
	for _, p := range members {
		if d.isMember(p.Steam64) && d.alive(p) {
			id = p.Steam64
		}
	}
	return id
}

func (d *Deriver) isMember(steam uint64) bool {
	_, ok := d.members[steam]
	return ok
}
