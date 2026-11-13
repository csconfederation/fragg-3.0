package clutches_test

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/calc/clutches"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Players 1-2 T (clan A), 11-12 CT (clan B); 0 is the shared bot row.
func run(winner fact.Side, t, ct round.Clutch) *row.Round {
	clan := map[fact.Side]string{fact.SideT: "A", fact.SideCT: "B"}[winner]
	end := round.Event{Event: match.Event{Fact: fact.RoundEnd{Winner: winner}, Win: true, Winner: clan}}
	r := &round.Round{TClan: "A", CTClan: "B", Events: []round.Event{end}, Commit: 1, Wins: []int{1},
		End: round.State{TClutch: t, CTClutch: ct}, OpenMembers: map[uint64]match.Member{}}
	out := &row.Round{Players: map[uint64]*row.Player{}, Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	for _, id := range []uint64{0, 1, 2, 11, 12} {
		out.Players[id] = &row.Player{}
	}
	clutches.Calc(r, out)
	return out
}

func TestWonAndAttempts(t *testing.T) {
	out := run(fact.SideT, round.Clutch{Size: 3, Steam: 1}, round.Clutch{Size: 1, Steam: 11})
	if got, want := out.Players[1].Clutches, (row.Clutches{Cl3: 1, Clutch1v3Attempts: 1}); got != want {
		t.Errorf("winning clutcher %+v, want %+v", got, want)
	}
	if got, want := out.Players[11].Clutches, (row.Clutches{Clutch1v1Attempts: 1}); got != want {
		t.Errorf("losing clutcher %+v: an attempt, not a win", got)
	}
	if !(out.Teams[fact.SideT].Clutches == 1 && out.Teams[fact.SideCT].Clutches == 0) {
		t.Errorf("teams %+v %+v, want the winner's clutch", out.Teams[fact.SideT], out.Teams[fact.SideCT])
	}
	out = run(fact.SideCT, round.Clutch{Size: 2, Steam: 1}, round.Clutch{Size: 5, Steam: 11})
	if got, want := out.Players[11].Clutches, (row.Clutches{Cl5: 1, Clutch1v5Attempts: 1}); got != want {
		t.Errorf("CT clutcher %+v, want %+v", got, want)
	}
	if got := out.Players[1].Clutches; got != (row.Clutches{Clutch1v2Attempts: 1}) {
		t.Errorf("T clutcher %+v, want an attempt", got)
	}
	if out.Teams[fact.SideCT].Clutches != 1 {
		t.Error("CT team clutch")
	}
}

// Q5: a clutch against no one is a clutch for the team, with no size to count; eco counts no attempt.
func TestNoEnemiesLeft(t *testing.T) {
	out := run(fact.SideT, round.Clutch{Size: 0, Steam: 1}, round.Clutch{})
	if out.Teams[fact.SideT].Clutches != 1 {
		t.Error("team clutch")
	}
	if got := out.Players[1].Clutches; got != (row.Clutches{}) {
		t.Errorf("%+v, want nothing", got)
	}
}

// A bot clutcher has no row in CSC's rule (Steam64 0) or eco's; a size over five is no clutch.
func TestBotAndOversize(t *testing.T) {
	out := run(fact.SideT, round.Clutch{Size: 2, Steam: 0}, round.Clutch{Size: 1, Steam: 11})
	if out.Teams[fact.SideT].Clutches != 0 || out.Players[0].Clutches != (row.Clutches{}) {
		t.Errorf("bot clutch counted: %+v %+v", out.Teams[fact.SideT], out.Players[0].Clutches)
	}
	out = run(fact.SideT, round.Clutch{Size: 6, Steam: 1}, round.Clutch{})
	if out.Teams[fact.SideT].Clutches != 0 || out.Players[1].Clutches != (row.Clutches{}) {
		t.Errorf("oversize clutch counted: %+v %+v", out.Teams[fact.SideT], out.Players[1].Clutches)
	}
}

// The slot's holder must have a row: a stranger's attempt is nobody's.
func TestStranger(t *testing.T) {
	out := run(fact.SideT, round.Clutch{Size: 2, Steam: 99}, round.Clutch{})
	if out.Players[99] != nil {
		t.Error("a row for a stranger")
	}
}
