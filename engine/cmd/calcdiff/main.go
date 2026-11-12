// Command calcdiff runs the calculators (engine/calc) beside the collectors on committed rounds and
// reports every difference by demo, round, event, row and field. Stat-layer rebuild PRs 6-16 each
// register one family below; PR 18 deletes this with the collectors.
//
//	go run ./engine/cmd/calcdiff [-allow FILE] [-emit FILE] DEMO.dem...
//
// The payload still comes from the collectors: calcdiff replays each round into them (payload.Replay)
// and reads their rows, so a difference is the calculators' alone.
//
// Two kinds of difference are found:
//   - Win placement, always: a round's Wins against its OldWins, where today's hold ran the Win.
//   - Family fields: the new row's family struct against the old row's, for each family in families.
//
// -allow reads the accepted differences, one per line: demo (file name), round, event, row, field,
// class, tab-separated; # starts a comment. A line accepts the difference with that key; never a
// whole round. Event is the buffer index of the RoundEnd for a Win ("win" row, field "at"), and -1
// for a family field, which is a round's sum. Row is "win", "player:<steam64>", "team:T", "team:CT" or "round". -emit
// writes the unlisted differences in that format, to review and add to the allowlist.
//
// Exit status 1 on an unlisted difference or an error.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/payload"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	"github.com/csconfederation/fragg-3.0/model"
)

// A family is one calculator's fields on row.Player: the embedded struct's name ("Combat"), and
// how the old side gets the eco-origin values the payload zeroes on round rows (constants.go):
// eco copies them from the round's model.RoundStats onto the old row. A family without any has nil.
type family struct {
	name string
	team string // the family's struct in row.Team, if it writes team parts
	eco  func(old *row.Player, rs *model.RoundStats)
}

// families are the compared families, in the plan's order: combat, KAST, trades, openings,
// clutches, utility, objectives (+Finish), survival, economy, positioning, swing.
var families = []family{
	{name: "Combat", eco: ecoCombat},
	{name: "KAST", eco: ecoKAST},
	{name: "Trades", eco: ecoTrades},
	{name: "Openings", team: "TeamOpenings", eco: ecoOpenings},
	{name: "Clutches", team: "TeamClutches", eco: ecoClutches},
	{name: "Utility"},
	{name: "Survival", team: "TeamSurvival", eco: func(old *row.Player, rs *model.RoundStats) {
		old.EcoDeathTime, old.EcoTimeAlive = rs.DeathTime, rs.TimeAlive
		if rs.Deaths > 0 && rs.DeathTime < 30.0 {
			old.EarlyDeaths = 1
		}
	}},
	{name: "Economy", team: "TeamEconomy", eco: func(old *row.Player, rs *model.RoundStats) {
		old.EcoKillValue, old.EcoDeathValue = rs.EconImpact, rs.EcoDeathValue
		old.KillsVsLowerBuy, old.KillsVsMuchLowerBuy = rs.LowBuyKills, rs.DisadvantagedBuyKills
	}},
	{name: "Positioning"},
	{name: "Swing", eco: func(old *row.Player, rs *model.RoundStats) {
		old.EcoProbabilitySwing, old.EcoAdjustedKills = rs.ProbabilitySwing, rs.EcoAdjustedKills
		old.TimeToKillSum, old.KillsWithTTK = rs.TimeToKillSum, rs.KillsWithTTK
	}},
}

// ecoCombat is the eco-origin Combat fields as one round's eco row gives them: the per-round form
// of the eco fold's lines for each (the fold sums these over rounds).
func ecoCombat(old *row.Player, rs *model.RoundStats) {
	c := &old.Combat
	c.AssistedKills = 0 // side_stats.go updateSupportStats: Assists when GotAssist
	if rs.GotAssist {
		c.AssistedKills = rs.Assists
	}
	c.AWPMultiKillRounds = b2i(rs.AWPKills >= 2) // updateAWPStats
	c.AWPDeaths = b2i(rs.LostAWP)                // updateAWPStats
	c.RoundsWithAWPKill = b2i(rs.AWPKill)        // updateAWPStats
	c.KnifeKills = b2i(rs.KnifeKill)             // updateTradeStats
	c.PistolVsRifleKills = b2i(rs.PistolVsRifleKill)
	c.UtilityKills = rs.UtilityKills // fold.go FoldRound
	c.PerfectKills = rs.PerfectKills // fold.go FoldRound
}

