package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/roundlog"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// A fixture's round, with the collectors' rows for it.
func fixture(t *testing.T) (*round.Round, *roundlog.Rows) {
	t.Helper()
	gz, err := os.ReadFile("../../roundlog/testdata/mid9217-1-r02.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := roundlog.Gunzip(gz)
	if err != nil {
		t.Fatal(err)
	}
	f, err := roundlog.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := roundlog.Replay(f)
	if err != nil {
		t.Fatal(err)
	}
	return got.Round, got.Rows
}

// damage is a combat calculator in miniature: enemy damage dealt, by the Dealt flag.
func damage(r *round.Round, out *row.Round) {
	for _, e := range r.Events[:r.Commit] {
		if h, ok := e.Fact.(fact.Hurt); ok && e.Dealt {
			out.Players[h.Attacker.Steam64].Damage += h.HealthDamageTaken
		}
	}
}

func wrong(r *round.Round, out *row.Round) {
	for _, p := range out.Players {
		p.Damage = 1
	}
}

func run(t *testing.T, allow map[string]bool, calcs ...calc.Calc) *report {
	t.Helper()
	r, rows := fixture(t)
	keep(t)
	families = []family{{name: "Combat"}}
	calc.All = calcs
	rp := &report{allow: allow, classes: map[string][2]int{}, byField: map[string][2]int{}}
	rp.familyDiffs("demo.dem", r, rows.CSC.Round, rows.Eco)
	return rp
}

// keep restores the registered families when the test ends.
func keep(t *testing.T) {
	f, c := families, calc.All
	t.Cleanup(func() { families, calc.All = f, c })
}

// Every registered family matches the collectors on every fixture round, floats exactly.
func TestFixtures(t *testing.T) {
	files, err := filepath.Glob("../../roundlog/testdata/*.json.gz")
	if err != nil || len(files) < 30 {
		t.Fatalf("%d fixtures, %v", len(files), err)
	}
	for _, path := range files {
		gz, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := roundlog.Gunzip(gz)
		if err != nil {
			t.Fatal(err)
		}
		f, err := roundlog.Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := roundlog.Replay(f)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		rp := &report{classes: map[string][2]int{}, byField: map[string][2]int{}}
		rp.familyDiffs(filepath.Base(path), got.Round, got.Rows.CSC.Round, got.Rows.Eco)
		// A round whose Win is placed after bomb facts the hold missed (design note section 6) moves swing.
		moved := false
		for k, w := range got.Round.OldWins {
			moved = moved || got.Round.Wins[k] != w.At
		}
		var bad []diff
		for _, d := range rp.unlisted {
			if !moved || d.field != "ecoProbabilitySwing" {
				bad = append(bad, d)
			}
		}
		if rp.rows == 0 || len(bad) != 0 {
			t.Errorf("%s: %d rows compared, differences %+v", path, rp.rows, bad)
		}
		if moved {
			// The moved Win is the whole difference: with each Win where the hold ran it, the rows match.
			for k, w := range got.Round.OldWins {
				got.Round.Wins[k] = w.At
			}
			again := &report{classes: map[string][2]int{}, byField: map[string][2]int{}}
			again.familyDiffs(filepath.Base(path), got.Round, got.Rows.CSC.Round, got.Rows.Eco)
			if len(again.unlisted) != 0 {
				t.Errorf("%s: %d differences from the moved Win, %d with the Win where the hold ran it: %+v", path,
					len(rp.unlisted), len(again.unlisted), again.unlisted)
			}
		}
	}
}

// With no family, nothing is compared. A calculator that matches the collectors reports nothing,
// one that doesn't reports each field by row, and an allowlist line accepts only its own key.
func TestFamilyDiffs(t *testing.T) {
	keep(t)
	families, calc.All = nil, nil
	r, rows := fixture(t)
	none := &report{classes: map[string][2]int{}, byField: map[string][2]int{}}
	none.familyDiffs("demo.dem", r, rows.CSC.Round, rows.Eco)
	if none.rows != 0 || none.fields != 0 || len(none.unlisted) != 0 {
		t.Fatalf("no families compared %d rows, %d fields, %d differences", none.rows, none.fields, len(none.unlisted))
	}

	// The miniature calculator writes only damage; the family's other fields differ and are ignored here.
	only := func(rp *report) (ds []diff) {
		for _, d := range rp.unlisted {
			if d.field == "damage" {
				ds = append(ds, d)
			}
		}
		return ds
	}
	if rp := run(t, nil, damage); rp.rows == 0 || rp.fields == 0 || len(only(rp)) != 0 {
		t.Fatalf("the Dealt flag's damage: %d rows, %d fields, differences %+v", rp.rows, rp.fields, only(rp))
	}

	bad := only(run(t, nil, wrong))
	if len(bad) < 2 {
		t.Fatalf("a wrong damage reported %+v", bad)
	}
	d := bad[0]
	if d.evt != -1 || !strings.HasPrefix(d.row, "player:") || d.new != "1" {
		t.Errorf("difference %+v", d)
	}

	allow := map[string]bool{}
	for _, d := range bad[1:] {
		allow[d.key()] = true
	}
	part := run(t, allow, wrong)
	if left := only(part); len(left) != 1 || left[0].key() != d.key() || part.byField["Combat.damage"][0] != len(bad)-1 {
		t.Errorf("allowlisting all but one left %+v", left)
	}
}

func TestWinDiffs(t *testing.T) {
	r, _ := fixture(t)
	rp := &report{classes: map[string][2]int{}, byField: map[string][2]int{}}
	rp.winDiffs("demo.dem", r)
	if rp.wins != 1 || rp.same != 1 || len(rp.unlisted) != 0 {
		t.Errorf("%d wins, %d same, %+v", rp.wins, rp.same, rp.unlisted)
	}
	r.Wins[0]++ // the hold ran it earlier than the rule places it: late
	rp.winDiffs("demo.dem", r)
	if rp.classes[classLate][1] != 1 {
		t.Errorf("classes %v", rp.classes)
	}
}

func TestLoadAllow(t *testing.T) {
	path := t.TempDir() + "/allow.tsv"
	line := "d.dem\t3\t70\twin\tat\ts6-late\n"
	if err := os.WriteFile(path, []byte("# comment\n\n"+line), 0o644); err != nil {
		t.Fatal(err)
	}
	allow, err := loadAllow(path)
	if err != nil || !allow[diff{demo: "d.dem", round: 3, evt: 70, row: "win", field: "at"}.key()] || len(allow) != 1 {
		t.Errorf("%v, %v", allow, err)
	}
	if err := os.WriteFile(path, []byte("d.dem\t3\t70\twin\tat\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAllow(path); err == nil {
		t.Error("a line without a class was accepted")
	}
}
