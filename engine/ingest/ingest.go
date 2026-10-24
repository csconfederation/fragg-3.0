// Package ingest turns a CS2 demo into facts. It is the engine's only demoinfocs importer
// (Core-planning#301, #575).
package ingest

import (
	"context"
	"errors"
	"io"

	"github.com/csconfederation/fragg-3.0/engine/fact"

	dem "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

// Run parses the demo and passes each fact to sink in dispatch order. It returns how the stream
// ended and the first error sink returned, which stops the parse. Handlers don't panic.
func Run(ctx context.Context, r io.Reader, sink fact.Sink) (fact.Terminal, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	p := dem.NewParser(r,
		option.IgnorePacketEntitiesPanic(true),
		option.UserCmdParsing(option.UserCmdParsingDisabled))
	defer p.Close()

	var sinkErr error
	emit := func(f fact.Fact) {
		if sinkErr != nil {
			return
		}
		if err := sink.Fact(f); err != nil {
			sinkErr = err
			cancel(err)
		}
	}
	gs := p.GameState()
	tick := gs.IngameTick
	snap := func() fact.Snapshot { return snapshot(p) }

	p.RegisterNetMessageHandler(func(m *msg.CDemoFileHeader) {
		emit(fact.Header{ServerName: m.GetServerName()})
	})
	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		emit(fact.ServerInfo{Map: m.GetMapName()})
	})
	p.RegisterEventHandler(func(events.RoundStart) {
		emit(fact.RoundStart{Tick: tick(), RoundsPlayed: gs.TotalRoundsPlayed()})
	})
	p.RegisterEventHandler(func(events.FrameDone) {
		emit(fact.Frame{Tick: tick(), Frame: p.CurrentFrame(), Sample: snap})
	})
	p.RegisterEventHandler(func(events.RoundFreezetimeEnd) {
		emit(fact.FreezeEnd{Snapshot: snap()})
	})
	p.RegisterEventHandler(func(e events.RoundEnd) {
		emit(fact.RoundEnd{
			Snapshot:   snap(),
			Winner:     fact.Side(e.Winner),
			Reason:     int(e.Reason),
			WinnerTeam: team(e.WinnerState),
			LoserTeam:  team(e.LoserState),
		})
	})
	p.RegisterEventHandler(func(e events.ScoreUpdated) {
		if e.TeamState == nil {
			return
		}
		t := team(e.TeamState)
		t.Score = e.NewScore
		emit(fact.ScoreUpdated{Tick: tick(), Team: t, Opponent: team(e.TeamState.Opponent)})
	})
	p.RegisterEventHandler(func(events.RoundEndOfficial) {
		emit(fact.RoundEndOfficial{Snapshot: snap()})
	})
	p.RegisterEventHandler(func(e events.Kill) {
		f := fact.Kill{Snapshot: snap(), Killer: actor(e.Killer), Victim: actor(e.Victim), Assister: actor(e.Assister),
			Weapon: equipment(e.Weapon), Headshot: e.IsHeadshot, AssistedFlash: e.AssistedFlash}
		if f.Victim != nil {
			for _, w := range e.Victim.Weapons() {
				if w != nil {
					f.Victim.Weapons = append(f.Victim.Weapons, int(w.Type))
				}
			}
		}
		emit(f)
	})
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		emit(fact.Hurt{Snapshot: snap(), Player: actor(e.Player), Attacker: actor(e.Attacker),
			Weapon: equipment(e.Weapon), HealthDamageTaken: e.HealthDamageTaken})
	})
	p.RegisterEventHandler(func(e events.PlayerFlashed) {
		emit(fact.Flashed{Tick: tick(), Player: actor(e.Player), Attacker: actor(e.Attacker),
			Duration: e.FlashDuration().Seconds()})
	})
	p.RegisterEventHandler(func(e events.BombPlanted) {
		emit(fact.BombPlanted{Snapshot: snap(), Player: actor(e.Player)})
	})
	p.RegisterEventHandler(func(e events.BombDefused) {
		emit(fact.BombDefused{Snapshot: snap(), Player: actor(e.Player)})
	})
	p.RegisterEventHandler(func(e events.BombExplode) {
		emit(fact.BombExplode{Snapshot: snap(), Player: actor(e.Player)})
	})
	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		f := fact.GrenadeThrow{Tick: tick()}
		if e.Projectile != nil {
			f.Thrower = actor(e.Projectile.Thrower)
			f.Weapon = equipment(e.Projectile.WeaponInstance)
		}
		emit(f)
	})
	p.RegisterEventHandler(func(e events.PlayerTeamChange) {
		if e.Player == nil {
			return
		}
		emit(fact.TeamChange{Tick: tick(), Player: player(e.Player, false), NewSide: fact.Side(e.NewTeam), NewTeam: team(e.NewTeamState)})
	})
	p.RegisterEventHandler(func(e events.PlayerInfo) {
		f := fact.PlayerInfo{Tick: tick(), XUID: e.Info.XUID}
		if pl := gs.Participants().AllByUserID()[e.Index]; pl != nil {
			f.Found, f.Resolved = true, pl.SteamID64
		}
		emit(f)
	})
	p.RegisterEventHandler(func(e events.PlayerConnect) {
		if e.Player != nil {
			emit(fact.Connect{Tick: tick(), Steam64: e.Player.SteamID64})
		}
	})
	p.RegisterEventHandler(func(e events.PlayerDisconnected) {
		if e.Player != nil {
			emit(fact.Disconnect{Snapshot: snap(), Steam64: e.Player.SteamID64})
		}
	})
	p.RegisterEventHandler(func(events.Footstep) {
		emit(fact.Footstep{Tick: tick()})
	})
	p.RegisterEventHandler(func(e events.GamePhaseChanged) {
		emit(fact.GamePhase{Tick: tick(), Phase: int(e.NewGamePhase)})
	})
	p.RegisterEventHandler(func(events.AnnouncementWinPanelMatch) {
		emit(fact.WinPanelMatch{Tick: tick()})
	})

	err := p.ParseToEnd(ctx)
	term := terminal(err, ctx.Err() != nil)
	term.End = snap()
	return term, sinkErr
}

