package roundlog

import (
	"fmt"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/payload"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/model"
)

// Replayed is a fixture run through the Deriver and today's collectors.
type Replayed struct {
	Round *round.Round // the handed-over round; Key is the fixture's own lifecycle round
	Rows  *Rows
}

// Replay feeds the fixture's input to a fresh Deriver and replays the committed round through
// fresh collectors, the way payload.Run does for a whole match.
func Replay(f *File) (*Replayed, error) {
	rp := payload.NewReplay()
	h := &handover{replay: rp}
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
	rows, err := RowsOf(rp, h.got.Key)
	if err != nil {
		return nil, err
	}
	return &Replayed{Round: h.got, Rows: rows}, nil
}

// handover keeps the round the Deriver hands over, and replays it.
type handover struct {
	replay *payload.Replay
	got    *round.Round
}

func (h *handover) Round(r *round.Round) error {
	h.got = r
	return h.replay.Round(r)
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

// Rows are what today's collectors hold for the round: the CSC round with its player and team
// rows, and eco's player rows.
type Rows struct {
	CSC CSC                          `json:"csc"`
	Eco map[uint64]*model.RoundStats `json:"eco"`
}

// CSC is a cscstats.Round with the rows its JSON hides.
type CSC struct {
	Round   *cscstats.Round       `json:"round"`
	Players map[uint64]*player    `json:"players"`
	Teams   map[string]*teamStats `json:"teams"`
}

type player struct {
	*cscstats.PlayerStats
	DamageList map[uint64]int `json:"damageList"`
}

type teamStats struct {
	*cscstats.TeamStats
	Buy cscstats.Buy `json:"buy"`
}

// RowsOf collects the rows of the round with this lifecycle key.
func RowsOf(rp *payload.Replay, key *match.Round) (*Rows, error) {
	c, e := rp.Rows(key)
	if c == nil || e == nil {
		return nil, fmt.Errorf("no rows for the round")
	}
	rows := &Rows{CSC: CSC{Round: c, Players: map[uint64]*player{}, Teams: map[string]*teamStats{}}, Eco: e.Stats()}
	for id, p := range c.PlayerStats {
		rows.CSC.Players[id] = &player{p, p.DamageList}
	}
	for clan, t := range c.TeamStats {
		rows.CSC.Teams[clan] = &teamStats{t, t.Buy}
	}
	return rows, nil
}
