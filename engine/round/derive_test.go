package round_test

import (
	"os"
	"slices"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/roundlog"
)

// replay runs a PR 2 fixture through the Deriver, as roundlog does.
func replay(t *testing.T, name string) *roundlog.Replayed {
	t.Helper()
	gz, err := os.ReadFile("../roundlog/testdata/" + name + ".json.gz")
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
	return got
}

// Win placement (design note section 6). Expected indices are read off each fixture's events: the
// explosion's RoundEnd and the bomb's Hurts and Kills (weapon 404) after it.
func TestWinPlacement(t *testing.T) {
	cases := []struct {
		fixture, why string
		wins         []int
	}{
		{"mid9223-1-r03", "RoundEnd 85; the bomb's Hurt and Kill at 88, 89 come after a Hurt at 87", []int{90}},
		{"mid9275-0-r23", "RoundEnd 73; a Hurt at 75; no bomb facts", []int{74}},
		{"mid5754-0-r04", "RoundEnd 88; a Hurt at 90; no bomb facts", []int{89}},
		{"mid9217-1-r02", "not an explosion: right after its RoundEnd, 117", []int{118}},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			if r := replay(t, c.fixture).Round; !slices.Equal(r.Wins, c.wins) {
				t.Errorf("%s: Wins %v, want %v", c.why, r.Wins, c.wins)
			}
		})
	}
}

// The flags the calculators read (mid9217-1 round 2: enemy damage and kills, then a team's).
func TestGateFlags(t *testing.T) {
	r := replay(t, "mid9217-1-r02").Round
	type flags struct{ dealt, taken bool }
	cases := []struct {
		event int
		why   string
		want  flags
	}{
		{26, "enemy damage", flags{true, true}},
		{38, "enemy kill", flags{false, false}},
		{39, "team damage", flags{false, false}},
		{41, "team kill", flags{false, false}},
	}
	for _, c := range cases {
		x := r.Events[c.event].Context
		if got := (flags{x.Dealt, x.Taken}); got != c.want {
			t.Errorf("event %d, %s: %+v, want %+v", c.event, c.why, got, c.want)
		}
	}
}
