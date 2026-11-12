package combat_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc"
	"github.com/csconfederation/fragg-3.0/engine/calc/combat"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/payload"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/roundlog"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Every PR 2 fixture: the calculator's Combat against the collectors' rows. The eco-origin fields are
// restored from eco's round rows as the fold reads them (side_stats.go, fold.go).
func TestFixtures(t *testing.T) {
	files, _ := filepath.Glob("../../roundlog/testdata/*.json.gz")
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	for _, path := range files {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".json.gz"), func(t *testing.T) {
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
				t.Fatal(err)
			}
			old, cur := payload.RoundRow(got.Rows.CSC.Round), calc.Run(got.Round)
			for id, op := range old.Players {
				if rs := got.Rows.Eco[id]; rs != nil {
					c := &op.Combat
					c.AssistedKills = 0
					if rs.GotAssist {
						c.AssistedKills = rs.Assists
					}
					c.AWPMultiKillRounds, c.AWPDeaths, c.RoundsWithAWPKill = b2i(rs.AWPKills >= 2), b2i(rs.LostAWP), b2i(rs.AWPKill)
					c.KnifeKills, c.PistolVsRifleKills = b2i(rs.KnifeKill), b2i(rs.PistolVsRifleKill)
					c.UtilityKills, c.PerfectKills = rs.UtilityKills, rs.PerfectKills
				}
				np := cur.Players[id]
				if np == nil {
					t.Fatalf("player %d: no row", id)
				}
				if !reflect.DeepEqual(op.Combat, np.Combat) {
					t.Errorf("player %d:\nold %+v\nnew %+v", id, op.Combat, np.Combat)
				}
			}
			if len(old.Players) != len(cur.Players) {
				t.Errorf("%d rows, want %d", len(cur.Players), len(old.Players))
			}
		})
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Hand-built rounds: players 1-2 T, 11-13 CT; 0 is the shared bot row.
func player(steam uint64) *fact.Player {
	side := fact.SideT
	if steam > 10 {
		side = fact.SideCT
	}
	return &fact.Player{Steam64: steam, Side: side, IsBot: steam == 0}
}

func ev(f fact.Fact, x round.Context) round.Event {
	return round.Event{Event: match.Event{Fact: f}, Context: x}
}

func build(events ...round.Event) *round.Round {
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{0, 1, 2, 11, 12, 13} {
		ms[id] = match.Member{Steam64: id, IsBot: id == 0}
	}
	return &round.Round{OpenMembers: ms, Events: events, Commit: len(events)}
}

func run(r *round.Round) *row.Round {
	out := &row.Round{Players: map[uint64]*row.Player{}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	combat.Calc(r, out)
	return out
}

func hurt(attacker, victim uint64, weapon, dmg int, x round.Context) round.Event {
	return ev(fact.Hurt{Player: player(victim), Attacker: player(attacker), Weapon: &weapon, HealthDamageTaken: dmg}, x)
}

func kill(killer, victim uint64, weapon int, hs bool) fact.Kill {
	return fact.Kill{Killer: player(killer), Victim: player(victim), Weapon: &weapon, Headshot: hs}
}

func TestDamage(t *testing.T) {
	both := round.Context{Dealt: true, Taken: true}
	out := run(build(
		hurt(1, 11, 7, 40, both),
		hurt(1, 11, fact.EqHE, 20, both),
		hurt(2, 11, fact.EqMolotov, 10, both),
		hurt(2, 12, fact.EqBomb, 50, round.Context{}), // ignored or team damage: no flag
		hurt(11, 1, 7, 15, round.Context{Dealt: false, Taken: true}),
	))
	want := row.Combat{Damage: 40 + 20, UtilDmg: 20, NadeDmg: 20, DamageTaken: 15}
	if got := out.Players[1].Combat; got != want {
		t.Errorf("player 1: %+v, want %+v", got, want)
	}
	if got := out.Players[2].Combat; got != (row.Combat{Damage: 10, UtilDmg: 10, InfernoDmg: 10}) {
		t.Errorf("player 2: %+v", got)
	}
	if got := out.Players[11].Combat.DamageTaken; got != 70 {
		t.Errorf("player 11 damage taken %d, want 70", got)
	}
}

// An enemy kill with an assist and a trade: every member has a counted row.
func TestKillAssistTrade(t *testing.T) {
	awp := fact.EqAWP
	f := fact.Kill{Killer: player(11), Victim: player(1), Assister: player(12), Weapon: &awp, Headshot: true}
	f.Victim.Weapons = []int{fact.EqAWP}
	out := run(build(ev(f, round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, Assist: true, EcoKill: true,
		Avenged: []round.Death{{Steam: 13, Tick: 1}}})))
	if got, want := out.Players[11].Combat, (row.Combat{Kills: 1, AwpKills: 1, CtAWP: 1, Hs: 1, RoundsWithAWPKill: 1, PerfectKills: 1}); got != want {
		t.Errorf("killer %+v, want %+v", got, want)
	}
	if got, want := out.Players[12].Combat, (row.Combat{Assists: 1, Eac: 1, SuppRounds: 1, AssistedKills: 1}); got != want {
		t.Errorf("assister %+v, want %+v", got, want)
	}
	if got, want := out.Players[1].Combat, (row.Combat{Deaths: 1, AWPDeaths: 1}); got != want {
		t.Errorf("victim %+v, want %+v", got, want)
	}
	if got := out.Players[13].Combat.Eac; got != 1 {
		t.Errorf("traded player eac %d, want 1", got)
	}
}

