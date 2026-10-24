package export

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Golden test over the CI demo set in testdata/ci/demos.json. The demos live in the public
// cscdemos bucket; scripts/fetch-ci-demos.sh stores them as <demSha256>.dem in the directory
// named by FRAGG_CI_DEMOS. Without it the test is skipped.
//
//	FRAGG_CI_DEMOS=/path/to/demos go test ./export -run TestCIDemos [-update]

var update = flag.Bool("update", false, "rewrite testdata/ci/golden from the current output")

const (
	ciDir = "testdata/ci"
	// Two parses of the same demo on demoinfocs v5 differ by float summation order (up to ~1e-14).
	floatTolerance = 1e-9
)

// unstableKeys vary between two v5 parses of the same demo far beyond float noise, so only their
// presence and type are compared (Core-planning #299, corpus determinism report).
var unstableKeys = map[string]bool{"distanceToTeammates": true}

type ciManifest struct {
	Bucket string   `json:"bucket"`
	Demos  []ciDemo `json:"demos"`
}

type ciDemo struct {
	Name         string       `json:"name"`
	Object       string       `json:"object"`
	ObjectSha256 string       `json:"objectSha256"`
	DemSha256    string       `json:"demSha256"`
	Source       string       `json:"source"`
	Transform    *ciTransform `json:"transform"`
	Covers       string       `json:"covers"`
	Skip         string       `json:"skip"`
	Want         ciWant       `json:"want"`
}

// ciTransform derives a synthetic fixture from Source (or from nothing, for zeros).
type ciTransform struct {
	Kind     string  `json:"kind"` // prefix, fill, zeros
	Fraction float64 `json:"fraction"`
	Bytes    int     `json:"bytes"`
	Value    byte    `json:"value"`
}

type ciWant struct {
	Err     string `json:"err"`
	Result  string `json:"result"`
	Encodes bool   `json:"encodes"`
	Golden  bool   `json:"golden"`
}

func TestCIDemos(t *testing.T) {
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	raw, err := os.ReadFile(filepath.Join(ciDir, "demos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m ciManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	byName := map[string]ciDemo{}
	for _, d := range m.Demos {
		byName[d.Name] = d
	}

	for _, d := range m.Demos {
		t.Run(d.Name, func(t *testing.T) {
			if d.Skip != "" {
				t.Skip(d.Skip)
			}
			data := loadCIDemo(t, dir, d, byName)
			game, perr := ProcessDemo(io.NopCloser(bytes.NewReader(data)))

			if got := errKind(perr); got != d.Want.Err {
				t.Errorf("error = %q, want %q (err: %v)", got, d.Want.Err, perr)
			}
			if game == nil {
				t.Fatal("ProcessDemo returned a nil game")
			}
			if game.Result != d.Want.Result {
				t.Errorf("result = %q, want %q", game.Result, d.Want.Result)
			}
			payload, encErr := json.Marshal(game)
			if (encErr == nil) != d.Want.Encodes {
				t.Fatalf("encodes = %v, want %v (err: %v)", encErr == nil, d.Want.Encodes, encErr)
			}
			if !d.Want.Golden {
				return
			}
			path := filepath.Join(ciDir, "golden", d.Name+".json.gz")
			if *update {
				writeGolden(t, path, payload)
				return
			}
			compareGolden(t, path, payload)
		})
	}
}

// errKind mirrors the order in which csgo-demo-worker's classifyParseResult checks errors.
func errKind(err error) string {
	switch {
	case err == nil:
		return ""
	case strings.Contains(err.Error(), "ErrInvalidFileType"):
		return "ErrInvalidFileType"
	case errors.Is(err, ErrNoValidRounds):
		return "ErrNoValidRounds"
	case strings.Contains(err.Error(), "ErrUnexpectedEndOfDemo"):
		return "ErrUnexpectedEndOfDemo"
	default:
		return "other"
	}
}

func loadCIDemo(t *testing.T, dir string, d ciDemo, byName map[string]ciDemo) []byte {
	t.Helper()
	var data []byte
	switch {
	case d.Transform == nil:
		b, err := os.ReadFile(filepath.Join(dir, d.DemSha256+".dem"))
		if err != nil {
			t.Fatalf("%v (run scripts/fetch-ci-demos.sh)", err)
		}
		data = b
	case d.Transform.Kind == "zeros":
		data = make([]byte, d.Transform.Bytes)
	default:
		src := loadCIDemo(t, dir, byName[d.Source], byName)
		n := len(src)
		at := int(float64(n) * d.Transform.Fraction)
		switch d.Transform.Kind {
		case "prefix":
			data = src[:at]
		case "fill":
			data = append([]byte{}, src...)
			for i := at; i < at+d.Transform.Bytes; i++ {
				data[i] = d.Transform.Value
			}
		default:
			t.Fatalf("unknown transform %q", d.Transform.Kind)
		}
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != d.DemSha256 {
		t.Fatalf("sha256 = %x, want %s", sum, d.DemSha256)
	}
	return data
}

func writeGolden(t *testing.T, path string, payload []byte) {
	t.Helper()
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.MarshalIndent(v, "", " ") // map keys come out sorted
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(append(canonical, '\n')); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func compareGolden(t *testing.T, path string, payload []byte) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v (regenerate with -update)", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.NewDecoder(zr).Decode(&want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	var diffs []string
	diffJSON("$", want, got, &diffs)
	if len(diffs) > 0 {
		shown := diffs
		if len(shown) > 40 {
			shown = shown[:40]
		}
		t.Errorf("%d differences from %s:\n%s", len(diffs), path, strings.Join(shown, "\n"))
	}
}

func diffJSON(path string, want, got any, diffs *[]string) {
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
			case unstableKeys[k]:
				if fmt.Sprintf("%T", wv) != fmt.Sprintf("%T", gv) {
					*diffs = append(*diffs, fmt.Sprintf("%s.%s: want %T, got %T", path, k, wv, gv))
				}
			default:
				diffJSON(path+"."+k, wv, gv, diffs)
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			*diffs = append(*diffs, fmt.Sprintf("%s: want array of %d, got %v", path, len(w), describe(got)))
			return
		}
		for i := range w {
			diffJSON(fmt.Sprintf("%s[%d]", path, i), w[i], g[i], diffs)
		}
	case float64:
		g, ok := got.(float64)
		if !ok || math.Abs(w-g) > floatTolerance {
			*diffs = append(*diffs, fmt.Sprintf("%s: want %v, got %v", path, w, got))
		}
	default:
		if want != got {
			*diffs = append(*diffs, fmt.Sprintf("%s: want %v, got %v", path, want, got))
		}
	}
}

func describe(v any) string {
	if a, ok := v.([]any); ok {
		return fmt.Sprintf("array of %d", len(a))
	}
	return fmt.Sprintf("%T", v)
}