// terminal classifies the parse error. A parser error found before a stop keeps its kind; the
// sink's own error is returned separately.
func terminal(err error, stopped bool) fact.Terminal {
	var de *dem.DecodeError
	switch {
	case err == nil:
		return fact.Terminal{Kind: fact.Clean}
	case errors.Is(err, dem.ErrInvalidFileType):
		return fact.Terminal{Kind: fact.InvalidFileType, Err: err}
	case errors.As(err, &de):
		return fact.Terminal{Kind: fact.Decode, Frame: de.Frame, Err: err}
	case errors.Is(err, dem.ErrUnexpectedEndOfDemo):
		return fact.Terminal{Kind: fact.Truncated, Err: err}
	case stopped:
		return fact.Terminal{Kind: fact.Stopped, Err: err}
	default:
		return fact.Terminal{Kind: fact.Failed, Err: err}
	}
}

func team(ts *common.TeamState) fact.Team {
	if ts == nil {
		return fact.Team{}
	}
	return fact.Team{Side: fact.Side(ts.Team()), ID: ts.ID(), Clan: ts.ClanName(), Score: ts.Score()}
}

func player(pl *common.Player, listed bool) fact.Player {
	pos := pl.Position()
	f := fact.Player{
		Steam64:       pl.SteamID64,
		Name:          pl.Name,
		Side:          fact.Side(pl.Team),
		IsBot:         pl.IsBot,
		Alive:         pl.IsAlive(),
		Connected:     pl.IsConnected,
		Listed:        listed,
		Money:         pl.Money(),
		Spent:         pl.MoneySpentThisRound(),
		EquipValue:    pl.EquipmentValueCurrent(),
		Pos:           [3]float64{pos.X, pos.Y, pos.Z},
		FlashDuration: float64(pl.FlashDuration),
	}
	if pl.TeamState != nil {
		f.TeamStateID = pl.TeamState.ID()
	}
	return f
}

// actor is an event's player, nil when the event has none.
func actor(pl *common.Player) *fact.Player {
	if pl == nil {
		return nil
	}
	f := player(pl, false)
	return &f
}

// equipment is an event's weapon type, nil when the event has none.
func equipment(e *common.Equipment) *int {
	if e == nil {
		return nil
	}
	t := int(e.Type)
	return &t
}

// snapshot reads every T and CT participant record, in the game's participant order, marking the
// records the team member lists hold (which are also the game's Playing records).
func snapshot(p *dem.Parser) fact.Snapshot {
	gs := p.GameState()
	t, ct := gs.TeamTerrorists(), gs.TeamCounterTerrorists()
	s := fact.Snapshot{Tick: gs.IngameTick(), Frame: p.CurrentFrame(), RoundsPlayed: gs.TotalRoundsPlayed(),
		Warmup: gs.IsWarmupPeriod(), FreezeTime: gs.IsFreezetimePeriod(), T: team(t), CT: team(ct)}
	for _, pl := range gs.Participants().All() {
		if pl.Team == common.TeamTerrorists || pl.Team == common.TeamCounterTerrorists {
			// TeamState.Members() and Participants().Playing() keep the records ByUserID keeps:
			// connected, with an entity.
			s.Players = append(s.Players, player(pl, pl.IsConnected && pl.Entity != nil))
		}
	}
	return s
}