// ecoKAST is eco's KAST round as the fold counts it (side_stats.go UpdateCommonRoundStats).
func ecoKAST(old *row.Player, rs *model.RoundStats) {
	old.KAST.EcoKastRounds = float64(b2i(rs.GotKill || rs.GotAssist || rs.Survived || rs.Traded))
}

// ecoTrades is the eco-origin Trades fields as one round's eco row gives them (side_stats.go
// updateTradeStats and applySideEcoCounters, fold.go FoldRound).
func ecoTrades(old *row.Player, rs *model.RoundStats) {
	t := &old.Trades
	t.TradeKills, t.FastTrades, t.TradeDenials = rs.TradeKills, rs.FastTrades, rs.TradeDenials
	t.OpeningDeathsTraded = b2i(rs.OpeningDeath && rs.Traded)
	t.SavedByTeammate, t.SavedTeammate = b2i(rs.SavedByTeammate), b2i(rs.SavedTeammate)
}

// ecoOpenings is the eco-origin Openings fields as one round's eco row gives them (side_stats.go
// updateAWPOpeningKills, fold.go FoldRound).
func ecoOpenings(old *row.Player, rs *model.RoundStats) {
	o := &old.Openings
	o.AWPOpeningKills = b2i(rs.AWPOpeningKill)
	o.ManAdvantageKills, o.ManDisadvantageDeaths = rs.ManAdvantageKills, rs.ManDisadvantageDeaths
}

