package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

type recorder struct {
	facts []fact.Fact
	stop  func(fact.Fact) error
}

func (r *recorder) Fact(f fact.Fact) error {
	r.facts = append(r.facts, f)
	if r.stop != nil {
		return r.stop(f)
	}
	return nil
}

// ciDemo reads a CI demo by name from FRAGG_CI_DEMOS (scripts/fetch-ci-demos.sh).
func ciDemo(t *testing.T, name string) []byte {
	t.Helper()
	dir := os.Getenv("FRAGG_CI_DEMOS")
	if dir == "" {
		t.Skip("FRAGG_CI_DEMOS not set; run scripts/fetch-ci-demos.sh")
	}
	raw, err := os.ReadFile("../../export/testdata/ci/demos.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Demos []struct{ Name, DemSha256 string }
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, d := range m.Demos {
		if d.Name == name {
			b, err := os.ReadFile(filepath.Join(dir, d.DemSha256+".dem"))
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
	t.Fatalf("no CI demo %q", name)
	return nil
}

func count[T fact.Fact](facts []fact.Fact) (n int, last T) {
	for _, f := range facts {
		if v, ok := f.(T); ok {
			n++
			last = v
		}
	}
	return n, last
}

func TestRunFullMatch(t *testing.T) {
	r := &recorder{}
	term, err := Run(context.Background(), bytes.NewReader(ciDemo(t, "mid9984-cache")), r)
	if err != nil || term.Kind != fact.Clean {
		t.Fatalf("terminal %+v, sink error %v", term, err)
	}
	if _, h := count[fact.Header](r.facts); !strings.HasPrefix(h.ServerName, "CSC Match - 9984 |") {
		t.Errorf("header server name %q", h.ServerName)
	}
	if _, si := count[fact.ServerInfo](r.facts); si.Map != "de_cache" {
		t.Errorf("map %q", si.Map)
	}
	if n, _ := count[fact.WinPanelMatch](r.facts); n != 1 {
		t.Errorf("win panels = %d, want 1", n)
	}
	ended := false
	for _, f := range r.facts {
		if g, ok := f.(fact.GamePhase); ok && g.Phase == fact.GamePhaseEnded {
			ended = true
		}
	}
	if !ended {
		t.Error("no GamePhase ended")
	}
	n, last := count[fact.RoundEnd](r.facts)
	if n < 21 || last.WinnerTeam.Score+last.LoserTeam.Score < 20 {
		t.Errorf("round ends = %d, last %+v", n, last)
	}
	if _, fe := count[fact.FreezeEnd](r.facts); len(fe.Players) < 10 {
		t.Errorf("last freeze-end snapshot has %d players", len(fe.Players))
	}
}

func TestRunTerminals(t *testing.T) {
	demo := ciDemo(t, "mid9984-cache")
	for _, tc := range []struct {
		name string
		data []byte
		want fact.TerminalKind
	}{
		{"truncated", demo[:len(demo)/2], fact.Truncated},
		{"zeros", make([]byte, 1<<20), fact.InvalidFileType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			term, err := Run(context.Background(), bytes.NewReader(tc.data), &recorder{})
			if err != nil || term.Kind != tc.want {
				t.Errorf("terminal %+v, sink error %v; want kind %d", term, err, tc.want)
			}
		})
	}
}

func TestRunStopsOnSinkError(t *testing.T) {
	stop := errors.New("invariant")
	r := &recorder{stop: func(f fact.Fact) error {
		if _, ok := f.(fact.RoundEnd); ok {
			return stop
		}
		return nil
	}}
	term, err := Run(context.Background(), bytes.NewReader(ciDemo(t, "mid9984-cache")), r)
	if !errors.Is(err, stop) || term.Kind != fact.Stopped {
		t.Fatalf("terminal %+v, sink error %v", term, err)
	}
	if _, ok := r.facts[len(r.facts)-1].(fact.RoundEnd); !ok {
		t.Errorf("facts continued after the sink error: last is %T", r.facts[len(r.facts)-1])
	}
}
