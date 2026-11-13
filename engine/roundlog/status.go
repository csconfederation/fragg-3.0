package roundlog

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
)

// Status is a short fact sequence for the lifecycle and the stat hook, and what Finish returns for
// it: an error, or the outcome. It pins statuses the demo corpus never reaches.
type Status struct {
	Version  int      `json:"version"`
	Name     string   `json:"name"`
	Why      string   `json:"why"`
	Facts    []Event  `json:"facts"` // as ingested; a Frame carries its sample beside it
	Terminal Terminal `json:"terminal"`
	Want     Want     `json:"want"`
}

// Terminal is a fact.Terminal; Err is its text.
type Terminal struct {
	Kind  string        `json:"kind"`
	Frame int           `json:"frame,omitempty"`
	Err   string        `json:"err,omitempty"`
	End   fact.Snapshot `json:"end"`
}

var terminalKinds = []string{"Clean", "Truncated", "Decode", "InvalidFileType", "Stopped", "Failed"}

func (t Terminal) fact() (fact.Terminal, error) {
	for i, k := range terminalKinds {
		if k == t.Kind {
			out := fact.Terminal{Kind: fact.TerminalKind(i), Frame: t.Frame, End: t.End}
			if t.Err != "" {
				out.Err = errors.New(t.Err)
			}
			return out, nil
		}
	}
	return fact.Terminal{}, fmt.Errorf("unknown terminal kind %q", t.Kind)
}

// Want is Finish's result: its error's text, or the outcome and its detail.
type Want struct {
	Error   string `json:"error,omitempty"`
	Outcome string `json:"outcome,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// Run feeds the facts to the lifecycle with the Deriver, as payload.Run does,
// and returns what Finish gives. A fact's error doesn't stop the feed: the lifecycle keeps its
// first error, and the facts after it must not change it.
func (s *Status) Run() (Want, error) {
	term, err := s.Terminal.fact()
	if err != nil {
		return Want{}, err
	}
	d := round.NewDeriver(discard{})
	m := match.New(d)
	for i, e := range s.Facts {
		f := e.Fact
		if fr, ok := f.(fact.Frame); ok {
			if e.Sample == nil {
				return Want{}, fmt.Errorf("fact %d: a Frame without a sample", i)
			}
			sample := *e.Sample
			fr.Sample = func() fact.Snapshot { return sample }
			f = fr
		}
		_ = m.Fact(f)
	}
	res, err := m.Finish(term)
	if err != nil {
		return Want{Error: err.Error()}, nil
	}
	if err := d.Close(); err != nil {
		return Want{Error: err.Error()}, nil
	}
	return Want{Outcome: res.Outcome.String(), Detail: res.Detail}, nil
}

// discard takes the Deriver's rounds and keeps none.
type discard struct{}

func (discard) Round(*round.Round) error { return nil }

// EncodeStatus is the fixture's JSON, uncompressed.
func EncodeStatus(s *Status) ([]byte, error) { return marshal(s) }

// DecodeStatus parses a status fixture's JSON.
func DecodeStatus(b []byte) (*Status, error) {
	var s Status
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.Version != Version {
		return nil, fmt.Errorf("fixture version %d, want %d", s.Version, Version)
	}
	return &s, nil
}
