package eco

import (
	"fmt"
	"math"

	"github.com/csconfederation/fragg-3.0/engine/fact"
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
	players        map[uint64]*model.PlayerStats
	round          map[uint64]*model.RoundStats
	trades         *TradeDetector
	swing          *SwingTracker
	roundHasKill   bool
	isPistolRound  bool
	roundNumber    int // eco's own count of opens, as in the oracle; labels only, not in the payload
	roundActive    bool
	mapName        string
	roundStartTime float64
	roundDecided   bool
	roundDecidedAt float64
	bombPlanted    bool
	dead           map[uint64]bool // this round's victims
}

// Round is one committed round's eco stats (the oracle's RoundSnapshot).
type Round struct {
	number int
	stats  map[uint64]*model.RoundStats
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

func (c *Collector) timeInRound(frame int) float64 {
	return float64(frame)/float64(rating.TickRate) - c.roundStartTime
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

func (c *Collector) ensureRound(steam uint64) *model.RoundStats {
	if _, ok := c.round[steam]; !ok {
		c.round[steam] = &model.RoundStats{IsPistolRound: c.isPistolRound} // late records too (D13)
	}
	return c.round[steam]
}

// countAlive is the oracle's CountAlivePlayers: alive humans per side, capped at 5.
func countAlive(ps []fact.Player) (t, ct int) {
	for _, p := range ps {
		if p.IsBot || !p.Alive {
			continue
		}
		if p.Side == fact.SideT {
			t++
		} else if p.Side == fact.SideCT {
			ct++
		}
	}
	return min(t, 5), min(ct, 5)
}

// Open is the oracle's BeginRound, called on every round open; num is the match's round number.
// D13: the pistol rounds are rounds 1 and 13, where the oracle used IsPistolRound on eco's own
// count of opens.
func (c *Collector) Open(s fact.Snapshot, num int) {
	c.round = map[uint64]*model.RoundStats{}
	c.dead = map[uint64]bool{}
	c.roundHasKill = false
	c.trades.Reset()
	c.roundDecided, c.roundDecidedAt = false, 0
	c.bombPlanted = false
	c.roundActive = true
	c.roundNumber++
	c.isPistolRound = num == rating.FirstHalfPistolRound || num == rating.SecondHalfPistolRound
	c.roundStartTime = float64(s.Frame) / float64(rating.TickRate)

	tAlive, ctAlive, tEquip, ctEquip := 0, 0, 0, 0
	for _, p := range playing(s) {
		if p.IsBot {
			continue
		}
		c.ensurePlayer(p, s)
		rs := c.ensureRound(p.Steam64)
		if p.Side == fact.SideT {
			rs.PlayerSide = "T"
			tAlive++
			tEquip += p.EquipValue
		} else if p.Side == fact.SideCT {
			rs.PlayerSide = "CT"
			ctAlive++
			ctEquip += p.EquipValue
		}
	}
	tAlive, ctAlive = min(tAlive, 5), min(ctAlive, 5)

	c.swing.ResetRoundWithClock(c.roundNumber, tAlive, ctAlive, c.mapName, s.Tick, DefaultRoundTimeSeconds)
	tAvg, ctAvg := 0.0, 0.0
	if tAlive > 0 {
		tAvg = float64(tEquip) / float64(tAlive)
	}
	if ctAlive > 0 {
		ctAvg = float64(ctEquip) / float64(ctAlive)
	}
	c.swing.SetEconomyFromValues(tAvg, ctAvg)
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
	rs := c.ensureRound(steam)
	switch side {
	case fact.SideT:
		rs.PlayerSide = "T"
	case fact.SideCT:
		rs.PlayerSide = "CT"
	}
}

// Win is the oracle's FinalizeRound: round-end trades, survival, clutches and residual swing.
func (c *Collector) Win(winner fact.Side, s fact.Snapshot) {
	if !c.roundActive || s.Warmup {
		return
	}
	duration := c.timeInRound(s.Frame)
	c.trades.ProcessRoundEndTrades(s.Frame, c.round)

	ps := playing(s)
	for _, p := range ps {
		r := c.ensureRound(p.Steam64)
		r.TeamWon = p.Side == winner
		if p.Alive {
			r.Survived = true
			r.TimeAlive = duration
		} else if r.DeathTime > 0 {
			r.TimeAlive = r.DeathTime
		}
	}
	for _, p := range ps {
		r := c.ensureRound(p.Steam64)
		mates := 0
		for _, o := range ps {
			if o.Alive && o.Side == p.Side {
				mates++
			}
		}
		if p.Alive && mates == 1 {
			r.WasLastAlive = true
		}
		if r.ClutchEnteredSize > 0 {
			r.ClutchAttempt = true
			r.ClutchSize = r.ClutchEnteredSize
			r.ClutchKills = r.Kills
			if r.TeamWon {
				r.ClutchWon = true
			}
		}
		if p.Alive && !r.TeamWon {
			r.SavedWeapons = true
		}
	}

	maxDamage := 0
	for _, r := range c.round {
		if r.Damage > maxDamage {
			maxDamage = r.Damage
		}
	}
	c.applyLedger(c.swing.ApplyRoundResidual(winner, c.roundContext(s), maxDamage), duration, "")
	c.swing.FinalizeRound()
}

// Commit is the oracle's SnapshotRound: the round's stats, copied when the round commits.
func (c *Collector) Commit() *Round {
	stats := make(map[uint64]*model.RoundStats, len(c.round))
	for id, rs := range c.round {
		cp := *rs
		cp.KillTimes = append([]float64(nil), rs.KillTimes...)
		cp.SwingContributions = append([]model.SwingContribution(nil), rs.SwingContributions...)
		stats[id] = &cp
	}
	return &Round{number: c.roundNumber, stats: stats}
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
			FoldRound(p, rs, r.number)
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

// roundContext is the oracle's buildPlayerRoundContext.
func (c *Collector) roundContext(s fact.Snapshot) map[uint64]swing.PlayerRoundContext {
	alive := map[uint64]bool{}
	for _, p := range playing(s) {
		if !p.IsBot {
			alive[p.Steam64] = p.Alive
		}
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

// Fact runs the oracle's eco handlers for f. Kills run handleKill, then handleRoundDecisionKill.
func (c *Collector) Fact(f fact.Fact) error {
	switch f := f.(type) {
	case fact.ServerInfo:
		c.mapName = f.Map
	case fact.Kill:
		if !f.Warmup && c.roundActive {
			c.kill(f)
		}
		if c.roundActive && f.Victim != nil {
			c.dead[f.Victim.Steam64] = true // suicides and team kills too, which kill skips
		}
		if c.roundActive && !c.roundDecided {
			if t, ct := countAlive(playing(f.Snapshot)); t == 0 || ct == 0 {
				c.roundDecided, c.roundDecidedAt = true, c.timeInRound(f.Frame)
			}
		}
	case fact.Hurt:
		if !f.Warmup && c.roundActive {
			c.hurt(f)
		}
	case fact.Flashed:
		if c.roundActive && f.Attacker != nil && f.Player != nil {
			r := c.ensureRound(f.Attacker.Steam64)
			if f.Attacker.Side != f.Player.Side {
				r.FlashAssists++
				r.EnemyFlashDuration += f.Duration
				r.EnemiesFlashed++
				c.swing.RecordFlash(f.Attacker.Steam64, f.Player.Steam64, f.Duration, f.Tick)
			} else if f.Attacker.Steam64 != f.Player.Steam64 {
				r.TeamFlashCount++
				r.TeamFlashDuration += f.Duration
			}
		}
	case fact.GrenadeThrow:
		if c.roundActive && f.Thrower != nil && f.Weapon != nil {
			r := c.ensureRound(f.Thrower.Steam64)
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
		c.ensurePlayer(*f.Player, f.Snapshot)
		c.ensureRound(f.Player.Steam64).PlantedBomb = true
		t := c.timeInRound(f.Frame)
		c.applyLedger(c.swing.RecordBombPlant(f.Player.Steam64, t, c.roundContext(f.Snapshot), f.Tick), t, "")
	case fact.BombDefused:
		if !c.roundActive {
			return nil
		}
		if f.Player == nil {
			return fmt.Errorf("bomb defused by nobody at tick %d", f.Tick) // the oracle dereferences nil
		}
		c.ensurePlayer(*f.Player, f.Snapshot)
		c.ensureRound(f.Player.Steam64).DefusedBomb = true
		t := c.timeInRound(f.Frame)
		c.applyLedger(c.swing.RecordBombDefuse(f.Player.Steam64, t, c.roundContext(f.Snapshot), f.Tick), t, "")
		c.roundDecided, c.roundDecidedAt = true, t
	case fact.BombExplode:
		if c.roundActive {
			c.roundDecided, c.roundDecidedAt = true, c.timeInRound(f.Frame)
		}
	}
	return nil
}

func (c *Collector) hurt(f fact.Hurt) {
	if f.Attacker == nil || f.Player == nil || f.Attacker.Side == f.Player.Side {
		return
	}
	dmg := f.HealthDamageTaken
	r := c.ensureRound(f.Attacker.Steam64)
	r.Damage += dmg
	c.ensureRound(f.Player.Steam64).DamageTaken += dmg
	if f.Weapon != nil {
		switch *f.Weapon {
		case eqHE:
			r.UtilityDamage += dmg
			r.HEDamage += dmg
		case eqMolotov, eqIncendiary:
			r.UtilityDamage += dmg
			r.FireDamage += dmg
		}
	}
	c.swing.RecordDamage(f.Attacker.Steam64, f.Player.Steam64, dmg, c.timeInRound(f.Frame))
}

// kill is the oracle's handleKill.
func (c *Collector) kill(f fact.Kill) {
	attacker, victim := f.Killer, f.Victim
	if attacker != nil && victim != nil && (attacker.Steam64 == victim.Steam64 || attacker.Side == victim.Side) {
		return
	}
	tick := f.Frame
	t := c.timeInRound(tick)
	ps := playing(f.Snapshot)
	var attackerEquip, victimEquip int
	var killValue, deathPenalty, tradeSpeed float64
	isTradeKill := false
	if attacker != nil && victim != nil {
		attackerEquip, victimEquip = attacker.EquipValue, victim.EquipValue
		killValue = rating.EcoKillValue(float64(attackerEquip), float64(victimEquip))
		deathPenalty = rating.EcoDeathPenalty(float64(victimEquip), float64(attackerEquip))
		isTradeKill, tradeSpeed = c.trades.CheckTradeKill(attacker, victim, tick, t)
	}

	// processVictimDeath
	if victim != nil {
		vr := c.ensureRound(victim.Steam64)
		vr.DeathTime = t
		if f.Weapon == nil || *f.Weapon != eqBomb {
			c.clutchEntry(victim, ps)
		}
		for _, w := range victim.Weapons {
			if w == eqAWP {
				vr.HadAWP, vr.LostAWP = true, true
				break
			}
		}
		c.trades.RecordDeath(victim, attacker, tick, t, ps)
	}

	// processTradeDetection
	if attacker != nil && victim != nil {
		tr := c.trades.CheckForTrade(attacker, victim, tick, t, c.players, c.round)
		if tr.IsTrade {
			ar := c.ensureRound(attacker.Steam64)
			ar.SavedTeammate = true
			ar.TradeDenials++
			if tr.TradeReallocation {
				mates := make([]uint64, 0)
				for _, p := range ps {
					if !p.IsBot && p.Side == attacker.Side && p.Steam64 != tr.TradedPlayerID {
						mates = append(mates, p.Steam64)
					}
				}
				c.applyLedger(c.swing.ApplyTradeReallocation(tr.TradedPlayerID, attacker.Side, tr.OriginalDeathPenalty,
					mates), t, victim.Name)
			}
		}
	}
	c.trades.ProcessExpiredTrades(tick, c.round)

	if attacker == nil || victim == nil {
		return
	}
	c.trades.RecordKill(attacker, victim, tick)

	// processKillerStats
	r := c.ensureRound(attacker.Steam64)
	vr := c.ensureRound(victim.Steam64)
	r.KillTimes = append(r.KillTimes, t)
	if c.roundDecided {
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
	tAlive, ctAlive := countAlive(ps)
	attackerAfter, victimAfter := ctAlive, tAlive
	if attacker.Side == fact.SideT {
		attackerAfter, victimAfter = tAlive, ctAlive
	}
	attackerBefore, victimBefore := attackerAfter, victimAfter+1
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

	// processOpeningKill
	if !c.roundHasKill {
		r.OpeningKill, r.EntryFragger, r.InvolvedInOpening = true, true, true
		if f.Weapon != nil && *f.Weapon == eqAWP {
			r.AWPOpeningKill = true
		}
		vr.OpeningDeath, vr.InvolvedInOpening = true, true
		c.roundHasKill = true
	}

	// processSwingTracking
	if isTradeKill {
		r.TradeKill = true
		r.TradeSpeed = tradeSpeed
	}
	losing := make([]uint64, 0)
	for _, p := range ps {
		if !p.IsBot && p.Alive && p.Side == victim.Side && p.Steam64 != victim.Steam64 {
			losing = append(losing, p.Steam64)
		}
	}
	kr := c.swing.RecordKill(attacker.Steam64, victim.Steam64, attacker.Side, victim.Side, float64(attackerEquip),
		float64(victimEquip), t, isTradeKill, f.Headshot, c.roundDecided, losing, tick)
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

	// processAssist
	if f.Assister != nil {
		ar := c.ensureRound(f.Assister.Steam64)
		ar.GotAssist = true
		ar.Assists++
	}
}

// clutchEntry is the oracle's checkClutchEntry: the victim's death leaves one teammate alive
// against at least one enemy. As in CSC (corpus finding 6), a bomb death can't start a clutch, and
// a player killed earlier in the tick isn't alive though their entity still says so.
func (c *Collector) clutchEntry(victim *fact.Player, ps []fact.Player) {
	mates, enemies := 0, 0
	var last uint64
	for _, p := range ps {
		if p.Steam64 == victim.Steam64 || !p.Alive || c.dead[p.Steam64] {
			continue
		}
		if p.Side == victim.Side {
			mates++
			last = p.Steam64
		} else {
			enemies++
		}
	}
	if mates == 1 && enemies > 0 {
		if r := c.ensureRound(last); r.ClutchEnteredSize == 0 {
			r.ClutchEnteredSize = enemies
		}
	}
}
