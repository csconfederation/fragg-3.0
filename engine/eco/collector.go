package eco

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
	"github.com/csconfederation/fragg-3.0/rating/swing"
)

// The demoinfocs equipment types the handlers test.
const (
	eqP2000      = 1
	eqRevolver   = 10
	eqAWP        = 309
	eqBomb       = 404
	eqKnife      = 405
	eqGrenades   = 500 // the first grenade type
	eqMolotov    = 502
	eqIncendiary = 503
	eqFlash      = 504
	eqSmoke      = 505
	eqHE         = 506
)

// Collector is the oracle's eco parser in attached mode (parser.AttachToParser): its stat
// handlers, and BeginRound, FinalizeRound and SnapshotRound, which the CSC pipeline drives. The
// oracle's probability DataCollector is left out: it only feeds offline probability tables.
type Collector struct {
	players       map[uint64]*model.PlayerStats
	round         map[uint64]*model.RoundStats
	trades        *TradeDetector
	swing         *SwingTracker
	isPistolRound bool
	roundNumber   int // eco's own count of opens, as in the oracle; labels only, not in the payload
	roundActive   bool
	mapName       string
	startTick     int // the round clock starts at the open's tick, as CSC's
	winTime       float64
	bombPlanted   bool
}

// Round is one committed round's eco stats (the oracle's RoundSnapshot).
type Round struct {
	stats map[uint64]*model.RoundStats
}

// New returns a collector with the default swing configuration.
func New() *Collector {
	return &Collector{
		players: map[uint64]*model.PlayerStats{},
		round:   map[uint64]*model.RoundStats{},
		trades:  NewTradeDetector(),
		swing:   NewSwingTrackerWithConfig(swing.DefaultConfig()),
	}
}

// playing is the oracle's Participants().Playing(): the snapshot's listed records.
func playing(s fact.Snapshot) []fact.Player {
	ps := make([]fact.Player, 0, len(s.Players))
	for _, p := range s.Players {
		if p.Listed {
			ps = append(ps, p)
		}
	}
	return ps
}

func (c *Collector) timeInRound(tick int) float64 {
	return float64(tick-c.startTick) / float64(rating.TickRate)
}

// ensurePlayer registers p. The name and clan are metadata the payload doesn't carry.
func (c *Collector) ensurePlayer(p fact.Player, s fact.Snapshot) {
	clan := s.T.Clan
	if p.Side == fact.SideCT {
		clan = s.CT.Clan
	}
	if ps, ok := c.players[p.Steam64]; !ok {
		c.players[p.Steam64] = &model.PlayerStats{SteamID: fmt.Sprintf("%d", p.Steam64), Name: p.Name, TeamName: clan}
	} else if ps.TeamName == "" {
		ps.TeamName = clan
	}
}

// row is steam's row in the open round. Rows are the round's human members (#591): anyone else
// (bots, players who connected after the open) gets a scratch row that is dropped.
func (c *Collector) row(steam uint64) *model.RoundStats {
	if r, ok := c.round[steam]; ok {
		return r
	}
	return &model.RoundStats{}
}

// Open is the oracle's BeginRound, called on every round open; num is the match's round number
// and members the round's members (bots included), whose alive counts are tAlive and ctAlive.
// D13: the pistol rounds are rounds 1 and 13, where the oracle used IsPistolRound on eco's own
// count of opens.
func (c *Collector) Open(s fact.Snapshot, num int, members []fact.Player, tAlive, ctAlive int) {
	c.round = map[uint64]*model.RoundStats{}
	c.trades.Reset()
	c.winTime = 0
	c.bombPlanted = false
	c.roundActive = true
	c.roundNumber++
	c.isPistolRound = num == rating.FirstHalfPistolRound || num == rating.SecondHalfPistolRound
	c.startTick = s.Tick

	tCount, ctCount, tEquip, ctEquip := 0, 0, 0, 0
	for _, p := range members {
		switch p.Side {
		case fact.SideT:
			tCount++
			tEquip += p.EquipValue
		case fact.SideCT:
			ctCount++
			ctEquip += p.EquipValue
		}
		if !p.IsBot {
			c.ensurePlayer(p, s)
			c.addRow(p.Steam64, p.Side)
		}
	}

	c.swing.ResetRoundWithClock(c.roundNumber, min(tAlive, 5), min(ctAlive, 5), c.mapName, s.Tick,
		DefaultRoundTimeSeconds)
	tAvg, ctAvg := 0.0, 0.0
	if tCount > 0 {
		tAvg = float64(tEquip) / float64(tCount)
	}
	if ctCount > 0 {
		ctAvg = float64(ctEquip) / float64(ctCount)
	}
	c.swing.SetEconomyFromValues(tAvg, ctAvg)
}

