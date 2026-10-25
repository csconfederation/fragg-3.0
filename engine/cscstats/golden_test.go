package cscstats

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// The CI goldens are the oracle's payloads (export/testdata/ci). The engine must match every CSC
// field; the rest isn't produced yet or isn't stored:
//   - eco*, swing_rating, ecoStatsOK: slice 3 (Core-planning#302)
//   - WPAlog, totalWPAlog: Stats discards them (D7)
//   - flags, potentialRound: parser state, not stats
//   - reconnectedPlayers, ConnectedAfterRoundStart: the oracle loses reconnects (#300 delta report)
var skipped = regexp.MustCompile(`^(eco.*|swing_rating|WPAlog|totalWPAlog|reconnectedPlayers|ConnectedAfterRoundStart)$`)

const (
	ciDir          = "../../export/testdata/ci"
	floatTolerance = 1e-9 // float summation order (#300)
)

type ciDemo struct {
	Name      string `json:"name"`
	DemSha256 string `json:"demSha256"`
	Source    string `json:"source"`
	Transform *struct {
		Kind     string  `json:"kind"`
		Fraction float64 `json:"fraction"`
	} `json:"transform"`
	Want struct {
		Golden bool `json:"golden"`
	} `json:"want"`
}

func TestCIGoldens(t *testing.T) {
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	raw, err := os.ReadFile(filepath.Join(ciDir, "demos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Demos []ciDemo }
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	byName := map[string]ciDemo{}
	for _, d := range m.Demos {
		byName[d.Name] = d
	}
	for _, d := range m.Demos {
		if !d.Want.Golden {
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			game, err := run(load(t, dir, d, byName))
			if err != nil {
				t.Fatal(err)
			}
			payload, err := json.Marshal(game)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(payload, &got); err != nil {
				t.Fatal(err)
			}
			want = readGolden(t, filepath.Join(ciDir, "golden", d.Name+".json.gz"))
			var diffs []string
			diff("$", want, got, &diffs)
			if len(diffs) > 0 {
				t.Errorf("%d differences:\n%s", len(diffs), strings.Join(diffs[:min(len(diffs), 40)], "\n"))
			}
		})
	}
}

// TestNaNStillFailsToEncode pins the oracle's mid8364 behavior until the NaN guards land.
func TestNaNStillFailsToEncode(t *testing.T) {
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	data, err := os.ReadFile(filepath.Join(dir, "380338d1be5574aaaec408a9274a8cac5e43d20d4d2d140f3b963965d719ceb1.dem"))
	if err != nil {
		t.Fatal(err)
	}
	game, err := run(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(game); err == nil || !strings.Contains(err.Error(), "NaN") {
		t.Fatalf("encode error = %v, want NaN", err)
	}
}

func run(data []byte) (*Game, error) {
	stats := New()
	m := match.New(stats)
	term, _ := ingest.Run(context.Background(), bytes.NewReader(data), m)
	res, err := m.Finish(term)
	if err != nil {
		return nil, err
	}
	return stats.Game(res), nil
}

func load(t *testing.T, dir string, d ciDemo, byName map[string]ciDemo) []byte {
	t.Helper()
	if d.Transform != nil {
		if d.Transform.Kind != "prefix" {
			t.Fatalf("unsupported transform %q", d.Transform.Kind)
		}
		src := load(t, dir, byName[d.Source], byName)
		return src[:int(float64(len(src))*d.Transform.Fraction)]
	}
	data, err := os.ReadFile(filepath.Join(dir, d.DemSha256+".dem"))
	if err != nil {
		t.Fatalf("%v (run scripts/fetch-ci-demos.sh)", err)
	}
	return data
}

func readGolden(t *testing.T, path string) any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.NewDecoder(zr).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func diff(path string, want, got any, diffs *[]string) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("%s: want object, got %T", path, got))
			return
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			if !skipped.MatchString(k) && !(path == "$" && (k == "flags" || k == "potentialRound")) {
				sorted = append(sorted, k)
			}
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			wv, wok := w[k]
			gv, gok := g[k]
			switch {
			case !wok:
				*diffs = append(*diffs, fmt.Sprintf("%s.%s: unexpected key", path, k))
			case !gok:
				*diffs = append(*diffs, fmt.Sprintf("%s.%s: missing key", path, k))
			default:
				diff(path+"."+k, wv, gv, diffs)
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			*diffs = append(*diffs, fmt.Sprintf("%s: want array of %d, got %T", path, len(w), got))
			return
		}
		for i := range w {
			diff(fmt.Sprintf("%s[%d]", path, i), w[i], g[i], diffs)
		}
	case float64:
		if g, ok := got.(float64); !ok || math.Abs(w-g) > floatTolerance {
			*diffs = append(*diffs, fmt.Sprintf("%s: want %v, got %v", path, w, got))
		}
	default:
		if want != got {
			*diffs = append(*diffs, fmt.Sprintf("%s: want %v, got %v", path, want, got))
		}
	}
}
