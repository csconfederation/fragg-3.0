package openings_test

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc/openings"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Players 1-2 T (clan A), 11-13 CT (clan B); 0 is the shared bot row, a T.
func side(id uint64) fact.Side {
	if id > 10 {
		return fact.SideCT
	}
	return fact.SideT
}

func kill(killer, victim uint64, weapon int, x round.Context) round.Event {
	k, v := &fact.Player{Steam64: killer, Side: side(killer), IsBot: killer == 0}, &fact.Player{Steam64: victim, Side: side(victim), IsBot: victim == 0}
	return round.Event{Event: match.Event{Fact: fact.Kill{Killer: k, Victim: v, Weapon: &weapon}}, Context: x}
}

func st(t, ct int, opening bool, phase round.Phase) round.State {
	return round.State{TAlive: t, CTAlive: ct, Opening: opening, Phase: phase}
}

// with is a kill context: the state before and after it.
func with(before, after round.State, flags round.Context) round.Context {
	flags.Before, flags.After = before, after
	return flags
}

var enemy = round.Context{Expected: true, Killer: true, Victim: true, Enemy: true, EcoKill: true}

// run is the family alone, over a row for each member and join.
func run(r *round.Round) *row.Round {
	out := &row.Round{Players: map[uint64]*row.Player{}, Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for id := range r.OpenMembers {
		out.Players[id] = &row.Player{}
	}
	for _, e := range r.Events {
		if e.Join != nil {
			out.Players[e.Join.Steam64] = &row.Player{}
		}
	}
	openings.Calc(r, out)
	return out
}

// build is a 5v5 round that the CT clan B wins, with these events before the commit.
func build(events ...round.Event) *round.Round {
	ms := map[uint64]match.Member{}
	for _, id := range []uint64{0, 1, 2, 11, 12, 13} {
		clan := "A"
		if id > 10 {
			clan = "B"
		}
		ms[id] = match.Member{Steam64: id, IsBot: id == 0, Clan: clan, Side: side(id)}
	}
	end := round.Event{Event: match.Event{Fact: fact.RoundEnd{Winner: fact.SideCT}, Win: true, Winner: "B"}}
	events = append(events, end)
	return &round.Round{OpenMembers: ms, TClan: "A", CTClan: "B", InitT: 5, InitCT: 5, Events: events,
		Commit: len(events), Wins: []int{len(events)}}
}

func TestOpeningKill(t *testing.T) {
	first := with(st(5, 5, true, round.PrePlant), st(5, 4, false, round.PrePlant), enemy)
	second := with(st(5, 4, false, round.PrePlant), st(5, 3, false, round.PrePlant), enemy)
	out := run(build(kill(1, 11, fact.EqAWP, first), kill(1, 12, 7, second), kill(1, 13, 7, second)))
	if got, want := out.Players[1].Openings, (row.Openings{Ok: 1, Entries: 1, Rwk: 1, ThreeK: 1, AWPOpeningKills: 1, ManAdvantageKills: 1}); got != want {
		t.Errorf("killer %+v, want %+v", got, want)
	}
	if got, want := out.Players[11].Openings, (row.Openings{Ol: 1, ManDisadvantageDeaths: 1}); got != want {
		t.Errorf("victim %+v, want %+v", got, want)
	}
	if got := out.Players[12].Openings; got != (row.Openings{}) {
		t.Errorf("second victim %+v: not the opening, not a man-disadvantage death", got)
	}
	// An opening kill after the plant is no entry; a CT's never is.
	post := with(st(5, 5, true, round.PostPlant), st(4, 5, false, round.PostPlant), enemy)
	if got := run(build(kill(11, 1, 7, post))).Players[11].Openings.Entries; got != 0 {
		t.Errorf("CT entry %d", got)
	}
	if got := run(build(kill(1, 11, 7, post))).Players[1].Openings.Entries; got != 0 {
		t.Errorf("post-plant entry %d", got)
	}
}

// The start is the first death of any kind: a team kill makes the victim's team 4v5 and the other
// 5v4, and a later enemy kill's victim in the opening sets the other team's 4v5 too (Q11).
func TestStarts(t *testing.T) {
	teamKill := with(st(5, 5, true, round.PrePlant), st(5, 4, true, round.PrePlant), round.Context{Expected: true, Killer: true, Victim: true})
	out := run(build(kill(12, 11, 7, teamKill)))
	if a, b := out.Teams[fact.SideT], out.Teams[fact.SideCT]; a.FiveVFourS != 1 || a.FourVFiveS != 0 || b.FourVFiveS != 1 || b.FiveVFourS != 0 {
		t.Errorf("T %+v, CT %+v", a.TeamOpenings, b.TeamOpenings)
	}
	if b := out.Teams[fact.SideCT]; b.FourVFiveW != 1 || b.FiveVFourW != 0 {
		t.Errorf("the CT winner started 4v5: %+v", b.TeamOpenings)
	}
	if a := out.Players[12].Openings; a != (row.Openings{}) {
		t.Errorf("team killer %+v: no opening", a)
	}
	again := with(st(5, 4, true, round.PrePlant), st(4, 4, false, round.PrePlant), enemy)
	out = run(build(kill(12, 11, 7, teamKill), kill(11, 1, 7, again)))
	if a := out.Teams[fact.SideT]; a.FourVFiveS != 1 || a.FiveVFourS != 1 {
		t.Errorf("both starts for T: %+v", a.TeamOpenings)
	}
	// A round that didn't start 5v5 has no start.
	r := build(kill(12, 11, 7, teamKill))
	r.InitT = 4
	if a := run(r).Teams[fact.SideCT]; a.FourVFiveS != 0 || a.FourVFiveW != 0 {
		t.Errorf("4v5 start in a 9-player round: %+v", a.TeamOpenings)
	}
	// The T team wins a 5v4 round.
	r = build(kill(12, 11, 7, teamKill))
	r.Events[len(r.Events)-1].Winner = "A"
	r.Events[len(r.Events)-1].Fact = fact.RoundEnd{Winner: fact.SideT}
	if a := run(r).Teams[fact.SideT]; a.FiveVFourW != 1 {
		t.Errorf("T won 5v4: %+v", a.TeamOpenings)
	}
}

func TestMultiKills(t *testing.T) {
	x := with(st(5, 5, false, round.PrePlant), st(5, 4, false, round.PrePlant), enemy)
	var ev []round.Event
	for _, v := range []uint64{11, 12, 13, 11} {
		ev = append(ev, kill(1, v, 7, x))
	}
	ev2 := append([]round.Event{}, ev...)
	if got := run(build(ev...)).Players[1].Openings; got.FourK != 1 || got.TwoK+got.ThreeK+got.FiveK != 0 {
		t.Errorf("%+v", got)
	}
	ev2 = append(ev2, kill(1, 12, 7, x), kill(1, 13, 7, x))
	if got := run(build(ev2...)).Players[1].Openings; got.FourK+got.FiveK != 0 {
		t.Errorf("six kills has no multi-kill flag: %+v", got)
	}
}

// Eco has no row for a bot: its eco-origin fields stay 0, while CSC's opening fields count.
func TestBotAndNonMember(t *testing.T) {
	x := with(st(5, 5, true, round.PrePlant), st(5, 4, false, round.PrePlant), enemy)
	out := run(build(kill(0, 11, fact.EqAWP, x)))
	if got, want := out.Players[0].Openings, (row.Openings{Ok: 1, Entries: 1, Rwk: 1}); got != want {
		t.Errorf("bot %+v, want %+v", got, want)
	}
	// An eco kill without a member killer: the human victim still has a man-disadvantage death.
	nm := with(st(5, 5, true, round.PrePlant), st(5, 4, false, round.PrePlant), round.Context{Expected: true, EcoKill: true})
	out = run(build(kill(5, 11, 7, nm)))
	if got := out.Players[11].Openings.ManDisadvantageDeaths; got != 1 {
		t.Errorf("victim of a non-member: %d, want 1", got)
	}
	if out.Players[5] != nil {
		t.Error("a row for a non-member")
	}
}

// A human who joins mid-round has an eco row from the join.
func TestLateJoiner(t *testing.T) {
	x := with(st(5, 5, false, round.PrePlant), st(5, 4, false, round.PrePlant), enemy)
	r := build(kill(5, 11, 7, x))
	join := round.Event{Event: match.Event{Fact: fact.TeamChange{}, Join: &match.Member{Steam64: 5, Clan: "A"}}}
	r.Events = append([]round.Event{r.Events[0], join, r.Events[0]}, r.Events[1])
	r.Commit, r.Wins = 4, []int{4}
	if got := run(r).Players[5].Openings.ManAdvantageKills; got != 1 {
		t.Errorf("joiner's kills after the join: %d, want 1", got)
	}
}