func sideName(side fact.Side) string {
	switch side {
	case fact.SideT:
		return "T"
	case fact.SideCT:
		return "CT"
	}
	return ""
}

// addRow gives a member a row on their side; late rows are pistol rows too (D13).
func (c *Collector) addRow(steam uint64, side fact.Side) {
	c.round[steam] = &model.RoundStats{IsPistolRound: c.isPistolRound, PlayerSide: sideName(side)}
}

// Join adds a human who joined the open round mid-round, on the side CSC gave them, so eco covers
// every round record CSC has. The oracle left such a record without a side and flagged the gap
// with its coverage check (Core-planning#303).
func (c *Collector) Join(steam uint64, name string, isBot bool, side fact.Side, clan string) {
	if isBot || !c.roundActive {
		return
	}
	if ps, ok := c.players[steam]; !ok {
		c.players[steam] = &model.PlayerStats{SteamID: fmt.Sprintf("%d", steam), Name: name, TeamName: clan}
	} else if ps.TeamName == "" {
		ps.TeamName = clan
	}
	if _, ok := c.round[steam]; !ok {
		c.addRow(steam, side)
	}
}

// Win is the oracle's FinalizeRound: round-end trades, survival, clutches and residual swing.
func (c *Collector) Win(winner fact.Side, s fact.Snapshot, alive []fact.Player, st round.State) {
	if !c.roundActive || s.Warmup {
		return
	}
	duration := c.timeInRound(s.Tick)
	c.winTime = duration
	c.trades.ProcessRoundEndTrades(s.Tick, c.round)

	for _, r := range c.round { // every row, a disconnected member's too (#591)
		r.TeamWon = r.PlayerSide == "T" && winner == fact.SideT || r.PlayerSide == "CT" && winner == fact.SideCT
	}
	ps := playing(s)
	isAlive, aliveOn := map[uint64]bool{}, map[fact.Side]int{}
	for _, p := range alive {
		isAlive[p.Steam64] = true
		aliveOn[p.Side]++
	}
	for _, p := range ps {
		r := c.row(p.Steam64)
		if isAlive[p.Steam64] && aliveOn[p.Side] == 1 {
			r.WasLastAlive = true
		}
	}
	// Clutches are the round's, as CSC's (#591, D23): each side's last member alive, however the
	// rest died, with the enemies alive then.
	for _, cl := range []round.Clutch{st.TClutch, st.CTClutch} {
		if r, ok := c.round[cl.Steam]; ok && cl.Size > 0 {
			r.ClutchAttempt, r.ClutchSize, r.ClutchKills, r.ClutchWon = true, cl.Size, r.Kills, r.TeamWon
		}
	}

	maxDamage := 0
	for _, r := range c.round {
		if r.Damage > maxDamage {
			maxDamage = r.Damage
		}
	}
	c.applyLedger(c.swing.ApplyRoundResidual(winner, c.roundContext(alive), maxDamage), duration, "")
	c.swing.FinalizeRound()
}

// Commit is the oracle's SnapshotRound: the round's stats, copied when the round commits. Survival
// is final here, as CSC's (#591): no death in the round, post-round deaths included.
func (c *Collector) Commit() *Round {
	stats := make(map[uint64]*model.RoundStats, len(c.round))
	for id, rs := range c.round {
		rs.Survived = rs.Deaths == 0
		rs.TimeAlive = rs.DeathTime
		if rs.Survived {
			rs.TimeAlive = c.winTime
		}
		rs.SavedWeapons = rs.Survived && !rs.TeamWon
		cp := *rs
		cp.KillTimes = append([]float64(nil), rs.KillTimes...)
		cp.SwingContributions = append([]model.SwingContribution(nil), rs.SwingContributions...)
		stats[id] = &cp
	}
	return &Round{stats: stats}
}

// Players folds the committed rounds into match totals and derives the ratings (the oracle's
// FoldSnapshots and FinalizeStats). engine/fold calls it once.
func (c *Collector) Players(rounds []*Round) map[uint64]*model.PlayerStats {
	for _, r := range rounds {
		for id, rs := range r.stats {
			p := c.players[id]
			if p == nil {
				p = &model.PlayerStats{SteamID: fmt.Sprintf("%d", id)}
				c.players[id] = p
			}
			FoldRound(p, rs)
			u := NewSideStatsUpdater(p, rs)
			u.UpdateCommonRoundStats()
			u.UpdateSideStats()
			p.RoundsPlayed++
		}
	}
	for _, p := range c.players {
		Derive(p, false)
	}
	return c.players
}

