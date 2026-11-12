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
	data *probability.DataCollector
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

func buy(r *round.Round, side fact.Side) teamBuy {
	b := teamBuy{Equip: []int{}, Money: []int{}}
	for _, p := range r.Roster.Members(r.Open, side) {
		b.Equip = append(b.Equip, p.EquipValue)
		b.Money = append(b.Money, p.Money)
	}
	return b
}

// Round takes a committed round's states and kills up to its last Win, as the old CLI did at its
// round snapshot; kills after the win aren't counted.
func (c *collector) Round(r *round.Round) error {
	c.data.RecordRoundStart(0, 0, false, "")
	k := commit{row: roundRow{Num: r.Num, Pistol: r.Pistol, FreezeEnd: r.FreezeEnd,
		T: buy(r, fact.SideT), CT: buy(r, fact.SideCT)}}
	var end round.State
	for i := 0; i <= r.Commit; i++ {
		for _, w := range r.OldWins {
			if w.At == i && !w.PostCommit {
				re := r.Events[w.Event].Fact.(fact.RoundEnd)
				end, k.pending = w.State, c.data.TakePending()
				k.row.Winner, k.row.Reason = int(re.Winner), re.Reason
			}
		}
		if i == r.Commit {
			break
		}
		e := r.Events[i]
		if e.Dropped {
			continue
		}
		snap := func(planted bool) {
			c.data.RecordStateSnapshot(min(e.Before.TAlive, 5), min(e.Before.CTAlive, 5), planted)
		}
		switch f := e.Fact.(type) {
		case fact.Kill:
			if !e.Expected {
				continue
			}
			snap(k.planted)
			if e.Enemy {
				c.data.RecordKill(float64(f.Killer.EquipValue), float64(f.Victim.EquipValue))
			}
		case fact.BombPlanted:
			snap(false)
			k.planted = true
		case fact.BombDefused, fact.BombExplode:
			snap(true)
		}
	}
	k.tAlive, k.ctAlive = min(end.TAlive, 5), min(end.CTAlive, 5)
	c.committed[r.Key] = k
	return nil
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
	c := &collector{data: probability.NewDataCollector(), committed: map[*match.Round]commit{}}
	d := round.NewDeriver(c)
	m := match.New(d)
	term, _ := ingest.Run(context.Background(), f, m)
	res, err := m.Finish(term)
	if err == nil {
		err = d.Close()
	}
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
		c.data.CommitPending(k.pending, k.tAlive, k.ctAlive, k.planted, fact.Side(k.row.Winner), res.Map)
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
