package economy

import (
	"testing"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
	"github.com/csconfederation/fragg-3.0/rating"
)

func newRound(num int, freeze bool, tEquip, ctEquip int) *round.Round {
	open := fact.Snapshot{Players: []fact.Player{{EquipValue: tEquip}, {EquipValue: tEquip + 200}, {EquipValue: ctEquip}}}
	return &round.Round{Num: num, FreezeEnd: freeze, Pistol: num == 1, TClan: "a", CTClan: "b", Open: open,
		Roster: match.Roster{T: []int{0, 1}, CT: []int{2}}, OpenMembers: map[uint64]match.Member{1: {}, 2: {}, 0: {}}}
}

func run(r *round.Round) *row.Round {
	out := &row.Round{Players: map[uint64]*row.Player{1: {}, 2: {}, 0: {}, 5: {}},
		Teams: map[fact.Side]*row.Team{fact.SideT: {}, fact.SideCT: {}}}
	Calc(r, out)
	return out
}

func win(r *round.Round, side fact.Side, clan string) {
	r.Events = append(r.Events, round.Event{Event: match.Event{Fact: fact.RoundEnd{Winner: side}, Win: true, Winner: clan}})
	r.Wins, r.Commit = []int{len(r.Events)}, len(r.Events)
}

func TestBuys(t *testing.T) {
	for _, c := range []struct {
		name         string
		r            *round.Round
		wantT, wantC row.Buy
		t, ct        row.TeamEconomy
	}{
		// Rounds 1 and 13 are pistol rounds whatever the money; the CT wins every round here.
		{"pistol", newRound(1, false, 800, 800), row.BuyPistol, row.BuyPistol,
			row.TeamEconomy{Pistols: 1, Buy: row.BuyPistol}, row.TeamEconomy{Pistols: 1, PistolsW: 1, Buy: row.BuyPistol}},
		// T average 1,000 (an eco) against a CT full buy: the CT's is an anti-eco, and won.
		{"anti-eco", newRound(2, true, 900, 4500), row.BuyEco, row.BuyFull,
			row.TeamEconomy{EcoBuys: 1, Buy: row.BuyEco}, row.TeamEconomy{FullBuys: 1, FullBuysW: 1, AntiEcos: 1, AntiEcosW: 1, Buy: row.BuyFull}},
		{"force", newRound(3, true, 2400, 3600), row.BuyForce, row.BuyFull,
			row.TeamEconomy{ForceBuys: 1, Buy: row.BuyForce}, row.TeamEconomy{FullBuys: 1, FullBuysW: 1, AntiEcos: 0, AntiEcosW: 0, Buy: row.BuyFull}},
		{"before the buy", newRound(4, false, 4000, 4000), row.BuyUnknown, row.BuyUnknown,
			row.TeamEconomy{}, row.TeamEconomy{}},
	} {
		win(c.r, fact.SideCT, "b")
		out := run(c.r)
		if got := out.Teams[fact.SideT].TeamEconomy; got != c.t {
			t.Errorf("%s: T %+v, want %+v", c.name, got, c.t)
		}
		if got := out.Teams[fact.SideCT].TeamEconomy; got != c.ct {
			t.Errorf("%s: CT %+v, want %+v", c.name, got, c.ct)
		}
	}
}

func TestKills(t *testing.T) {
	killer := &fact.Player{Steam64: 1, EquipValue: 5000}
	victim := &fact.Player{Steam64: 2, EquipValue: 1000}
	bot := &fact.Player{Steam64: 0, IsBot: true, EquipValue: 1000}
	kill := func(k, v *fact.Player, eco bool) round.Event {
		return round.Event{Event: match.Event{Fact: fact.Kill{Killer: k, Victim: v}}, Context: round.Context{EcoKill: eco}}
	}
	dropped := kill(killer, victim, true)
	dropped.Dropped = true
	r := newRound(2, true, 1000, 1000)
	r.Events = []round.Event{
		kill(killer, victim, true),  // a rifle kills a pistol
		kill(victim, killer, true),  // and the other way round
		kill(killer, victim, false), // a team kill or a bomb kill: no eco value
		kill(killer, bot, true),     // the bot has no row
		kill(bot, victim, true),     // the bot kills: only the death counts
		dropped,
	}
	// Player 5 kills before joining the round, and again after.
	late := &fact.Player{Steam64: 5, EquipValue: 5000}
	r.Events = append(r.Events, kill(late, victim, true),
		round.Event{Event: match.Event{Fact: fact.TeamChange{}, Join: &match.Member{Steam64: 5}}}, kill(late, victim, true))
	r.Commit = len(r.Events)
	out := run(r)
	kv, dp := rating.EcoKillValue(5000, 1000), rating.EcoDeathPenalty(1000, 5000)
	kv2, dp2 := rating.EcoKillValue(1000, 5000), rating.EcoDeathPenalty(5000, 1000)
	if kv >= 1 || kv2 <= 1 {
		t.Fatalf("test values: %v, %v", kv, kv2)
	}
	want1 := row.Economy{EcoKillValue: kv + kv, EcoDeathValue: dp2, KillsVsLowerBuy: 2, KillsVsMuchLowerBuy: 2}
	want2 := row.Economy{EcoKillValue: kv2, EcoDeathValue: dp + rating.EcoDeathPenalty(1000, 1000) + dp + dp}
	if kv > 0.85 {
		want1.KillsVsMuchLowerBuy = 0
	}
	if got := out.Players[1].Economy; got != want1 {
		t.Errorf("player 1: %+v, want %+v", got, want1)
	}
	if got := out.Players[2].Economy; got != want2 {
		t.Errorf("player 2: %+v, want %+v", got, want2)
	}
	kv5 := rating.EcoKillValue(5000, 1000)
	if got := out.Players[5].Economy; got.EcoKillValue != kv5 {
		t.Errorf("player 5, who joined after a kill: %+v, want one kill worth %v", got, kv5)
	}
	if got := out.Players[0].Economy; got != (row.Economy{}) {
		t.Errorf("bot row: %+v", got)
	}
}
