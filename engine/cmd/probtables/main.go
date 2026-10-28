// Command probtables runs the engine over a demo and records, for its committed rounds, what swing's
// probability tables are built from (rating/probability's DataCollector: alive-count states before
// each kill and bomb event, kill equipment matchups, the winner) and each team's equipment at freeze
// end, for buy-type classification (Core-planning#590). It writes OUT.prob.json and OUT.rounds.json
// and prints one JSON status line.
//
//	go run ./engine/cmd/probtables DEMO.dem OUT
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/rating/probability"
)

// teamBuy is one team's equipment at the round's last open.
type teamBuy struct {
	Equip []int `json:"equip"` // each listed member's equipment value
	Money []int `json:"money"` // each member's money left
}

type roundRow struct {
	Num    int  `json:"num"`
	Pistol bool `json:"pistol"` // the lifecycle's pistol flag: both sides at $800
	// FreezeEnd is the last open being the freeze-end one; otherwise the round committed from its
	// early open and the equipment is from before the buy.
	FreezeEnd bool    `json:"freezeEnd"`
	Winner    int     `json:"winner"` // 2 T, 3 CT
	Reason    int     `json:"reason"`
	T         teamBuy `json:"t"`
	CT        teamBuy `json:"ct"`
}

type collector struct {
	round   *round.Tracker
	data    *probability.DataCollector
	mapName string
	planted bool
	freeze  bool // a freeze-end fact is being processed: the open it causes is post-buy
	cur     roundRow
	pending probability.PendingRound
	end     round.State
	// committed holds each commit until Finish says which rounds count: an in-demo restore (D12)
	// supersedes rounds after they commit.
	committed map[*match.Round]commit
}

type commit struct {
	pending         probability.PendingRound
	tAlive, ctAlive int
	planted         bool
	row             roundRow
}

func buy(m *match.Match, s fact.Snapshot, side fact.Side) teamBuy {
	b := teamBuy{Equip: []int{}, Money: []int{}}
	for _, p := range m.Members(s, side) {
		b.Equip = append(b.Equip, p.EquipValue)
		b.Money = append(b.Money, p.Money)
	}
	return b
}

func (c *collector) Open(m *match.Match, r *match.Round, s fact.Snapshot, pistol bool) error {
	c.round.Open(r)
	c.planted = false
	c.data.RecordRoundStart(0, 0, false, "")
	c.cur = roundRow{Num: r.Num, Pistol: pistol, FreezeEnd: c.freeze,
		T: buy(m, s, fact.SideT), CT: buy(m, s, fact.SideCT)}
	return nil
}

func (c *collector) RoundEnd(m *match.Match, f fact.RoundEnd) error {
	c.round.RoundEnd(f)
	return nil
}

// Win takes the round's states and kills, as the old CLI did at its round snapshot; kills after the
// win aren't counted.
func (c *collector) Win(m *match.Match, r *match.Round, f fact.RoundEnd) error {
	c.round.Win()
	c.end = c.round.State()
	c.pending = c.data.TakePending()
	c.cur.Winner, c.cur.Reason = int(f.Winner), f.Reason
	return nil
}

func (c *collector) Commit(m *match.Match, r *match.Round, tick int) error {
	c.committed[r] = commit{c.pending, min(c.end.TAlive, 5), min(c.end.CTAlive, 5), c.planted, c.cur}
	return nil
}

func (c *collector) Join(m *match.Match, r *match.Round, mem match.Member) error {
	c.round.Join(r, mem.Steam64)
	return nil
}

func (c *collector) Fact(m *match.Match, f fact.Fact) error {
	if si, ok := f.(fact.ServerInfo); ok {
		c.mapName = si.Map
	}
	x, err := c.round.Fact(m, f)
	if err != nil {
		return err
	}
	snap := func(planted bool) {
		c.data.RecordStateSnapshot(min(x.Before.TAlive, 5), min(x.Before.CTAlive, 5), planted)
	}
	switch f := f.(type) {
	case fact.Kill:
		if !x.Expected {
			return nil
		}
		snap(c.planted)
		if x.Enemy {
			c.data.RecordKill(float64(f.Killer.EquipValue), float64(f.Victim.EquipValue))
		}
	case fact.BombPlanted:
		snap(false)
		c.planted = true
	case fact.BombDefused, fact.BombExplode:
		snap(true)
	}
	return nil
}

// sink marks freeze-end facts before the lifecycle sees them, since Open doesn't say what caused it.
type sink struct {
	m *match.Match
	c *collector
}

func (s sink) Fact(f fact.Fact) error {
	_, s.c.freeze = f.(fact.FreezeEnd)
	return s.m.Fact(f)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: probtables DEMO.dem OUT")
		os.Exit(2)
	}
	status := map[string]any{"demo": os.Args[1]}
	defer func() { json.NewEncoder(os.Stdout).Encode(status) }()
	f, err := os.Open(os.Args[1])
	if err != nil {
		status["error"] = err.Error()
		return
	}
	defer f.Close()
	c := &collector{round: round.New(), data: probability.NewDataCollector(), committed: map[*match.Round]commit{}}
	m := match.New(c)
	term, _ := ingest.Run(context.Background(), f, sink{m, c})
	res, err := m.Finish(term)
	if err != nil {
		status["error"] = err.Error()
		return
	}
	status["outcome"], status["map"], status["rounds"] = res.Outcome.String(), res.Map, len(res.Rounds)
	if res.Outcome != match.Ended {
		return // only whole matches feed the tables
	}
	rows := []roundRow{}
	for _, r := range res.Rounds {
		k := c.committed[r]
		c.data.CommitPending(k.pending, k.tAlive, k.ctAlive, k.planted, fact.Side(k.row.Winner), c.mapName)
		rows = append(rows, k.row)
	}
	if err := c.data.SaveToFile(os.Args[2] + ".prob.json"); err != nil {
		status["error"] = err.Error()
		return
	}
	b, err := json.Marshal(map[string]any{"map": res.Map, "rounds": rows})
	if err == nil {
		err = os.WriteFile(os.Args[2]+".rounds.json", b, 0o644)
	}
	if err != nil {
		status["error"] = err.Error()
	}
}
