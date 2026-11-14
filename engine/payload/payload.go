// Package payload runs the engine over a demo and builds the payload the worker sends Stats: the
// calculators' rows of the committed rounds, folded once (Core-planning#303, #681).
package payload

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/fold"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// The errors Run returns besides a *match.InvariantError and stream failures. The worker maps them
// to statuses: ErrInvalidFileType 400; ErrNoRounds and ErrUnfinished 422 (D8); everything else 500.
var (
	ErrInvalidFileType = errors.New("not a CS2 demo")
	ErrNoRounds        = errors.New("no valid rounds")
	ErrUnfinished      = errors.New("the demo stops before the match ends")
	ErrDecode          = errors.New("demo data could not be decoded")
)

// Run parses r and returns the payload and the match result. A map the rounds don't cover (D9) is a score-only
// payload; a stream that breaks after the match ended still yields the full payload.
func Run(ctx context.Context, r io.Reader) (*Payload, *match.Result, error) {
	s := &rows{rows: map[*match.Round]*row.Round{}}
	d := round.NewDeriver(s)
	m := match.New(d)
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
	if err := d.Close(); err != nil {
		return nil, nil, err
	}
	if res.Outcome != match.Ended {
		return build(res, nil, nil), res, nil
	}
	rounds := make([]*row.Round, 0, len(res.Rounds))
	for _, r := range res.Rounds {
		rounds = append(rounds, s.rows[r])
	}
	folded, err := fold.Fold(rounds)
	if err != nil {
		return nil, nil, err
	}
	return build(res, folded, rounds), res, nil
}

// rows takes each committed round from the Deriver and keeps its calculated rows, by the lifecycle's
// round: the Deriver also hands over rounds the lifecycle later supersedes, which the fold never reads.
type rows struct{ rows map[*match.Round]*row.Round }

func (s *rows) Round(r *round.Round) error {
	s.rows[r.Key] = calc.Run(r)
	return nil
}