func (c *Collector) applyLedger(allocs []swing.PlayerSwingAllocation, timeInRound float64, opponent string) {
	for _, a := range allocs {
		if a.Amount == 0 {
			continue
		}
		rs, ok := c.round[a.SteamID]
		if !ok {
			continue
		}
		rs.ProbabilitySwing += a.Amount
		rs.AddSwingContribution(model.SwingContribution{Type: string(a.Reason), Amount: a.Amount,
			TimeInRound: timeInRound, Opponent: opponent})
	}
}

// roundContext is the oracle's buildPlayerRoundContext over the round's alive members.
func (c *Collector) roundContext(members []fact.Player) map[uint64]swing.PlayerRoundContext {
	alive := map[uint64]bool{}
	for _, p := range members {
		alive[p.Steam64] = true
	}
	players := map[uint64]swing.PlayerRoundContext{}
	for id, rs := range c.round {
		side := fact.SideUnassigned
		if rs.PlayerSide == "T" {
			side = fact.SideT
		} else if rs.PlayerSide == "CT" {
			side = fact.SideCT
		}
		positive, negative := 0.0, 0.0
		if rs.ProbabilitySwing > 0 {
			positive = rs.ProbabilitySwing
		} else if rs.ProbabilitySwing < 0 {
			negative = -rs.ProbabilitySwing
		}
		players[id] = swing.PlayerRoundContext{SteamID: id, Side: side, Alive: alive[id], Damage: rs.Damage,
			Kills: rs.Kills, FlashAssists: rs.FlashAssists, PositiveSwing: positive, NegativeSwing: negative,
			PlantedBomb: rs.PlantedBomb, DefusedBomb: rs.DefusedBomb}
	}
	return players
}

// Fact runs the oracle's eco handlers for f with its round context. Kills run handleKill, then
// handleRoundDecisionKill; kills count as the round decides (#591): expected, not bomb-ignored.
func (c *Collector) Fact(f fact.Fact, x round.Context) error {
	err := c.fact(f, x)
	c.swing.SetAlive(x.After.TAlive, x.After.CTAlive) // swing's alive counts are the round's
	return err
}

func (c *Collector) fact(f fact.Fact, x round.Context) error {
	switch f := f.(type) {
	case fact.ServerInfo:
		c.mapName = f.Map
	case fact.Kill:
		if x.Victim {
			c.death(f)
		}
		swung := x.Expected && !x.BombIgnored && c.kill(f, x)
		if x.Victim && !swung {
			c.swing.RecordDeath(f.Victim.Steam64, f.Victim.Side)
		}
		if x.Assist {
			ar := c.row(f.Assister.Steam64)
			ar.GotAssist = true
			ar.Assists++
		}
		if x.FlashAssisted { // CSC's flash assists (#591); eco counted every enemy flash
			c.row(x.FlashAssister).FlashAssists++
		}
	case fact.Hurt:
		if !f.Warmup && c.roundActive && x.Enemy && !x.BombIgnored {
			c.hurt(f)
		}
	case fact.Flashed:
		if c.roundActive && f.Attacker != nil && f.Player != nil {
			r := c.row(f.Attacker.Steam64)
			if f.Attacker.Side != f.Player.Side {
				if x.Flash { // CSC's filter (#591); swing keeps its own flash credit below
					r.EnemyFlashDuration += f.Duration
					r.EnemiesFlashed++
				}
				c.swing.RecordFlash(f.Attacker.Steam64, f.Player.Steam64, f.Duration, f.Tick)
			} else if f.Attacker.Steam64 != f.Player.Steam64 {
				r.TeamFlashCount++
				r.TeamFlashDuration += f.Duration
			}
		}
	case fact.GrenadeThrow:
		if c.roundActive && f.Thrower != nil && f.Weapon != nil {
			r := c.row(f.Thrower.Steam64)
			switch *f.Weapon {
			case eqFlash:
				r.FlashesThrown++
			case eqSmoke:
				r.SmokesThrown++
			case eqHE:
				r.HEsThrown++
			case eqMolotov, eqIncendiary:
				r.MolotovsThrown++
			}
		}
	case fact.BombPlanted:
		if !c.roundActive {
			return nil
		}
		if f.Player == nil {
			return fmt.Errorf("bomb planted by nobody at tick %d", f.Tick) // the oracle dereferences nil
		}
		c.bombPlanted = true
		c.row(f.Player.Steam64).PlantedBomb = true
		t := c.timeInRound(f.Tick)
		c.applyLedger(c.swing.RecordBombPlant(f.Player.Steam64, t, c.roundContext(x.Alive), f.Tick), t, "")
	case fact.BombDefused:
		if !c.roundActive {
			return nil
		}
		if f.Player == nil {
			return fmt.Errorf("bomb defused by nobody at tick %d", f.Tick) // the oracle dereferences nil
		}
		c.row(f.Player.Steam64).DefusedBomb = true
		t := c.timeInRound(f.Tick)
		c.applyLedger(c.swing.RecordBombDefuse(f.Player.Steam64, t, c.roundContext(x.Alive), f.Tick), t, "")
	}
	return nil
}

