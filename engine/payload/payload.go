// Package payload runs the engine over a demo and builds the payload the worker sends Stats: the
// CSC and eco stats of the committed rounds, folded once (Core-planning#303).
package payload

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/fold"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
)

// The errors Run returns besides a *match.InvariantError and stream failures. The worker maps them
// to statuses: ErrInvalidFileType 400; ErrNoRounds and ErrUnfinished 422 (D8); everything else 500.
var (
	ErrInvalidFileType = errors.New("not a CS2 demo")
	ErrNoRounds        = errors.New("no valid rounds")
	ErrUnfinished      = errors.New("the demo stops before the match ends")
	ErrDecode          = errors.New("demo data could not be decoded")
)

// Run parses r and returns the payload and the match result. A partial demo (D9) is a score-only
// payload; a stream that breaks after the match ended still yields the full payload.
func Run(ctx context.Context, r io.Reader) (*Payload, *match.Result, error) {
	s := &stats{round: round.New(), csc: cscstats.New(), eco: eco.New(), rounds: map[*match.Round]*eco.Round{}}
	m := match.New(s)
	term, _ := ingest.Run(ctx, r, m)
	res, err := m.Finish(term)
	var inv *match.InvariantError
	switch {
	case err != nil && errors.As(err, &inv):
		return nil, nil, err
	case err != nil && term.Kind == fact.InvalidFileType:
		return nil, nil, fmt.Errorf("%w: %w", ErrInvalidFileType, err)
	case err != nil && term.Kind == fact.Decode:
		return nil, nil, fmt.Errorf("%w: %w", ErrDecode, err)
	case err != nil:
		return nil, nil, err
	case res.Outcome == match.NoRounds:
		return nil, res, ErrNoRounds
	case res.Outcome == match.Unfinished:
		return nil, res, fmt.Errorf("%w: %s", ErrUnfinished, res.Detail)
	}
	game := s.csc.Game(res)
	if res.Outcome == match.Ended {
		rounds := make([]*eco.Round, 0, len(res.Rounds))
		for _, r := range res.Rounds {
			rounds = append(rounds, s.rounds[r])
		}
		if err := fold.Game(game, s.eco, rounds); err != nil {
			return nil, nil, err
		}
	}
	return build(game, res), res, nil
}

// stats drives both pipelines in the oracle's order: per event eco then CSC (eco's handlers are
// registered first); at round phases CSC then eco (CSC's hooks call eco last). The shared round
// state is updated before either sees a phase or fact.
type stats struct {
	round  *round.Tracker
	csc    *cscstats.Collector
	eco    *eco.Collector
	rounds map[*match.Round]*eco.Round
}

func (s *stats) Open(m *match.Match, r *match.Round, snap fact.Snapshot, pistol bool) error {
	s.round.Open(r)
	if err := s.csc.Open(m, r, snap, pistol); err != nil {
		return err
	}
	st := s.round.State()
	s.eco.Open(snap, r.Num, append(m.Members(snap, fact.SideT), m.Members(snap, fact.SideCT)...), st.TAlive, st.CTAlive)
	return nil
}

func (s *stats) RoundEnd(m *match.Match, f fact.RoundEnd) error {
	s.round.RoundEnd(f)
	return nil
}

func (s *stats) Win(m *match.Match, r *match.Round, f fact.RoundEnd) error {
	s.round.Win()
	if err := s.csc.Win(m, r, f); err != nil {
		return err
	}
	s.eco.Win(f.Winner, f.Snapshot)
	return nil
}

func (s *stats) Commit(m *match.Match, r *match.Round, tick int) error {
	if err := s.csc.Commit(m, r, tick, s.round.State()); err != nil {
		return err
	}
	s.rounds[r] = s.eco.Commit()
	return nil
}

func (s *stats) Join(m *match.Match, r *match.Round, mem match.Member) error {
	s.round.Join(r, mem.Steam64)
	if err := s.csc.Join(m, r, mem); err != nil {
		return err
	}
	s.eco.Join(mem.Steam64, mem.Name, mem.IsBot, mem.Side, mem.Clan)
	return nil
}

func (s *stats) Fact(m *match.Match, f fact.Fact) error {
	x, err := s.round.Fact(m, f)
	if err != nil {
		return err
	}
	if err := s.eco.Fact(f); err != nil {
		return err
	}
	return s.csc.Fact(m, f, x)
}
