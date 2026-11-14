package roundlog

import (
	"fmt"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Replayed is a fixture run through the Deriver and the calculators.
type Replayed struct {
	Round *round.Round // the handed-over round; Key is the fixture's own lifecycle round
	Rows  *Rows        // the calculators' rows for Round
}

// Replay feeds the fixture's input to a fresh Deriver, the way payload.Run does for a whole match.
func Replay(f *File) (*Replayed, error) {
	h := &handover{}
	d := round.NewDeriver(h)
	mr := f.Round.Lifecycle()
	if err := d.Event(match.Event{Fact: fact.ServerInfo{Map: f.Map}}); err != nil {
		return nil, err
	}
	if err := d.Open(mr, f.Open, f.Roster); err != nil {
		return nil, err
	}
	commit := func() error { return d.Commit(mr, f.EndTick) }
	for i, e := range f.Events {
		if i == f.Commit {
			if err := commit(); err != nil {
				return nil, err
			}
		}
		if err := d.Event(e.Event); err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
	}
	if f.Commit == len(f.Events) {
		if err := commit(); err != nil {
			return nil, err
		}
	}
	if err := d.Close(); err != nil {
		return nil, err
	}
	if h.got == nil {
		return nil, fmt.Errorf("the round was not handed over")
	}
	return &Replayed{Round: h.got, Rows: RowsOf(h.got)}, nil
}

// handover keeps the round the Deriver hands over.
type handover struct{ got *round.Round }

func (h *handover) Round(r *round.Round) error {
	h.got = r
	return nil
}

// Lifecycle is the match.Round the round was opened with.
func (r Round) Lifecycle() *match.Round {
	mr := &match.Round{Num: r.Num, StartTick: r.StartTick, Pistol: r.Pistol, FreezeEnd: r.FreezeEnd,
		InitT: r.InitT, InitCT: r.InitCT, TClan: r.TClan, CTClan: r.CTClan, Members: map[uint64]match.Member{}}
	for _, m := range r.Members {
		mr.Members[m.Steam64] = m
	}
	return mr
}

// roundOf is the fixture's Round for a lifecycle round at its open.
func roundOf(mr *match.Round) Round {
	r := Round{Num: mr.Num, StartTick: mr.StartTick, Pistol: mr.Pistol, FreezeEnd: mr.FreezeEnd, InitT: mr.InitT,
		InitCT: mr.InitCT, TClan: mr.TClan, CTClan: mr.CTClan}
	for _, m := range mr.Members {
		r.Members = append(r.Members, m)
	}
	slices.SortFunc(r.Members, func(a, b match.Member) int { return cmpU(a.Steam64, b.Steam64) })
	return r
}

func cmpU(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Rows are the calculators' rows for a round: the round row, and its player and team rows, which the
// round row's JSON hides.
type Rows struct {
	Round   *row.Round              `json:"round"`
	Players map[uint64]*row.Player  `json:"players"`
	Teams   map[fact.Side]*row.Team `json:"teams"`
}

// RowsOf is calc.Run of r.
func RowsOf(r *round.Round) *Rows {
	out := calc.Run(r)
	return &Rows{Round: out, Players: out.Players, Teams: out.Teams}
}
