// Package calc holds the stat calculators that replace the collectors (Core-planning#681). Each
// reads one committed round, as the Deriver handed it over, and writes only its own family of the
// round's row model. The payload runs them through Run and folds the rows (engine/fold); cmd/calcdiff
// compares them with the collectors until PR 18 deletes those.
package calc

import (
	"strconv"

	"github.com/csconfederation/fragg-3.0/engine/calc/clutches"
	"github.com/csconfederation/fragg-3.0/engine/calc/combat"
	"github.com/csconfederation/fragg-3.0/engine/calc/economy"
	"github.com/csconfederation/fragg-3.0/engine/calc/kast"
	"github.com/csconfederation/fragg-3.0/engine/calc/objectives"
	"github.com/csconfederation/fragg-3.0/engine/calc/openings"
	"github.com/csconfederation/fragg-3.0/engine/calc/positioning"
	"github.com/csconfederation/fragg-3.0/engine/calc/survival"
	"github.com/csconfederation/fragg-3.0/engine/calc/swing"
	"github.com/csconfederation/fragg-3.0/engine/calc/trades"
	"github.com/csconfederation/fragg-3.0/engine/calc/utility"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc is one family's calculator. It keeps nothing between calls, reads only constants, reads
// r.Events[:r.Commit] without the Dropped ones, and applies each Win at its r.Wins index. It
// can't fail: the Deriver made every check.
type Calc func(r *round.Round, out *row.Round)

// All run in this order: the tier 1 calculators, which read only r, then tier 2, which also read
// tier 1's results in out (design note section 3). The families land one per PR.
var All = []Calc{
	combat.Calc,
	kast.Calc,
	trades.Calc,
	openings.Calc,
	clutches.Calc,
	utility.Calc,
	objectives.Calc,
	survival.Calc,
	economy.Calc,
	positioning.Calc,
	swing.Calc,
	objectives.Finish, // tier 2
}

// Run is r's rows: the round's meta, one player row per member, then every calculator.
func Run(r *round.Round) *row.Round {
	out := &row.Round{Meta: meta(r), Players: map[uint64]*row.Player{},
		Teams: map[fact.Side]*row.Team{fact.SideT: {Clan: r.TClan}, fact.SideCT: {Clan: r.CTClan}}}
	for id, m := range r.OpenMembers {
		out.Players[id] = &row.Player{ID: identity(m)}
	}
	for _, e := range r.Events[:r.Commit] {
		if e.Join != nil {
			out.Players[e.Join.Steam64] = &row.Player{ID: identity(*e.Join)}
		}
	}
	for _, c := range All {
		c(r, out)
	}
	// Team meta: each team's R, and the R win of the team named the winner (by clan, as a lobby's
	// winner can disagree with the winning side).
	t, ct := out.Teams[fact.SideT], out.Teams[fact.SideCT]
	t.TR, ct.CtR = 1, 1
	if w, ok := latestWin(r); ok {
		t.TRW, ct.CtRW = b2i(w.Clan == r.TClan), b2i(w.Clan == r.CTClan)
	}
	return out
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// identity is who a member is, as the round row carries it.
func identity(m match.Member) row.ID {
	return row.ID{Name: m.Name, SteamID: strconv.FormatUint(m.Steam64, 10), IsBot: m.IsBot, TeamENUM: m.TeamID,
		TeamClanName: m.Clan, Side: int(m.Side)}
}

// latestWin is the latest Win whose RoundEnd comes before the commit: a tail's Win isn't the
// round's (Q67).
func latestWin(r *round.Round) (w round.Win, ok bool) {
	for _, c := range r.WinList() {
		if c.Event < r.Commit {
			w, ok = c, true
		}
	}
	return w, ok
}

// meta is the round's own fields: numbers, ticks, init counts and the latest Win's winner.
func meta(r *round.Round) row.Meta {
	m := row.Meta{RoundNum: r.Num, StartingTick: r.StartTick, EndingTick: r.EndTick,
		InitTerroristCount: r.InitT, InitCTCount: r.InitCT}
	if w, ok := latestWin(r); ok {
		m.WinnerClanName, m.WinnerENUM = w.Clan, int(w.Side)
		m.RoundEndReason = fact.RoundEndName(r.Events[w.Event].Fact.(fact.RoundEnd).Reason)
	}
	return m
}