// A flash assist goes to the assister when the game marks it, else to the victim's latest flasher
// (D34 is not applied: the assister keeps eac for both).
func TestFlashAssist(t *testing.T) {
	x := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, Assist: true, FlashAssisted: true, FlashAssister: 13}
	f := kill(11, 1, 7, false)
	f.Assister = player(12)
	out := run(build(ev(f, x)))
	if got, want := out.Players[13].Combat, (row.Combat{Eac: 1, SuppRounds: 1}); got != want {
		t.Errorf("flasher %+v, want %+v", got, want)
	}
	f.AssistedFlash = true
	out = run(build(ev(f, x)))
	if got := out.Players[13].Combat; got != (row.Combat{}) {
		t.Errorf("marked flash assist: flasher %+v, want nothing from combat", got)
	}
	if got := out.Players[12].Combat.Eac; got != 1 {
		t.Errorf("assister eac %d, want 1", got)
	}
}

// Supporters are everyone who damaged the victim but the killer; over 60 damage in all counts a round.
func TestSupporters(t *testing.T) {
	d := round.Context{Dealt: true}
	k := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true}
	out := run(build(
		hurt(11, 1, 7, 90, d),
		hurt(12, 1, 7, 40, d),
		ev(kill(11, 1, 7, false), k), // 12 has 40, the killer is skipped
		hurt(12, 2, 7, 30, d),
		hurt(12, 1, 7, 25, d), // 12's damage on 1 is now 65
		ev(kill(13, 1, 7, false), k),
	))
	if got := out.Players[12].Combat; got.SuppDamage != 40+65 || got.SuppRounds != 1 {
		t.Errorf("supporter %+v, want 105 damage and a support round", got)
	}
	if got := out.Players[11].Combat.SuppDamage; got != 90 {
		t.Errorf("player 11 supported with %d, want 90 (the second kill's)", got)
	}
}

// Bots share row 0: a second bot death re-adds the victim's whole damage list, and eco has no bot row.
func TestBotRow(t *testing.T) {
	d := round.Context{Dealt: true}
	k := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, EcoKill: true}
	f := kill(11, 0, fact.EqKnife, true)
	out := run(build(
		hurt(12, 0, 7, 30, d),
		hurt(12, 0, 7, 35, d),
		ev(f, k),
		ev(f, k),
		ev(kill(0, 1, fact.EqHE, true), k),
	))
	if got := out.Players[0].Combat; got != (row.Combat{Deaths: 2, Kills: 1, Hs: 1}) {
		t.Errorf("bot %+v: eco-origin fields must stay 0", got)
	}
	if got := out.Players[12].Combat; got.SuppDamage != 130 || got.SuppRounds != 1 {
		t.Errorf("supporter %+v, want 130 (65 per bot death)", got)
	}
	if got := out.Players[11].Combat; got.KnifeKills != 1 || got.PerfectKills != 2 {
		t.Errorf("killer %+v, want a knife round and 2 perfect kills", got)
	}
}

// EcoKill has no membership test: a killer who isn't in the round gets no row.
func TestNonMemberKiller(t *testing.T) {
	x := round.Context{Expected: true, EcoKill: true}
	out := run(build(ev(kill(99, 1, fact.EqAWP, true), x)))
	if len(out.Players) != 6 || out.Players[99] != nil {
		t.Errorf("players %d, want the 6 members only", len(out.Players))
	}
	if got := out.Players[1].Combat; got != (row.Combat{}) {
		t.Errorf("victim %+v, want nothing: no counted death", got)
	}
}

// A human who joins mid-round has an eco row from the join: an earlier kill was a scratch row's.
func TestLateJoiner(t *testing.T) {
	x := round.Context{Expected: true, EcoKill: true}
	join := ev(fact.TeamChange{}, round.Context{})
	join.Join = &match.Member{Steam64: 99}
	out := calc.Run(build(ev(kill(99, 1, fact.EqKnife, true), x), join, ev(kill(99, 2, fact.EqKnife, true), x)))
	if got := out.Players[99].Combat; got.PerfectKills != 1 || got.KnifeKills != 1 {
		t.Errorf("joiner %+v, want only the kill after the join", got)
	}
}

// Utility, pistol-vs-rifle and AWP multi-kill rounds are eco's gates.
func TestEcoOrigin(t *testing.T) {
	x := round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, EcoKill: true}
	rifle := kill(1, 11, 1, false) // P2000 against a rifle buy
	rifle.Victim.EquipValue = 4000
	eco := kill(1, 12, 1, false)
	eco.Victim.EquipValue = 3500 // not above 3500
	out := run(build(
		ev(rifle, x), ev(eco, x),
		ev(kill(2, 11, fact.EqAWP, false), x),
		ev(kill(2, 12, fact.EqAWP, false), x),
		ev(kill(2, 13, fact.EqMolotov, false), x),
		ev(kill(2, 13, fact.EqIncendiary, false), x),
	))
	if got, want := out.Players[1].Combat.PistolVsRifleKills, 1; got != want {
		t.Errorf("pistol vs rifle %d, want %d", got, want)
	}
	c := out.Players[2].Combat
	if c.AWPMultiKillRounds != 1 || c.RoundsWithAWPKill != 1 || c.UtilityKills != 2 || c.AwpKills != 2 {
		t.Errorf("player 2 %+v", c)
	}
}

// Dropped events are skipped.
func TestDropped(t *testing.T) {
	e := hurt(1, 11, 7, 40, round.Context{Dealt: true, Taken: true})
	e.Dropped = true
	if got := run(build(e)).Players[1].Combat; got != (row.Combat{}) {
		t.Errorf("%+v, want nothing from a dropped event", got)
	}
}