// hurt is enemy damage the round counts.
func (c *Collector) hurt(f fact.Hurt) {
	dmg := f.HealthDamageTaken
	r := c.row(f.Attacker.Steam64)
	r.Damage += dmg
	c.row(f.Player.Steam64).DamageTaken += dmg
	if f.Weapon != nil {
		if *f.Weapon >= eqGrenades && *f.Weapon <= eqHE { // any grenade, impact hits too, as CSC's (#591)
			r.UtilityDamage += dmg
		}
		switch *f.Weapon {
		case eqHE:
			r.HEDamage += dmg
		case eqMolotov, eqIncendiary:
			r.FireDamage += dmg
		}
	}
	c.swing.RecordDamage(f.Attacker.Steam64, f.Player.Steam64, dmg, c.timeInRound(f.Tick))
}

// kill is the oracle's handleKill. It reports whether the kill reached swing's kill transition.
func (c *Collector) kill(f fact.Kill, x round.Context) bool {
	attacker, victim := f.Killer, f.Victim
	// A suicide is on its own side; bots share Steam64 0, so equal IDs alone don't make one.
	if attacker != nil && victim != nil && (attacker.Steam64 == victim.Steam64 && !attacker.IsBot || attacker.Side == victim.Side) {
		return false
	}
	tick := f.Tick
	t := c.timeInRound(tick)
	ps := playing(f.Snapshot)
	var attackerEquip, victimEquip int
	var killValue, deathPenalty, tradeSpeed float64
	isTradeKill := len(x.Avenged) > 0 // engine/round's trades (D22); speed from the oldest avenged death
	if isTradeKill {
		tradeSpeed = t - c.timeInRound(x.Avenged[0].Tick)
	}
	if attacker != nil && victim != nil {
		attackerEquip, victimEquip = attacker.EquipValue, victim.EquipValue
		killValue = rating.EcoKillValue(float64(attackerEquip), float64(victimEquip))
		deathPenalty = rating.EcoDeathPenalty(float64(victimEquip), float64(attackerEquip))
	}

	// processVictimDeath
	if victim != nil {
		c.trades.RecordDeath(victim, attacker, tick, ps)
	}

	// processTradeDetection: every death the kill avenges; swing reallocates the oldest's penalty.
	if isTradeKill {
		for _, a := range x.Avenged {
			d := c.row(a.Steam)
			d.Traded, d.TradeDeath, d.SavedByTeammate = true, true, true
		}
		ar := c.row(attacker.Steam64)
		ar.SavedTeammate = true
		ar.TradeDenials++
		oldest := x.Avenged[0].Steam
		if d, ok := c.round[oldest]; ok && d.LastDeathSwing < 0 {
			mates := make([]uint64, 0) // the round's human members on the trader's side (#591)
			for _, id := range slices.Sorted(maps.Keys(c.round)) {
				if c.round[id].PlayerSide == sideName(attacker.Side) && id != oldest {
					mates = append(mates, id)
				}
			}
			c.applyLedger(c.swing.ApplyTradeReallocation(oldest, attacker.Side, d.LastDeathSwing, mates), t,
				victim.Name)
		}
	}
	if attacker != nil && victim != nil {
		c.trades.Clear(victim.Steam64) // the victim's kills can't be traded any more
	}
	c.trades.ProcessExpiredTrades(tick, c.round)

	if attacker == nil || victim == nil {
		return false
	}

	// processKillerStats
	r := c.row(attacker.Steam64)
	vr := c.row(victim.Steam64)
	r.KillTimes = append(r.KillTimes, t)
	// An exit frag is a kill once the round is decided, as CSC reads it (#591): won, defused or exploded.
	decided := x.Before.Phase == round.Decided
	if decided {
		r.IsExitFrag = true
		r.ExitFrags++
	}
	r.Kills++
	r.GotKill = true
	r.EconImpact += killValue
	if f.Headshot {
		r.Headshots++
	}
	if ttk := c.swing.GetTimeToKill(attacker.Steam64, victim.Steam64, t); ttk >= 0 {
		r.TimeToKillSum += ttk
		r.KillsWithTTK++
	}
	if killValue < 1.0 {
		r.LowBuyKills++
	}
	if killValue <= 0.85 {
		r.DisadvantagedBuyKills++
	}
	attackerBefore, victimBefore := alive(x.Before, attacker.Side), alive(x.Before, victim.Side)
	attackerAfter, victimAfter := alive(x.After, attacker.Side), alive(x.After, victim.Side)
	if attackerBefore <= victimBefore && attackerAfter > victimAfter {
		r.ManAdvantageKills++
	}
	if victimBefore >= attackerBefore && victimAfter < attackerAfter {
		vr.ManDisadvantageDeaths++
	}
	vr.EcoDeathValue += deathPenalty

	// processWeaponStats
	if f.Weapon != nil {
		switch *f.Weapon {
		case eqAWP:
			r.AWPKills++
			r.AWPKill = true
		case eqKnife:
			r.KnifeKill = true
		case eqHE, eqMolotov, eqIncendiary:
			r.UtilityKills++
		}
		if *f.Weapon >= eqP2000 && *f.Weapon <= eqRevolver && victimEquip > 3500 {
			r.PistolVsRifleKill = true
		}
	}

	// processOpeningKill: the round's opening, as CSC's (#591)
	if x.Before.Opening && x.Enemy {
		r.OpeningKill, r.EntryFragger, r.InvolvedInOpening = true, true, true
		if f.Weapon != nil && *f.Weapon == eqAWP {
			r.AWPOpeningKill = true
		}
		vr.OpeningDeath, vr.InvolvedInOpening = true, true
	}

	// processSwingTracking
	if isTradeKill {
		r.TradeKills++
		if tradeSpeed > 0 && tradeSpeed < 2.0 {
			r.FastTrades++
		}
	}
	// Swing's kill transition is the counted victim's death (#591): not a non-member's.
	if x.Victim {
		losing := make([]uint64, 0)
		for _, p := range x.Alive {
			if !p.IsBot && p.Side == victim.Side {
				losing = append(losing, p.Steam64)
			}
		}
		kr := c.swing.RecordKill(attacker.Steam64, victim.Steam64, attacker.Side, victim.Side, float64(attackerEquip),
			float64(victimEquip), t, isTradeKill, f.Headshot, decided, losing, tick)
		c.applyLedger(kr.Allocations, t, victim.Name)
		for _, a := range kr.Allocations {
			if a.SteamID == victim.Steam64 && a.Reason == swing.SwingReasonVictimDeath {
				vr.LastDeathSwing = a.Amount
			}
		}
		if kr.SurvivalCreditPerPlayer > 0 {
			allocs := make([]swing.PlayerSwingAllocation, 0, len(kr.SurvivalBeneficiaries))
			for _, id := range kr.SurvivalBeneficiaries {
				allocs = append(allocs, swing.PlayerSwingAllocation{SteamID: id, Side: attacker.Side,
					Amount: kr.SurvivalCreditPerPlayer, Reason: swing.SwingReasonSurvivalCredit})
			}
			c.applyLedger(allocs, t, victim.Name)
		}
		if kr.Swing.EcoMultiplier > 0 {
			r.EcoAdjustedKills += kr.Swing.EcoMultiplier
		}
	}

	// processEcoKillFlags
	ratio := float64(victimEquip) / math.Max(float64(attackerEquip), 500.0)
	if ratio > 2.0 {
		r.EcoKill = true
	}
	if ratio < 0.5 {
		vr.AntiEcoKill = true
	}
	if f.Headshot {
		r.PerfectKills++
	}

	return x.Victim
}

// death records a death the round counts (#591): every death, team kills and suicides too.
func (c *Collector) death(f fact.Kill) {
	v := c.row(f.Victim.Steam64)
	v.Deaths++
	v.DeathTime = c.timeInRound(f.Tick)
	for _, w := range f.Victim.Weapons {
		if w == eqAWP {
			v.HadAWP, v.LostAWP = true, true
			break
		}
	}
}

func alive(s round.State, side fact.Side) int {
	if side == fact.SideT {
		return s.TAlive
	}
	return s.CTAlive
}
