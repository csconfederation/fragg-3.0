// Command derivediff runs the baseline's stat pipeline (round.Tracker, the Win hold and the old
// collectors' validation) and round.Deriver on the same match, event by event, and prints every
// difference in state, context or error. PR 18 deletes it with the old pipeline.
//
//	go run ./engine/cmd/derivediff DEMO.dem...
//
// It also checks the flags the old collectors test inline (Dealt, Taken, EcoKill, EcoDamage), which
// the Tracker has none of: from the collectors' own state (CSC's rows, eco's roundActive), and for
// every hurt that moves damage, from how CSC's rows moved.
//
// Exit status 1 on any difference. The old driver sees the hook's events, so it takes a pending
// Win's residual roster from the RoundEnd rather than the lifecycle at the hold's end, and the
// collectors' own outputs are compared by the payload gates, not here.
package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
)

const maxShown = 5 // differences printed per demo

type nopHandover struct{}

func (nopHandover) Round(*round.Round) error { return nil }

type pendingWin struct {
	r     *match.Round
	f     fact.RoundEnd
	e     match.Event
	state round.State
}

// stats drives both pipelines. The old one is the baseline's payload.stats, fed from the hooks.
type stats struct {
	m       *match.Match
	d       *round.Deriver
	tr      *Tracker
	csc     *cscstats.Collector
	eco     *eco.Collector
	cur     *match.Round
	pending *pendingWin
	ecoOpen bool // eco's roundActive: a round has opened
	events  int
	diffs   int
	shown   int
}

func (s *stats) diff(what string, e match.Event, format string, a ...any) {
	s.diffs++
	if s.shown < maxShown {
		s.shown++
		fmt.Printf("  event %d %s %s: %s\n", s.events, fmt.Sprintf("%T", e.Fact), what, fmt.Sprintf(format, a...))
	}
}

func text(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *stats) sameErr(old, new error, what string, e match.Event) {
	if text(old) != text(new) {
		s.diff(what, e, "error old %q, new %q", text(old), text(new))
	}
}

func (s *stats) Open(r *match.Round, snap fact.Snapshot, roster match.Roster) error {
	oerr := s.endWave()
	if oerr == nil {
		s.cur = r
		s.tr.Open(r)
		oerr = s.csc.Open(s.m, r, snap, r.Pistol)
		if oerr == nil {
			st := s.tr.State()
			s.ecoOpen = true
			s.eco.Open(snap, r.Num, append(s.m.Members(snap, fact.SideT), s.m.Members(snap, fact.SideCT)...),
				st.TAlive, st.CTAlive)
		}
	}
	nerr := s.d.Open(r, snap, roster)
	s.sameErr(oerr, nerr, "open", match.Event{})
	return nerr
}

func (s *stats) Commit(r *match.Round, tick int) error {
	oerr := s.csc.Commit(s.m, r, tick, s.tr.State())
	if oerr == nil {
		s.eco.Commit()
	}
	nerr := s.d.Commit(r, tick)
	s.sameErr(oerr, nerr, "commit", match.Event{})
	return nerr
}

// win is the baseline's Win, run where its hold ran it.
func (s *stats) win(w *pendingWin) error {
	s.tr.Win()
	if err := s.csc.Win(s.m, w.r, w.f); err != nil {
		return err
	}
	st := s.tr.State()
	s.eco.Win(w.f.Winner, w.f.Snapshot, s.alive(w.e, w.f.Snapshot), st)
	return nil
}

