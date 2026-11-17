package roundlog

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// Every fact type has a kind, so a type added to engine/fact is stored, or this fails.
func TestKindTableCoversFacts(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../fact/fact.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var facts []string
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Name.Name == "fact" {
			facts = append(facts, fn.Recv.List[0].Type.(*ast.Ident).Name)
		}
	}
	if len(facts) < 20 {
		t.Fatalf("found only %v", facts)
	}
	for _, name := range facts {
		if _, ok := kinds[name]; !ok {
			t.Errorf("fact %s has no kind", name)
		}
	}
	if len(kinds) != len(facts) {
		t.Errorf("%d kinds for %d facts", len(kinds), len(facts))
	}
}

// Each kind survives encode and decode, a Frame without its function.
func TestKindRoundTrip(t *testing.T) {
	weapon := 7
	p := &fact.Player{Steam64: 1, Name: "a<b>&", Weapons: []int{}, Pos: [3]float64{1.5, -0, 3}}
	snap := fact.Snapshot{Tick: 9, Players: []fact.Player{*p}}
	for name, typ := range kinds {
		f := reflect.New(typ).Elem().Interface().(fact.Fact)
		switch name {
		case "Kill":
			f = fact.Kill{Snapshot: snap, Killer: p, Weapon: &weapon}
		case "Frame":
			f = fact.Frame{Tick: 3, Frame: 4}
		}
		e := Event{match.Event{Fact: f, Live: true}}
		b, err := marshal(e)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var back Event
		if err := back.UnmarshalJSON(b); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(e, back) {
			t.Errorf("%s: %+v came back as %+v", name, e, back)
		}
	}
}

func TestFinalEvent(t *testing.T) {
	b, err := marshal(Event{match.Event{Final: true, Live: true}})
	if err != nil || string(b) != `{"kind":"Final","live":true}` {
		t.Fatalf("%s, %v", b, err)
	}
	var e Event
	if err := e.UnmarshalJSON(b); err != nil || !e.Final || e.Fact != nil {
		t.Fatalf("%+v, %v", e, err)
	}
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	gz, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Gunzip(gz)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Each fixture decodes and encodes to the same bytes, and replays through the Deriver and the
// calculators to the stored rows. -update (shared with the status test) rewrites the rows after an approved output change.
func TestFixtures(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixtures")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw := read(t, path)
			f, err := Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			again, err := Encode(f)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, again) {
				t.Fatal("decode then encode changed the bytes")
			}
			if f.Commit < 0 || f.Commit > len(f.Events) {
				t.Fatalf("commit %d of %d events", f.Commit, len(f.Events))
			}
			got, err := Replay(f)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := marshal(got.Rows)
			if err != nil {
				t.Fatal(err)
			}
			if *update {
				f.Rows = rows
				enc, err := Encode(f)
				if err != nil {
					t.Fatal(err)
				}
				gz, err := Gzip(enc)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, gz, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			if !bytes.Equal(rows, f.Rows) {
				t.Errorf("replayed rows differ from the stored ones (%d vs %d bytes)", len(rows), len(f.Rows))
			}
			if got.Round.Num != f.Round.Num {
				t.Errorf("round %d; fixture round %d", got.Round.Num, f.Round.Num)
			}
		})
	}
}
