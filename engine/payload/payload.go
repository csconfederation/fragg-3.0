// Package payload runs the engine over a demo and builds the worker's payload: the CSC stats with
// the eco metrics merged in, as the oracle's export.ProcessDemo does (Core-planning#302).
package payload

import (
	"context"
	"io"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// Run parses r and returns the payload and the match result.
func Run(ctx context.Context, r io.Reader) (*cscstats.Game, *match.Result, error) {
	s := &stats{csc: cscstats.New(), eco: eco.New(), rounds: map[*match.Round]*eco.Round{}}
	m := match.New(s)
	term, _ := ingest.Run(ctx, r, m)
	res, err := m.Finish(term)
	if err != nil {
		return nil, nil, err
	}
	game := s.csc.Game(res)
	rounds := make([]*eco.Round, 0, len(res.Rounds))
	for _, r := range res.Rounds {
		rounds = append(rounds, s.rounds[r])
	}
	mergeEco(game, s.eco, rounds, res.Map)
	return game, res, nil
}

// stats drives both pipelines in the oracle's order: per event eco then CSC (eco's handlers are
// registered first); at round phases CSC then eco (CSC's hooks call eco last).
type stats struct {
	csc    *cscstats.Collector
	eco    *eco.Collector
	rounds map[*match.Round]*eco.Round
}

func (s *stats) Open(m *match.Match, r *match.Round, snap fact.Snapshot, pistol bool) error {
	if err := s.csc.Open(m, r, snap, pistol); err != nil {
		return err
	}
	s.eco.Open(snap, r.Num)
	return nil
}

func (s *stats) RoundEnd(m *match.Match, f fact.RoundEnd) error { return s.csc.RoundEnd(m, f) }

func (s *stats) Win(m *match.Match, r *match.Round, f fact.RoundEnd) error {
	if err := s.csc.Win(m, r, f); err != nil {
		return err
	}
	s.eco.Win(f.Winner, f.Snapshot)
	return nil
}

func (s *stats) Commit(m *match.Match, r *match.Round, tick int) error {
	if err := s.csc.Commit(m, r, tick); err != nil {
		return err
	}
	s.rounds[r] = s.eco.Commit()
	return nil
}

func (s *stats) Join(m *match.Match, r *match.Round, mem match.Member) error {
	return s.csc.Join(m, r, mem)
}

func (s *stats) Fact(m *match.Match, f fact.Fact) error {
	if err := s.eco.Fact(f); err != nil {
		return err
	}
	return s.csc.Fact(m, f)
}