func (s *stats) alive(e match.Event, snap fact.Snapshot) []fact.Player {
	var out []fact.Player
	for _, side := range []fact.Side{fact.SideT, fact.SideCT} {
		for _, p := range e.Roster.Members(snap, side) {
			if s.tr.IsMember(p.Steam64) && s.tr.AliveP(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func (s *stats) endWave() error {
	w := s.pending
	if w == nil {
		return nil
	}
	s.pending = nil
	return s.win(w)
}

func (s *stats) Event(e match.Event) error {
	s.events++
	var ctx *round.Context
	oerr := s.old(e, &ctx)
	nerr := s.d.Event(e)
	s.sameErr(oerr, nerr, "event", e)
	if oerr == nil && nerr == nil {
		if ctx != nil && !reflect.DeepEqual(*ctx, s.d.Last()) {
			s.diff("context", e, "old %+v\n    new %+v", *ctx, s.d.Last())
		}
		if os, ns := s.tr.State(), s.d.State(); os != ns {
			s.diff("state", e, "old %+v, new %+v", os, ns)
		}
	}
	return nerr
}

// old is the baseline's per-fact path: the explosion Win's hold, then round, eco and CSC.
func (s *stats) old(e match.Event, ctx **round.Context) error {
	if s.pending != nil && (e.Final || endsBombWave(e.Fact) || isRoundEnd(e.Fact)) {
		if err := s.endWave(); err != nil {
			return err
		}
	}
	switch f := e.Fact.(type) {
	case nil, fact.PlayerInfo:
		return nil
	case fact.RoundEnd:
		s.tr.RoundEnd(f)
		if e.Win {
			w := &pendingWin{r: s.cur, f: f, e: e}
			if f.Reason == 1 {
				s.pending = w
				return nil
			}
			return s.win(w)
		}
		return nil
	case fact.TeamChange:
		if e.Join == nil {
			return nil
		}
		s.tr.Join(s.cur, e.Join.Steam64)
		if err := s.csc.Join(s.m, s.cur, *e.Join); err != nil {
			return err
		}
		s.eco.Join(e.Join.Steam64, e.Join.IsBot, e.Join.Side, e.Join.Clan)
		return nil
	case fact.Frame:
		if e.Sample != nil {
			sample := *e.Sample
			f.Sample = func() fact.Snapshot { return sample }
		}
		return s.fact(e, f, ctx)
	}
	if e.Dropped {
		return nil
	}
	return s.fact(e, e.Fact, ctx)
}

func (s *stats) fact(e match.Event, f fact.Fact, ctx **round.Context) error {
	x, err := s.tr.Fact(s.m, f)
	if err != nil {
		return err
	}
	check := s.gates(f, &x)
	*ctx = &x
	if err := s.eco.Fact(f, x); err != nil {
		return err
	}
	err = s.csc.Fact(s.m, f, x)
	if check != nil {
		check(e)
	}
	return err
}

// gates fills the flags the old collectors test inline (the Tracker has none), from their own
// state: CSC's rows, eco's roundActive. For a hurt that moves damage it returns a check run after the
// CSC collector: its rows must have changed exactly as the flags say.
func (s *stats) gates(f fact.Fact, x *round.Context) func(match.Event) {
	switch f := f.(type) {
	case fact.Kill:
		k, v := f.Killer, f.Victim
		x.EcoKill = x.Expected && !x.BombIgnored && k != nil && v != nil &&
			!(k.Steam64 == v.Steam64 && !k.IsBot || k.Side == v.Side)
	case fact.Hurt:
		live := s.m.Live()
		row := func(p *fact.Player) *cscstats.PlayerStats {
			if p == nil {
				return nil
			}
			return s.csc.Row(p.Steam64)
		}
		victim, attacker := row(f.Player), row(f.Attacker)
		x.Taken = live && victim != nil && !x.BombIgnored && x.Enemy
		x.Dealt = live && victim != nil && attacker != nil && f.Player.Side != f.Attacker.Side
		x.EcoDamage = s.ecoOpen && !f.Warmup && x.Enemy && !x.BombIgnored
		if f.HealthDamageTaken <= 0 {
			return nil
		}
		var dealt0, taken0 int
		if attacker != nil {
			dealt0 = attacker.Damage
		}
		if victim != nil {
			taken0 = victim.DamageTaken
		}
		dealt, taken := x.Dealt, x.Taken
		want := func(flag bool) int {
			if flag {
				return f.HealthDamageTaken
			}
			return 0
		}
		return func(e match.Event) {
			if attacker != nil && attacker.Damage-dealt0 != want(dealt) {
				s.diff("gate", e, "Dealt %v but the CSC row's damage moved %d", dealt, attacker.Damage-dealt0)
			}
			if victim != nil && victim.DamageTaken-taken0 != want(taken) {
				s.diff("gate", e, "Taken %v but the CSC row's damage taken moved %d", taken, victim.DamageTaken-taken0)
			}
		}
	}
	return nil
}

func isRoundEnd(f fact.Fact) bool { _, ok := f.(fact.RoundEnd); return ok }

func run(path string) (events, diffs int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	s := &stats{tr: NewTracker(), csc: cscstats.New(), eco: eco.New()}
	s.d = round.NewDeriver(nopHandover{})
	s.m = match.New(s)
	term, _ := ingest.Run(context.Background(), f, s.m)
	_, ferr := s.m.Finish(term)
	if cerr := s.d.Close(); cerr != nil {
		return s.events, s.diffs, cerr
	}
	if ferr != nil {
		fmt.Printf("  match: %v\n", ferr)
	}
	return s.events, s.diffs, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: derivediff DEMO.dem...")
		os.Exit(2)
	}
	bad := false
	for _, p := range os.Args[1:] {
		fmt.Println(p)
		events, diffs, err := run(p)
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			bad = true
		}
		fmt.Printf("  events %d, differences %d\n", events, diffs)
		bad = bad || diffs > 0
	}
	if bad {
		os.Exit(1)
	}
}
