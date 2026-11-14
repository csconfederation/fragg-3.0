package payload

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/row"
)

// fields.tsv lists every JSON field of the payload per scope (match, team, player, round) with its
// JSON type, Go type, owning struct and constants. Stats CI checks it against the Prisma columns.
// Regenerate with `go test ./engine/payload -run TestFieldList -update`.
const fieldsFile = "fields.tsv"

// scopeTypes are the payload's row types by scope.
func scopeTypes() map[string]reflect.Type {
	pt := reflect.TypeOf(Payload{})
	elem := func(name string) reflect.Type {
		f, _ := pt.FieldByName(name)
		return f.Type.Elem().Elem()
	}
	return map[string]reflect.Type{
		"match":  pt,
		"team":   reflect.TypeOf(Team{}),
		"player": elem("Players"),
		"round":  reflect.TypeOf(Round{}),
	}
}

var scopeOrder = []string{"match", "team", "player", "round"}

// constantScopes are the constants' scopes by field scope.
var constantScopes = map[string][]string{
	"team":   {scopeTeamMatch},
	"player": {scopeRoundPlayer, scopeBot, scopeSideT, scopeSideCT},
	"round":  {scopeRound},
}

// jsonType is the type of f's value in the payload: number, string, boolean or array, and null
// for a pointer ("string|null"). A `,string` tag option makes a number a string.
func jsonType(sf reflect.StructField) string {
	t := sf.Type
	null := t.Kind() == reflect.Pointer
	if null {
		t = t.Elem()
	}
	var s string
	switch t.Kind() {
	case reflect.Bool:
		s = "boolean"
	case reflect.String:
		s = "string"
	case reflect.Slice:
		s = "array"
	case reflect.Int, reflect.Int64, reflect.Uint64, reflect.Float64:
		s = "number"
		if _, opts, _ := strings.Cut(sf.Tag.Get("json"), ","); slices.Contains(strings.Split(opts, ","), "string") {
			s = "string"
		}
	default:
		panic(fmt.Sprintf("%s: no JSON type for %s", sf.Name, t))
	}
	if null {
		s += "|null"
	}
	return s
}

func generate() string {
	var b strings.Builder
	b.WriteString("json_name\tjson_type\tgo_type\tscope\towner\tconstants\n")
	types := scopeTypes()
	for _, scope := range scopeOrder {
		t := types[scope]
		fs := jsonFields(t) // panics on a name found twice at any depth
		slices.SortFunc(fs, func(a, b field) int { return strings.Compare(a.name, b.name) })
		for _, f := range fs {
			sf := t.FieldByIndex(f.index)
			owner := t
			if len(f.index) > 1 {
				owner = t.FieldByIndex(f.index[:len(f.index)-1]).Type
				if owner.Kind() == reflect.Pointer {
					owner = owner.Elem()
				}
			}
			var cs []string
			for _, cscope := range constantScopes[scope] {
				if v, ok := constants[cscope][f.name]; ok {
					cs = append(cs, fmt.Sprintf("%s=%v", cscope, v))
				}
			}
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\n", f.name, jsonType(sf), sf.Type, scope, owner, strings.Join(cs, ";"))
		}
	}
	return b.String()
}

// listed returns the committed list's names per scope.
func listed(t *testing.T) map[string][]string {
	data, err := os.ReadFile(fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n")[1:] {
		c := strings.Split(line, "\t")
		out[c[3]] = append(out[c[3]], c[0])
	}
	return out
}

func keys(t *testing.T, v any) []string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(m))
}

func equalKeys(t *testing.T, what string, got, want []string) {
	t.Helper()
	got, want = slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("%s: %d keys, want %d; missing %v, extra %v", what, len(got), len(want),
			slices.DeleteFunc(slices.Clone(want), func(k string) bool { return slices.Contains(got, k) }),
			slices.DeleteFunc(slices.Clone(got), func(k string) bool { return slices.Contains(want, k) }))
	}
}

func TestFieldList(t *testing.T) {
	want := generate()
	if *update {
		if err := os.WriteFile(fieldsFile, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s is out of date; regenerate with: go test ./engine/payload -run TestFieldList -update", fieldsFile)
	}

	// The marshalled keys, not the reflected ones: encoding/json drops a name found twice at one
	// depth, so a dropped field fails here.
	names := listed(t)
	for scope, n := range map[string]int{"match": 10, "team": 35, "player": 168, "round": 18} {
		if len(names[scope]) != n {
			t.Errorf("%s has %d fields, want %d", scope, len(names[scope]), n)
		}
	}
	full := &Payload{Teams: []Team{{Name: "a", Team: &row.Team{}}},
		Players: []*row.Player{{}},
		Rounds:  []Round{{Round: &row.Round{}, Players: []*row.Player{{}}}}}
	equalKeys(t, "full match", keys(t, full), names["match"])
	equalKeys(t, "full team", keys(t, full.Teams[0]), names["team"])
	equalKeys(t, "full player", keys(t, full.Players[0]), names["player"])
	equalKeys(t, "full round", keys(t, full.Rounds[0]), names["round"])
	equalKeys(t, "round player", keys(t, full.Rounds[0].Players[0]), names["player"])

	scoreOnly := &Payload{Teams: []Team{{Name: "a"}}} // D9: teams and score only
	equalKeys(t, "score-only match", keys(t, scoreOnly),
		slices.DeleteFunc(slices.Clone(names["match"]), func(k string) bool { return k == "players" || k == "rounds" }))
	equalKeys(t, "score-only team", keys(t, scoreOnly.Teams[0]), []string{"name", "score"})
}

func TestRowTags(t *testing.T) {
	var round []string
	for _, f := range jsonFields(reflect.TypeFor[row.Player]()) {
		if reflect.TypeFor[row.Player]().FieldByIndex(f.index).Tag.Get("fold") == "-" {
			round = append(round, f.name)
		}
	}
	equalKeys(t, "fold:\"-\"", round, []string{"teamENUM", "health", "deathTick", "lurkerBlips",
		"mostRecentFlasher", "mostRecentFlashVal", "ctAWP"})
}

func TestConstants(t *testing.T) {
	for scope, n := range map[string]int{scopeRoundPlayer: 110, scopeBot: 46, scopeSideT: 10, scopeSideCT: 10,
		scopeTeamMatch: 7, scopeRound: 4} {
		if len(constants[scope]) != n {
			t.Errorf("%s: %d constants, want %d", scope, len(constants[scope]), n)
		}
	}
	// Every name exists on its row type, and every value is settable.
	for scope, dst := range map[string]any{scopeRoundPlayer: &row.Player{}, scopeBot: &row.Player{},
		scopeSideT: &row.Player{}, scopeSideCT: &row.Player{}, scopeTeamMatch: &row.Team{}, scopeRound: &row.Round{}} {
		setConstants(dst, scope)
	}
	p := &row.Player{}
	setConstants(p, scopeRoundPlayer)
	if p.Health != 100 {
		t.Errorf("round-player health %d, want 100", p.Health)
	}
}

func TestEngineVersion(t *testing.T) {
	if engineVersion == "" {
		t.Error("engineVersion is empty")
	}
}