// ecoClutches is the eco-origin clutch attempts as one round's eco row gives them (fold.go FoldRound).
func ecoClutches(old *row.Player, rs *model.RoundStats) {
	c := &old.Clutches
	if !rs.ClutchAttempt {
		return
	}
	switch rs.ClutchSize {
	case 1:
		c.Clutch1v1Attempts = 1
	case 2:
		c.Clutch1v2Attempts = 1
	case 3:
		c.Clutch1v3Attempts = 1
	case 4:
		c.Clutch1v4Attempts = 1
	case 5:
		c.Clutch1v5Attempts = 1
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Differences found.
const (
	classLate    = "s6-late"    // a Win placed after the bomb facts the hold missed (design note section 6)
	classNeutral = "s6-neutral" // a Win placed before facts that don't read it; nothing moves
	classOther   = "s6-other"   // a Win placed before a kill, hurt, flash or bomb fact: unexpected
)

type diff struct {
	demo       string
	round, evt int
	row, field string
	class      string
	old, new   string // the values, for a family field
}

func (d diff) key() string {
	return strings.Join([]string{d.demo, strconv.Itoa(d.round), strconv.Itoa(d.evt), d.row, d.field}, "\t")
}

// report is what a run found.
type report struct {
	allow                        map[string]bool
	demos, skipped, rounds, kept int
	wins, same, tail             int
	classes                      map[string][2]int // win class: listed, unlisted
	rows, fields                 int               // compared
	byField                      map[string][2]int // "family.field": listed, unlisted
	unlisted                     []diff
	shown                        int
}

func (rp *report) add(d diff, bucket map[string][2]int, name string) {
	c := bucket[name]
	if rp.allow[d.key()] {
		c[0]++
	} else {
		c[1]++
		rp.unlisted = append(rp.unlisted, d)
		if rp.shown < 20 {
			rp.shown++
			fmt.Printf("  %s round %d event %d %s %s: %s", d.demo, d.round, d.evt, d.row, d.field, d.class)
			if d.old != "" || d.new != "" {
				fmt.Printf(" old %s new %s", d.old, d.new)
			}
			fmt.Println()
		}
	}
	bucket[name] = c
}

// collect replays each committed round into the collectors and keeps it for the comparison.
type collect struct {
	replay *payload.Replay
	rounds []*round.Round
}

func (c *collect) Round(r *round.Round) error {
	c.rounds = append(c.rounds, r)
	return c.replay.Round(r)
}

// demo runs one demo through the lifecycle, the Deriver and the collectors, then compares.
func (rp *report) demo(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	c := &collect{replay: payload.NewReplay()}
	d := round.NewDeriver(c)
	m := match.New(d)
	term, _ := ingest.Run(context.Background(), f, m)
	res, err := m.Finish(term)
	if err != nil || res.Outcome != match.Ended {
		rp.skipped++ // the payload gates cover status
		return nil
	}
	if err := d.Close(); err != nil {
		return err
	}
	rp.demos++
	kept := map[*match.Round]bool{}
	for _, r := range res.Rounds {
		kept[r] = true
	}
	name := filepath.Base(path)
	for _, r := range c.rounds {
		rp.rounds++
		rp.winDiffs(name, r)
		if kept[r.Key] {
			rp.kept++
			csc, eco := c.replay.Rows(r.Key)
			rp.familyDiffs(name, r, csc, eco.Stats())
		}
	}
	return nil
}

// winDiffs compares each Win's placement with where today's hold ran it. A Win placed later passed
// bomb facts the hold had missed; one placed earlier passed only facts that don't read the Win.
func (rp *report) winDiffs(demo string, r *round.Round) {
	for k, w := range r.OldWins {
		rp.wins++
		switch {
		case w.PostCommit:
			rp.tail++
		case r.Wins[k] == w.At:
			rp.same++
		default:
			class := classLate
			if r.Wins[k] < w.At {
				// Placed earlier: the facts between must not touch anything a Win or a calculator reads.
				class = classNeutral
				for _, e := range r.Events[r.Wins[k]:w.At] {
					switch e.Fact.(type) {
					case fact.Kill, fact.Hurt, fact.Flashed, fact.BombPlanted, fact.BombDefused:
						if !e.Dropped {
							class = classOther
						}
					}
				}
			}
			rp.add(diff{demo: demo, round: r.Num, evt: w.Event, row: "win", field: "at", class: class,
				old: strconv.Itoa(w.At), new: strconv.Itoa(r.Wins[k])}, rp.classes, class)
		}
	}
}

// familyDiffs compares the registered families' fields of each player row.
func (rp *report) familyDiffs(demo string, r *round.Round, csc *cscstats.Round, eco map[uint64]*model.RoundStats) {
	if len(families) == 0 {
		return
	}
	old, cur := payload.RoundRow(csc), calc.Run(r)
	for _, id := range slices.Sorted(maps.Keys(old.Players)) {
		if _, ok := cur.Players[id]; !ok {
			rp.add(diff{demo: demo, round: r.Num, evt: -1, row: playerRow(id), field: "-", class: "missing row"},
				rp.byField, "row")
		}
	}
	for _, id := range slices.Sorted(maps.Keys(cur.Players)) {
		op := old.Players[id]
		if op == nil {
			rp.add(diff{demo: demo, round: r.Num, evt: -1, row: playerRow(id), field: "-", class: "extra row"},
				rp.byField, "row")
			continue
		}
		rp.rows++
		for _, fam := range families {
			if fam.eco != nil {
				if rs := eco[id]; rs != nil {
					fam.eco(op, rs)
				}
			}
			ov := reflect.ValueOf(op).Elem().FieldByName(fam.name)
			nv := reflect.ValueOf(cur.Players[id]).Elem().FieldByName(fam.name)
			rp.fieldDiffs(demo, r.Num, playerRow(id), fam.name, ov, nv)
		}
	}
	for side, clan := range map[fact.Side]string{fact.SideT: r.TClan, fact.SideCT: r.CTClan} {
		ot, nt := csc.TeamStats[clan], cur.Teams[side]
		if ot == nil || nt == nil {
			rp.add(diff{demo: demo, round: r.Num, evt: -1, row: teamRow(side), field: "-", class: "missing row"},
				rp.byField, "row")
			continue
		}
		oldTeam := payload.RoundTeam(ot)
		for _, fam := range families {
			if fam.team != "" {
				rp.fieldDiffs(demo, r.Num, teamRow(side), fam.team, reflect.ValueOf(oldTeam).Elem().FieldByName(fam.team),
					reflect.ValueOf(nt).Elem().FieldByName(fam.team))
			}
		}
	}
}

func teamRow(side fact.Side) string {
	return "team:" + map[fact.Side]string{fact.SideT: "T", fact.SideCT: "CT"}[side]
}

func playerRow(id uint64) string { return "player:" + strconv.FormatUint(id, 10) }

// fieldDiffs compares one family struct field by field, floats exactly: one ULP is a difference.
func (rp *report) fieldDiffs(demo string, num int, rowName, fam string, ov, nv reflect.Value) {
	for i := range ov.NumField() {
		// An internal sum (json:"-") is compared too, by its Go name: the family's eco hook sets the old side's.
		name, _, _ := strings.Cut(ov.Type().Field(i).Tag.Get("json"), ",")
		if name == "-" {
			name = ov.Type().Field(i).Name
		}
		rp.fields++
		if o, n := ov.Field(i), nv.Field(i); !reflect.DeepEqual(o.Interface(), n.Interface()) {
			rp.add(diff{demo: demo, round: num, evt: -1, row: rowName, field: name, class: "value",
				old: fmt.Sprint(o.Interface()), new: fmt.Sprint(n.Interface())}, rp.byField, fam+"."+name)
		}
	}
}

func loadAllow(path string) (map[string]bool, error) {
	allow := map[string]bool{}
	if path == "" {
		return allow, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) != 6 {
			return nil, fmt.Errorf("%s: want 6 tab-separated columns, got %q", path, line)
		}
		allow[strings.Join(cols[:5], "\t")] = true
	}
	return allow, sc.Err()
}

func (rp *report) print() {
	fmt.Printf("demos %d (skipped %d: not a full match), rounds %d (kept %d)\n", rp.demos, rp.skipped, rp.rounds, rp.kept)
	fmt.Printf("wins %d: placed as the hold ran them %d, in a post-commit tail %d (not compared)\n", rp.wins, rp.same, rp.tail)
	for _, class := range []string{classLate, classNeutral, classOther} {
		c := rp.classes[class]
		fmt.Printf("  %s: %d (allowlisted %d, unlisted %d)\n", class, c[0]+c[1], c[0], c[1])
	}
	fmt.Printf("families %d: rows compared %d, fields %d\n", len(families), rp.rows, rp.fields)
	for _, name := range slices.Sorted(maps.Keys(rp.byField)) {
		c := rp.byField[name]
		fmt.Printf("  %s: %d (allowlisted %d, unlisted %d)\n", name, c[0]+c[1], c[0], c[1])
	}
}

func main() {
	allowPath := flag.String("allow", "", "allowlist file")
	emit := flag.String("emit", "", "write the unlisted differences here, in allowlist format")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: calcdiff [-allow FILE] [-emit FILE] DEMO.dem...")
		os.Exit(2)
	}
	allow, err := loadAllow(*allowPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	rp := &report{allow: allow, classes: map[string][2]int{}, byField: map[string][2]int{}}
	bad := false
	for _, p := range flag.Args() {
		if err := rp.demo(p); err != nil {
			fmt.Printf("%s: error: %v\n", p, err)
			bad = true
		}
	}
	rp.print()
	if *emit != "" {
		var b strings.Builder
		for _, d := range rp.unlisted {
			b.WriteString(d.key() + "\t" + d.class + "\n")
		}
		if err := os.WriteFile(*emit, []byte(b.String()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if bad || len(rp.unlisted) > 0 {
		os.Exit(1)
	}
}
