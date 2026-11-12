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

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// fields.tsv lists every JSON field of the payload per scope (match, team, player, round) with its
// Go type, owning struct and constants. Stats CI checks it against the Prisma columns.
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

func generate() string {
	var b strings.Builder
	b.WriteString("json_name\tgo_type\tscope\towner\tconstants\n")
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
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", f.name, sf.Type, scope, owner, strings.Join(cs, ";"))
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
		out[c[2]] = append(out[c[2]], c[0])
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
	for scope, n := range map[string]int{"match": 9, "team": 35, "player": 169, "round": 18} {
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

	partial := &Payload{Teams: []Team{{Name: "a"}}} // D9: teams and score only
	equalKeys(t, "partial match", keys(t, partial),
		slices.DeleteFunc(slices.Clone(names["match"]), func(k string) bool { return k == "players" || k == "rounds" }))
	equalKeys(t, "partial team", keys(t, partial.Teams[0]), []string{"name", "score"})
}

func TestRowTags(t *testing.T) {
	var round []string
	for _, f := range jsonFields(reflect.TypeFor[row.Player]()) {
		if reflect.TypeFor[row.Player]().FieldByIndex(f.index).Tag.Get("fold") == "-" {
			round = append(round, f.name)
		}
	}
	equalKeys(t, "fold:\"-\"", round, []string{"teamENUM", "health", "deathTick", "lurkerBlips",
		"distanceToTeammates", "mostRecentFlasher", "mostRecentFlashVal", "ctAWP"})
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

// The copy loses no field and no value: every JSON name is on both sides, and integers widen.
func TestCopyRow(t *testing.T) {
	src := &cscstats.PlayerStats{Kills: 255, Assists: 28, Deaths: 53, TKills: 200, Rating: 1.25, Name: "n",
		SteamID: "76561198000000000", MostRecentFlasher: 7, Health: 100}
	var p row.Player
	copyRow(&p, src)
	if p.Kills != 255 || p.Assists != 28 || p.Deaths != 53 || p.TKills != 200 || p.Rating != 1.25 || p.Name != "n" ||
		p.SteamID != "76561198000000000" || p.MostRecentFlasher != 7 || p.Health != 100 {
		t.Errorf("copy: %+v", p)
	}
	r := &row.Round{}
	copyRow(r, &cscstats.Round{RoundNum: 66, Planter: 5})
	if r.RoundNum != 66 || r.Planter != 5 {
		t.Errorf("round copy: %+v", r)
	}
	copyRow(&row.Team{}, &cscstats.TeamStats{TR: 3})
}
