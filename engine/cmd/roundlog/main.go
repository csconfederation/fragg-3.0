// Command roundlog writes round-log fixtures: for each requested round of a demo, the deriver's
// input as gzipped JSON with the rows the calculators make from it. It refuses to write a round
// whose fixture doesn't replay to the same buffer and rows.
//
//	go run ./engine/cmd/roundlog [-n] [-o DIR] DEMO.dem ROUND...
//
// ROUND is a number or "all". It prints one line per round: events, commit index, Wins, trades
// after a Win before the commit, and the tail's fact kinds, which is how fixtures are picked. -n
// writes nothing.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/roundlog"
)

var matchID = regexp.MustCompile(`mid\d+-\d`)

// handover keeps the wanted committed rounds.
type handover struct {
	want map[int]bool
	all  bool
	got  map[*match.Round]*round.Round
}

func (h *handover) Round(r *round.Round) error {
	if h.all || h.want[r.Num] {
		h.got[r.Key] = r
	}
	return nil
}

func main() {
	out := flag.String("o", "engine/roundlog/testdata", "output directory")
	dry := flag.Bool("n", false, "check and describe the rounds, write nothing")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: roundlog [-o DIR] DEMO.dem ROUND...")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), flag.Args()[1:], *out, *dry); err != nil {
		fmt.Fprintln(os.Stderr, "roundlog:", err)
		os.Exit(1)
	}
}

func run(path string, rounds []string, out string, dry bool) error {
	want := map[int]bool{}
	all := false
	for _, s := range rounds {
		if s == "all" {
			all = true
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("round %q: %w", s, err)
		}
		want[n] = true
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sum := sha256.New()

	h := &handover{want: want, all: all, got: map[*match.Round]*round.Round{}}
	d := round.NewDeriver(h)
	rec := &roundlog.Recorder{Next: d, Want: func(n int) bool { return all || want[n] }}
	m := match.New(rec)
	term, _ := ingest.Run(context.Background(), io.TeeReader(f, sum), m)
	res, err := m.Finish(term)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := d.Close(); err != nil {
		return err
	}
	if _, err := io.Copy(sum, f); err != nil { // the hash covers the whole file, not what ingest read
		return err
	}
	name := filepath.Base(path)
	id := matchID.FindString(name)
	if id == "" {
		id = strings.TrimSuffix(strings.NewReplacer(" ", "_", ":", "_").Replace(name), ".dem")
	}
	if !dry {
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}
	}
	nums := slices.Sorted(maps.Keys(want))
	if all {
		nums = nil
		for _, r := range res.Rounds {
			nums = append(nums, r.Num)
		}
	}
	for _, n := range nums {
		idx := slices.IndexFunc(res.Rounds, func(r *match.Round) bool { return r.Num == n })
		if idx < 0 {
			return fmt.Errorf("%s: round %d isn't a committed round (outcome %v)", name, n, res.Outcome)
		}
		key := res.Rounds[idx]
		lf, got := rec.Logs[key], h.got[key]
		if lf == nil || got == nil || lf.Commit < 0 {
			return fmt.Errorf("%s: round %d wasn't recorded", name, n)
		}
		lf.Demo, lf.Sha256 = name, hex.EncodeToString(sum.Sum(nil))
		if err := roundlog.Check(lf, got, roundlog.RowsOf(got)); err != nil {
			return fmt.Errorf("%s round %d: %w", name, n, err)
		}
		if dry {
			fmt.Printf("%s r%d\t%s\n", id, n, describe(got))
			continue
		}
		raw, err := roundlog.Encode(lf)
		if err != nil {
			return err
		}
		gz, err := roundlog.Gzip(raw)
		if err != nil {
			return err
		}
		file := filepath.Join(out, fmt.Sprintf("%s-r%02d.json.gz", id, n))
		if err := os.WriteFile(file, gz, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s\t%d bytes (%d raw)\t%s\n", file, len(gz), len(raw), describe(got))
	}
	return nil
}

// describe is the round's selection evidence, read from the handed-over buffer.
func describe(r *round.Round) string {
	var explosions []string
	trades, wins, disconnects, joins := 0, 0, 0, 0
	for i, e := range r.Events[:r.Commit] {
		if _, ok := e.Fact.(fact.Disconnect); ok {
			disconnects++
		}
		if e.Join != nil {
			joins++
		}
		if re, ok := e.Fact.(fact.RoundEnd); ok && e.Win {
			wins++
			if re.Reason == 1 {
				explosions = append(explosions, strconv.Itoa(i))
			}
		}
		if _, ok := e.Fact.(fact.Kill); ok && wins > 0 && len(e.Avenged) > 0 {
			trades++
		}
	}
	tail := map[string]int{}
	for _, e := range r.Events[r.Commit:] {
		tail[strings.TrimPrefix(fmt.Sprintf("%T", e.Fact), "fact.")]++
	}
	var ks []string
	for _, k := range slices.Sorted(maps.Keys(tail)) {
		ks = append(ks, fmt.Sprintf("%s %d", k, tail[k]))
	}
	return fmt.Sprintf("events %d, commit %d; wins %d (explosion events %s); trades after a Win %d; disconnects %d, joins %d; tail [%s]",
		len(r.Events), r.Commit, wins, strings.Join(explosions, ","), trades, disconnects, joins, strings.Join(ks, ", "))
}
