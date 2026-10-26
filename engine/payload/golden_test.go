package payload

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// testdata/golden holds the engine's payloads for the CI demos (testdata/ci at the root): the oracle's,
// plus the approved changes (Core-planning#302, #303), each recorded in its PR. -update rewrites
// them. Since #303 they are the v2 payload, compared whole.
var update = flag.Bool("update", false, "rewrite testdata/golden from the current output")

const (
	ciDir          = "../../testdata/ci"
	floatTolerance = 1e-9 // float summation order (#300)
)

type ciDemo struct {
	Name      string `json:"name"`
	DemSha256 string `json:"demSha256"`
	Source    string `json:"source"`
	Transform *struct {
		Kind     string  `json:"kind"`
		Fraction float64 `json:"fraction"`
		Bytes    int     `json:"bytes"`
		Value    byte    `json:"value"`
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
	demos := ciDemos(t)
	byName := map[string]ciDemo{}
	for _, d := range demos {
		byName[d.Name] = d
	}
	for _, d := range demos {
		if !d.Want.Golden {
			continue
		}
		t.Run(d.Name, func(t *testing.T) {
			game, _, err := Run(context.Background(), bytes.NewReader(load(t, dir, d, byName)))
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
			path := filepath.Join("testdata", "golden", d.Name+".json.gz")
			if *update {
				writeGolden(t, path, got)
				return
			}
			want = readGolden(t, path)
			var diffs []string
			diff("$", want, got, &diffs)
			if len(diffs) > 0 {
				t.Errorf("%d differences:\n%s", len(diffs), strings.Join(diffs[:min(len(diffs), 40)], "\n"))
			}
		})
	}
}

// TestCIStatuses pins what each CI demo yields: a full payload, a score-only one (D9), or the
// error the worker maps to a status.
func TestCIStatuses(t *testing.T) {
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	want := map[string]error{
		"mid9984-cache": nil, "mid8474-ancient-tie": nil, "mid8364-test-lobby-nan": nil, "truncated-tail": nil,
		"mid9072-inferno-partial": nil, "mid9077-inferno-zero-rounds": ErrNoRounds,
		"mid1598-csgo-header": ErrInvalidFileType, "not-a-demo": ErrInvalidFileType,
		"truncated-mid": ErrUnfinished, "corrupt-mid": ErrDecode,
	}
	demos := ciDemos(t)
	byName := map[string]ciDemo{}
	for _, d := range demos {
		byName[d.Name] = d
	}
	for _, d := range demos {
		t.Run(d.Name, func(t *testing.T) {
			w, ok := want[d.Name]
			if !ok {
				t.Fatal("no expected status")
			}
			p, _, err := Run(context.Background(), bytes.NewReader(load(t, dir, d, byName)))
			if w != nil {
				if !errors.Is(err, w) {
					t.Fatalf("err %v; want %v", err, w)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if full := d.Name != "mid9072-inferno-partial"; p.FullStatsOK != full || (len(p.Players) > 0) != full {
				t.Errorf("fullStatsOK %v with %d player rows; want full %v", p.FullStatsOK, len(p.Players), full)
			}
		})
	}
}

// TestNoKillLobbyEncodes: mid8364, a test lobby with no kills, made the oracle's rating 0/0
// (corpus finding 1). json.Marshal rejects NaN and Inf, so encoding proves there are none.
func TestNoKillLobbyEncodes(t *testing.T) {
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	data, err := os.ReadFile(filepath.Join(dir, "380338d1be5574aaaec408a9274a8cac5e43d20d4d2d140f3b963965d719ceb1.dem"))
	if err != nil {
		t.Fatal(err)
	}
	game, _, err := Run(context.Background(), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(game); err != nil {
		t.Fatal(err)
	}
}

func ciDemos(t *testing.T) []ciDemo {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(ciDir, "demos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Demos []ciDemo }
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m.Demos
}

func load(t *testing.T, dir string, d ciDemo, byName map[string]ciDemo) []byte {
	t.Helper()
	switch {
	case d.Transform == nil:
		data, err := os.ReadFile(filepath.Join(dir, d.DemSha256+".dem"))
		if err != nil {
			t.Fatalf("%v (run scripts/fetch-ci-demos.sh)", err)
		}
		return data
	case d.Transform.Kind == "zeros":
		return make([]byte, d.Transform.Bytes)
	}
	src := load(t, dir, byName[d.Source], byName)
	at := int(float64(len(src)) * d.Transform.Fraction)
	switch d.Transform.Kind {
	case "prefix":
		return src[:at]
	case "fill":
		data := append([]byte{}, src...)
		for i := at; i < at+d.Transform.Bytes; i++ {
			data[i] = d.Transform.Value
		}
		return data
	}
	t.Fatalf("unsupported transform %q", d.Transform.Kind)
	return nil
}

func writeGolden(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", " ") // map keys come out sorted
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(append(b, '\n')); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
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
			sorted = append(sorted, k)
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
