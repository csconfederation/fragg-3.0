package cscstats

import (
	"fmt"
	"math"
	"strconv"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// The oracle's constants (csc/parser.go).
const (
	mr          = 12
	tickRate    = 64
	tradeCutoff = 4 // seconds
	eqBomb      = 404
	eqAWP       = 309
)

var multiKillBonus = [...]float64{0, 0, 0.3, 0.7, 1.2, 2}
var clutchBonus = [...]float64{0, 0.2, 0.6, 1.2, 2, 3}

var roundEndReasons = map[int]string{
	0:  "StillInProgress", //base values
	1:  "TargetBombed",
	2:  "VIPEscaped",
	3:  "VIPKilled",
	4:  "TerroristsEscaped",
	5:  "CTStoppedEscape",
	6:  "TerroristsStopped",
	7:  "BombDefused",
	8:  "CTWin",
	9:  "TerroristsWin",
	10: "Draw",
	11: "HostagesRescued",
	12: "TargetSaved",
	13: "HostagesNotRescued",
	14: "TerroristsNotEscaped",
	15: "VIPNotEscaped",
	16: "GameStart",
	17: "TerroristsSurrender",
	18: "CTSurrender",
	19: "TerroristsPlanted",
	20: "CTsReachedHostage",
}

func isRoundFinalInHalf(round int8) bool {
	return round%mr == 0 || (round > mr*2 && round%3 == 0)
}

// roundFlags are the oracle's per-round flags that the stats read (its Game.Flags, minus the
// lifecycle ones match owns).
type roundFlags struct {
	PrePlant, PostPlant, PostWinCon bool
	TAlive, CtAlive                 int
	TMoney                          bool
	TClutchVal, CtClutchVal         int
	TClutchSteam, CtClutchSteam     uint64
	OpeningKill                     bool
	LastTickProcessed               int
	TicksProcessed                  int
}

// Collector is the oracle's CSC stat pipeline (csc/parser.go handlers and processRoundFinal) over
// facts. It implements match.Collector; Game builds the payload once the match finishes.
type Collector struct {
	potential *round
	flags     roundFlags
	rounds    map[*match.Round]*round
}

// New returns a collector before any round.
func New() *Collector { return &Collector{rounds: map[*match.Round]*round{}} }

func missing(what string, steam uint64) error {
	return fmt.Errorf("no player stats for %s %d", what, steam) // the oracle dereferences nil here
}

func (c *Collector) Open(m *match.Match, r *match.Round, s fact.Snapshot, pistol bool) error {
	pr := &round{RoundNum: int8(r.Num), StartingTick: r.StartTick, PlayerStats: map[uint64]*playerStats{},
		TeamStats: map[string]*teamStats{}, WPAlog: make([]*wpalog, 0),
		InitTerroristCount: r.InitT, InitCTerroristCount: r.InitCT}
	for id, mem := range r.Members {
		pr.PlayerStats[id] = newPlayer(mem)
	}
	pr.TeamStats[r.TClan] = &teamStats{TR: 1}
	pr.TeamStats[r.CTClan] = &teamStats{CtR: 1}
	if pistol {
		for _, t := range pr.TeamStats {
			t.Pistols = 1
		}
	}
	c.potential = pr
	c.rounds[r] = pr
	c.flags = roundFlags{PrePlant: true, OpeningKill: true, TAlive: r.InitT, CtAlive: r.InitCT}
	return nil
}

func newPlayer(mem match.Member) *playerStats {
	return &playerStats{Name: mem.Name, SteamID: strconv.FormatUint(mem.Steam64, 10), IsBot: mem.IsBot,
		Side: int(mem.Side), TeamENUM: mem.TeamID, TeamClanName: mem.Clan, Health: 100,
		TradeList: map[uint64]int{}, DamageList: map[uint64]int{}}
}

func (c *Collector) Join(m *match.Match, r *match.Round, mem match.Member) error {
	if pr := c.rounds[r]; pr != nil {
		pr.PlayerStats[mem.Steam64] = newPlayer(mem)
	}
	return nil
}

func (c *Collector) RoundEnd(m *match.Match, f fact.RoundEnd) error {
	if f.Winner == fact.SideT {
		c.flags.TMoney = true
	}
	return nil
}

// Win is the stats part of the oracle's processRoundOnWinCon.
func (c *Collector) Win(m *match.Match, r *match.Round, f fact.RoundEnd) error {
	pr := c.potential
	pr.WinnerENUM = int(f.Winner)
	pr.RoundEndReason = roundEndReasons[f.Reason]
	pr.WinnerClanName = r.WinnerClan
	c.flags.PrePlant, c.flags.PostPlant, c.flags.PostWinCon = false, false, true
	return nil
}

// Commit is the integrity branch of the oracle's processRoundFinal.
func (c *Collector) Commit(m *match.Match, r *match.Round, tick int) error {
	pr := c.rounds[r]
	if pr == nil {
		return fmt.Errorf("round %d committed without an open", r.Num)
	}
	pr.EndingTick = tick
	pr.IntegrityCheck = true
	ts := pr.TeamStats[pr.WinnerClanName]
	if ts == nil {
		return fmt.Errorf("round %d: no team stats for winner %q", r.Num, pr.WinnerClanName)
	}
	if ts.Pistols == 1 {
		ts.PistolsW = 1
	}
	if ts.FourVFiveS == 1 {
		ts.FourVFiveW = 1
	} else if ts.FiveVFourS == 1 {
		ts.FiveVFourW = 1
	}
	if ts.TR == 1 {
		ts.TRW = 1
	} else if ts.CtR == 1 {
		ts.CtRW = 1
	}

	for _, cl := range []struct {
		side  int
		val   int
		steam uint64
	}{{2, c.flags.TClutchVal, c.flags.TClutchSteam}, {3, c.flags.CtClutchVal, c.flags.CtClutchSteam}} {
		if pr.WinnerENUM != cl.side || cl.steam == 0 {
			continue
		}
		ts.Clutches = 1
		p := pr.PlayerStats[cl.steam]
		if p == nil {
			return missing("clutcher", cl.steam)
		}
		if cl.val >= len(clutchBonus) {
			return fmt.Errorf("round %d: clutch 1v%d", r.Num, cl.val)
		}
		p.ImpactPoints += clutchBonus[cl.val]
		switch cl.val {
		case 1:
			p.Cl_1 = 1
		case 2:
			p.Cl_2 = 1
		case 3:
			p.Cl_3 = 1
		case 4:
			p.Cl_4 = 1
		case 5:
			p.Cl_5 = 1
		}
		break // the oracle's if/else-if: T first, then CT
	}

	highest := 0.0
	for _, p := range pr.PlayerStats {
		if p.Deaths == 0 {
			p.KastRounds = 1
			if p.TeamENUM != pr.WinnerENUM {
				p.Saves = 1
				t := pr.TeamStats[p.TeamClanName]
				if t == nil {
					return fmt.Errorf("round %d: no team stats for %q", r.Num, p.TeamClanName)
				}
				t.Saves = 1
			}
		}
		p.ImpactPoints += p.KillPoints
		p.ImpactPoints += float64(p.Damage) / float64(250)
		if int(p.Kills) >= len(multiKillBonus) {
			return fmt.Errorf("round %d: %d kills by %s", r.Num, p.Kills, p.SteamID)
		}
		p.ImpactPoints += multiKillBonus[p.Kills]
		switch p.Kills {
		case 2:
			p.TwoK = 1
		case 3:
			p.ThreeK = 1
		case 4:
			p.FourK = 1
		case 5:
			p.FiveK = 1
		}
		if p.ImpactPoints > highest {
			highest = p.ImpactPoints
		}
		if p.TeamENUM == pr.WinnerENUM {
			p.WinPoints = p.ImpactPoints
			p.RF = 1
		} else {
			p.RA = 1
		}
	}
	mip := 0
	for _, p := range pr.PlayerStats {
		if p.ImpactPoints == highest {
			mip++
		}
	}
	for _, p := range pr.PlayerStats {
		if p.ImpactPoints == highest {
			p.Mip = 1.0 / float64(mip)
		}
	}

	var lurker uint64
	blips, invalid := 0, false
	for _, p := range pr.PlayerStats {
		if p.Side == 2 && p.LurkerBlips > blips {
			blips = p.LurkerBlips
			lurker, _ = strconv.ParseUint(p.SteamID, 10, 64)
		}
	}
	for _, p := range pr.PlayerStats {
		id, _ := strconv.ParseUint(p.SteamID, 10, 64)
		if p.Side == 2 && p.LurkerBlips == blips && id != lurker {
			invalid = true
		}
	}
	if !invalid && blips > 3 {
		pr.PlayerStats[lurker].LurkRounds = 1
	}
	return nil
}

// isDuringExpectedRound is the oracle's check that an event belongs to the open round.
func (c *Collector) isDuringExpectedRound(roundsPlayed int) bool {
	num := int(c.potential.RoundNum)
	return num == roundsPlayed+1 || (num == roundsPlayed && c.flags.PostWinCon)
}

func (c *Collector) Fact(m *match.Match, f fact.Fact) error {
	if !m.Live() {
		return nil
	}
	switch f := f.(type) {
	case fact.Frame:
		return c.lurk(m, f)
	case fact.Kill:
		return c.kill(m, f)
	case fact.Hurt:
		return c.hurt(f)
	case fact.Flashed:
		return c.flashed(f)
	case fact.BombPlanted:
		if !c.flags.PostWinCon {
			if f.Player == nil {
				return fmt.Errorf("bomb planted by nobody")
			}
			c.flags.PrePlant, c.flags.PostPlant, c.flags.TMoney = false, true, true
			c.potential.Planter = f.Player.Steam64
			c.potential.BombStartTick = f.Tick
		}
	case fact.BombDefused:
		if !c.flags.PostWinCon {
			if f.Player == nil {
				return fmt.Errorf("bomb defused by nobody")
			}
			c.flags.PrePlant, c.flags.PostPlant, c.flags.PostWinCon = false, false, true
			c.potential.EndDueToBombEvent = true
			c.potential.Defuser = f.Player.Steam64
			p := c.potential.PlayerStats[f.Player.Steam64]
			if p == nil {
				return missing("defuser", f.Player.Steam64)
			}
			p.ImpactPoints += 0.5
		}
	case fact.BombExplode:
		if !c.flags.PostWinCon {
			c.flags.PrePlant, c.flags.PostPlant, c.flags.PostWinCon = false, false, true
			c.potential.EndDueToBombEvent = true
			if c.potential.Planter != 0 {
				p := c.potential.PlayerStats[c.potential.Planter]
				if p == nil {
					return missing("planter", c.potential.Planter)
				}
				p.ImpactPoints += 0.5
			}
		}
	case fact.GrenadeThrow:
		if f.Weapon == nil {
			return fmt.Errorf("grenade without a weapon at tick %d", f.Tick)
		}
		switch *f.Weapon {
		case 506, 505, 504, 502, 503:
			if f.Thrower == nil {
				return fmt.Errorf("grenade without a thrower at tick %d", f.Tick)
			}
			p := c.potential.PlayerStats[f.Thrower.Steam64]
			if p == nil {
				return missing("thrower", f.Thrower.Steam64)
			}
			switch *f.Weapon {
			case 506:
				p.NadesThrown++
			case 505:
				p.SmokeThrown++
			case 504:
				p.FlashThrown++
			default:
				p.FiresThrown++
			}
		}
	case fact.Disconnect:
		c.flags.TAlive, c.flags.CtAlive = 0, 0
		for _, p := range m.Members(f.Snapshot, fact.SideT) {
			if p.Alive {
				c.flags.TAlive++
			}
		}
		for _, p := range m.Members(f.Snapshot, fact.SideCT) {
			if p.Alive {
				c.flags.CtAlive++
			}
		}
	}
	return nil
}

// lurk is the oracle's FrameDone lurker sampler: every 4 s of a round after the first 18 s, the
// live T furthest from... nearest-but-not-close to their teammates gets a blip.
func (c *Collector) lurk(m *match.Match, f fact.Frame) error {
	if !m.InRound() || c.flags.LastTickProcessed+4*tickRate >= f.Tick {
		return nil
	}
	c.flags.LastTickProcessed = f.Tick
	c.flags.TicksProcessed++
	if c.flags.TAlive <= 2 || c.flags.PostWinCon || f.Tick <= 18*tickRate+c.potential.StartingTick {
		return nil
	}
	pS := c.potential.PlayerStats
	ts := m.Members(f.Sample(), fact.SideT)
	for _, t := range ts {
		if !t.Alive {
			continue
		}
		for _, mate := range ts {
			if t.Steam64 == mate.Steam64 || !mate.Alive {
				continue
			}
			dist := int(distance(t.Pos, mate.Pos))
			if dist < 500 {
				if pS[t.Steam64] == nil {
					return missing("lurk candidate", t.Steam64)
				}
				pS[t.Steam64].DistanceToTeammates = -999999
			}
			if pS[t.Steam64] != nil {
				pS[t.Steam64].DistanceToTeammates += dist
			}
		}
	}
	var lurker uint64
	best := 999999
	for _, t := range ts {
		if t.Alive && pS[t.Steam64] != nil {
			if d := pS[t.Steam64].DistanceToTeammates; d < best && d > 0 {
				best, lurker = d, t.Steam64
			}
		}
	}
	if lurker != 0 {
		pS[lurker].LurkerBlips++
	}
	return nil
}

// distance is r3.Vector.Distance (a.Sub(b).Norm()), with its arithmetic order.
func distance(a, b [3]float64) float64 {
	x, y, z := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(x*x + y*y + z*z)
}

func (c *Collector) hurt(f fact.Hurt) error {
	pr := c.potential
	pS := pr.PlayerStats
	equipment := -999
	if f.Weapon != nil {
		equipment = *f.Weapon
	}
	bombAtHalf := equipment == eqBomb && isRoundFinalInHalf(pr.RoundNum)
	switch {
	case f.Player != nil && pS[f.Player.Steam64] != nil && !bombAtHalf:
		pS[f.Player.Steam64].DamageTaken += f.HealthDamageTaken
	case f.Player != nil && f.Player.Connected && !bombAtHalf && !f.FreezeTime:
		// The oracle cancels the parse (errMissingPlayerStats).
		return missing("connected hurt player", f.Player.Steam64)
	}
	if f.Player != nil && pS[f.Player.Steam64] != nil && f.Attacker != nil && f.Player.Side != f.Attacker.Side {
		a := pS[f.Attacker.Steam64]
		if a == nil {
			return missing("attacker", f.Attacker.Steam64)
		}
		a.Damage += f.HealthDamageTaken
		pS[f.Player.Steam64].DamageList[f.Attacker.Steam64] += f.HealthDamageTaken
		if equipment >= 500 && equipment <= 506 {
			a.UtilDmg += f.HealthDamageTaken
			if equipment == 506 {
				a.NadeDmg += f.HealthDamageTaken
			}
			if equipment == 502 || equipment == 503 {
				a.InfernoDmg += f.HealthDamageTaken
			}
		}
	}
	return nil
}

func (c *Collector) flashed(f fact.Flashed) error {
	if f.Player == nil || f.Attacker == nil {
		return nil
	}
	tick := float64(f.Tick)
	blindTicks := f.Duration * float64(tickRate)
	victim, flasher := f.Player, f.Attacker
	if flasher.Side != victim.Side && blindTicks > float64(tickRate) && victim.Alive && victim.FlashDuration < blindTicks/float64(tickRate)+1 {
		pS := c.potential.PlayerStats
		fp, vp := pS[flasher.Steam64], pS[victim.Steam64]
		if fp == nil {
			return missing("flasher", flasher.Steam64)
		}
		fp.Ef++
		fp.EnemyFlashTime += blindTicks / float64(tickRate)
		if vp == nil {
			return missing("flashed player", victim.Steam64)
		}
		if tick+blindTicks > vp.MostRecentFlashVal {
			vp.MostRecentFlashVal = tick + blindTicks
			vp.MostRecentFlasher = flasher.Steam64
		}
	}
	return nil
}

// kill is the oracle's CSC kill handler.
func (c *Collector) kill(m *match.Match, f fact.Kill) error {
	if !c.isDuringExpectedRound(f.RoundsPlayed) {
		return nil
	}
	pr := c.potential
	pS := pr.PlayerStats
	tick := f.Tick
	if f.Weapon == nil {
		return fmt.Errorf("kill without a weapon at tick %d", tick) // the oracle reads e.Weapon.Type
	}
	killer, victim, assister := f.Killer, f.Victim, f.Assister
	killerExists := killer != nil && pS[killer.Steam64] != nil
	victimExists := victim != nil && pS[victim.Steam64] != nil
	assisterExists := assister != nil && pS[assister.Steam64] != nil
	if *f.Weapon == eqBomb && isRoundFinalInHalf(pr.RoundNum) {
		killerExists, victimExists, assisterExists = false, false, false
	}

	killValue, multiplier := 1.0, 1.0
	traded, assisted, flashAssisted := false, false, false

	if victimExists {
		v := pS[victim.Steam64]
		v.Deaths++
		v.DeathTick = tick
		switch victim.Side {
		case fact.SideT:
			c.flags.TAlive--
			v.DeathPlacement = float64(pr.InitTerroristCount - c.flags.TAlive)
		case fact.SideCT:
			c.flags.CtAlive--
			v.DeathPlacement = float64(pr.InitCTerroristCount - c.flags.CtAlive)
		}

		if c.flags.OpeningKill && pr.InitCTerroristCount+pr.InitTerroristCount == 10 {
			team := v.TeamClanName
			t := pr.TeamStats[team]
			if t == nil {
				return fmt.Errorf("round %d: no team stats for %q", pr.RoundNum, team)
			}
			t.FourVFiveS = 1
			for name, other := range pr.TeamStats {
				if name != team {
					other.FiveVFourS = 1
				}
			}
		}

		for supp, dmg := range v.DamageList {
			if killerExists && supp == killer.Steam64 {
				continue
			}
			s := pS[supp]
			if s == nil {
				return missing("supporter", supp)
			}
			s.SuppDamage += dmg
			if s.SuppDamage > 60 {
				s.SuppRounds = 1
			}
		}

		if !c.flags.PostWinCon {
			if c.flags.TAlive == 1 && c.flags.TClutchVal == 0 {
				c.flags.TClutchVal = c.flags.CtAlive
				for _, t := range m.Members(f.Snapshot, fact.SideT) {
					if t.Alive && victim.Steam64 != t.Steam64 {
						c.flags.TClutchSteam = t.Steam64
					}
				}
			}
			if c.flags.CtAlive == 1 && c.flags.CtClutchVal == 0 {
				c.flags.CtClutchVal = c.flags.TAlive
				for _, ct := range m.Members(f.Snapshot, fact.SideCT) {
					if ct.Alive && victim.Steam64 != ct.Steam64 {
						c.flags.CtClutchSteam = ct.Steam64
					}
				}
			}
		}

		v.TicksAlive = tick - pr.StartingTick
		for dead, deadTick := range v.TradeList {
			if tick-deadTick < tradeCutoff*tickRate {
				d := pS[dead]
				if d == nil {
					return missing("traded player", dead)
				}
				d.Traded = 1
				d.Eac++
				d.KastRounds = 1
			}
		}
	}

	if assisterExists && victimExists && assister.TeamStateID != victim.TeamStateID {
		a := pS[assister.Steam64]
		a.Assists++
		a.Eac++
		a.KastRounds = 1
		a.SuppRounds = 1
		assisted = true
		if f.AssistedFlash {
			a.FAss++
			flashAssisted = true
		} else if float64(tick) < pS[victim.Steam64].MostRecentFlashVal {
			fl := pS[pS[victim.Steam64].MostRecentFlasher]
			if fl == nil {
				return missing("flash assister", pS[victim.Steam64].MostRecentFlasher)
			}
			fl.FAss++
			fl.Eac++
			fl.SuppRounds = 1
			flashAssisted = true
		}
	}

	if killerExists && victimExists && killer.TeamStateID != victim.TeamStateID {
		k, v := pS[killer.Steam64], pS[victim.Steam64]
		k.Kills++
		k.KastRounds = 1
		k.Rwk = 1
		k.TradeList[victim.Steam64] = tick
		if *f.Weapon == eqAWP {
			k.AwpKills++
			if killer.Side == fact.SideCT {
				k.CtAWP++
			}
		}
		if f.Headshot {
			k.Hs++
		}
		for _, deadTick := range v.TradeList {
			if tick-deadTick < tradeCutoff*tickRate {
				k.Trades++
				traded = true
				break
			}
		}

		switch {
		case c.flags.PrePlant:
			if killer.Side == fact.SideT {
				killValue = 1.2
			} else if killer.Side == fact.SideCT {
				killValue = 1
			}
		case c.flags.PostPlant:
			if killer.Side == fact.SideT {
				killValue = 1
			} else if killer.Side == fact.SideCT {
				killValue = 1.2
			}
		case c.flags.PostWinCon:
			switch pr.WinnerENUM {
			case 2:
				if killer.Side == fact.SideT {
					killValue = 0.8
				}
				if killer.Side == fact.SideCT {
					killValue = 0.6
				}
			case 3:
				if killer.Side == fact.SideT {
					killValue = 0.5
				}
				if killer.Side == fact.SideCT {
					if c.flags.TMoney {
						killValue = 0.6
					} else {
						killValue = 0.8
					}
				}
			}
		}

		if c.flags.OpeningKill {
			c.flags.OpeningKill = false
			k.Ok = 1
			v.Ol = 1
			if killer.Side == fact.SideT {
				if c.flags.PrePlant {
					multiplier += 0.8
					k.Entries = 1
				} else {
					multiplier += 0.3
				}
			} else if killer.Side == fact.SideCT {
				multiplier += 0.5
			}
		} else if traded {
			multiplier += 0.3
		}
		if flashAssisted {
			multiplier += 0.2
		}
		if assisted {
			killValue -= 0.15
			pS[assister.Steam64].ImpactPoints += 0.15
		}
		killValue *= multiplier

		ecoRatio := float64(victim.EquipValue) / float64(killer.EquipValue)
		ecoMod := 1.0
		if ecoRatio > 4 {
			ecoMod += 0.25
		} else if ecoRatio > 2 {
			ecoMod += 0.14
		} else if ecoRatio < 0.25 {
			ecoMod -= 0.25
		} else if ecoRatio < 0.5 {
			ecoMod -= 0.14
		}
		killValue *= ecoMod
		k.KillPoints += killValue
	}
	return nil
}
