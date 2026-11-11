package payload

import (
	"maps"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
)

// replay drives today's collectors from each committed round's buffer, in the oracle's order: per
// event eco then CSC; at round phases CSC then eco. It goes until the calculators replace the
// collectors (PR 18).
type replay struct {
	csc    *cscstats.Collector
	eco    *eco.Collector
	rounds map[*match.Round]*eco.Round   // by the lifecycle's round
	seen   map[*match.Round]*match.Round // the round each collector knows, by the lifecycle's round
	roster match.Roster                  // the current event's, for the collector's Members
}

func newReplay() *replay {
	return &replay{csc: cscstats.New(), eco: eco.New(), rounds: map[*match.Round]*eco.Round{},
		seen: map[*match.Round]*match.Round{}}
}

// views is res with the rounds the CSC collector holds.
func (p *replay) views(res *match.Result) *match.Result {
	v := &match.Result{Rounds: make([]*match.Round, 0, len(res.Rounds))}
	for _, r := range res.Rounds {
		v.Rounds = append(v.Rounds, p.seen[r])
	}
	return v
}

// The collector's view of the lifecycle: buffered events are live and in a round.
func (p *replay) Live() bool    { return true }
func (p *replay) InRound() bool { return true }
func (p *replay) Members(s fact.Snapshot, side fact.Side) []fact.Player {
	return p.roster.Members(s, side)
}

// Round replays one committed round. A round's own copy stands in for the lifecycle's, whose members
// gain joins after the open.
func (p *replay) Round(r *round.Round) error {
	mr := &match.Round{Num: r.Num, StartTick: r.StartTick, Pistol: r.Pistol, FreezeEnd: r.FreezeEnd,
		InitT: r.InitT, InitCT: r.InitCT, TClan: r.TClan, CTClan: r.CTClan, Members: maps.Clone(r.OpenMembers)}
	p.seen[r.Key] = mr

	p.roster = r.Roster
	if err := p.csc.Open(p, mr, r.Open, r.Pistol); err != nil {
		return err
	}
	// eco's map is the latest ServerInfo, as it was at the open.
	if err := p.eco.Fact(fact.ServerInfo{Map: r.Map}, round.Context{}); err != nil {
		return err
	}
	p.eco.Open(r.Open, r.Num, append(r.Roster.Members(r.Open, fact.SideT), r.Roster.Members(r.Open, fact.SideCT)...),
		r.InitT, r.InitCT)

	wins := func(i int, post bool) error {
		for _, w := range r.OldWins {
			if w.At != i || w.PostCommit != post {
				continue
			}
			e := r.Events[w.Event]
			re := e.Fact.(fact.RoundEnd)
			mr.WinnerClan = e.Winner
			if err := p.csc.Win(p, mr, re); err != nil {
				return err
			}
			p.eco.Win(re.Winner, re.Snapshot, w.Alive, w.State)
		}
		return nil
	}
	for i := 0; i <= len(r.Events); i++ {
		if err := wins(i, false); err != nil {
			return err
		}
		if i == r.Commit {
			if err := p.csc.Commit(p, mr, r.EndTick, r.End); err != nil {
				return err
			}
			p.rounds[r.Key] = p.eco.Commit()
		}
		if err := wins(i, true); err != nil {
			return err
		}
		if i == len(r.Events) {
			break
		}
		if err := p.event(mr, &r.Events[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *replay) event(mr *match.Round, e *round.Event) error {
	if e.Dropped {
		return nil
	}
	p.roster = e.Roster
	f := e.Fact
	switch f := f.(type) {
	case fact.RoundEnd:
		return nil
	case fact.TeamChange:
		if e.Join == nil {
			return nil
		}
		if err := p.csc.Join(p, mr, *e.Join); err != nil {
			return err
		}
		p.eco.Join(e.Join.Steam64, e.Join.IsBot, e.Join.Side, e.Join.Clan)
		return nil
	case fact.Frame:
		f.Sample = func() fact.Snapshot { return *e.Sample }
		return p.fact(f, e.Context)
	}
	return p.fact(f, e.Context)
}

func (p *replay) fact(f fact.Fact, x round.Context) error {
	if err := p.eco.Fact(f, x); err != nil {
		return err
	}
	return p.csc.Fact(p, f, x)
}
