package parser

import (
	"fmt"

	"github.com/csconfederation/fragg-3.0/engine/eco"
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/model"
	"github.com/csconfederation/fragg-3.0/rating"
	"github.com/csconfederation/fragg-3.0/rating/probability"
	"github.com/csconfederation/fragg-3.0/rating/swing"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

// RoundSnapshot is a deep copy of one round's eco stats, captured when CSC
// commits a round. After removeInvalidRounds, only surviving snapshots are folded.
type RoundSnapshot struct {
	RoundNumber int
	Winner      common.Team
	MapName     string
	TAlive      int
	CTAlive     int
	BombPlanted bool
	Pending     probability.PendingRound
	Stats       map[uint64]*model.RoundStats
}

// AttachToParser registers eco event handlers on an existing demoinfocs parser.
// Round lifecycle is not registered: the caller drives BeginRound / FinalizeRound
// / SnapshotRound from CSC hooks.
func AttachToParser(p *demoinfocs.Parser, enableLogging, kdprModifier bool, swingCfg swing.Config) *DemoParser {
	state := NewMatchStateWithConfig(swingCfg)
	dp := &DemoParser{
		parser:            p,
		state:             state,
		logger:            NewLogger(enableLogging),
		collector:         probability.NewDataCollector(),
		kdprModifier:      kdprModifier,
		externalLifecycle: true,
	}
	dp.state.MatchStarted = true
	dp.registerHandlers()
	return dp
}

// BeginRound resets round-scoped eco state. CSC calls this from initRound;
// standalone parsing calls it from freeze-time end.
func (d *DemoParser) BeginRound() {
	d.resetRoundState()
	d.state.RoundActive = true
	d.state.IsKnifeRound = false
	d.state.RoundNumber++
	d.state.IsPistolRound = rating.IsPistolRound(d.state.RoundNumber)
	d.state.RoundStartTime = d.currentTime()

	gs := d.parser.GameState()
	participants := gs.Participants().Playing()

	for _, p := range participants {
		if p.Team == common.TeamTerrorists {
			d.state.CurrentSide = "T"
			break
		} else if p.Team == common.TeamCounterTerrorists {
			d.state.CurrentSide = "CT"
			break
		}
	}

	d.logger.LogRoundStart(d.state.RoundNumber)

	tAlive := 0
	ctAlive := 0
	tEquipTotal := 0
	ctEquipTotal := 0

	for _, p := range participants {
		if p.IsBot {
			continue
		}
		d.state.ensurePlayer(p)
		roundStats := d.state.ensureRound(p)
		roundStats.IsPistolRound = d.state.IsPistolRound

		if p.Team == common.TeamTerrorists {
			roundStats.PlayerSide = "T"
			tAlive++
			tEquipTotal += p.EquipmentValueCurrent()
		} else if p.Team == common.TeamCounterTerrorists {
			roundStats.PlayerSide = "CT"
			ctAlive++
			ctEquipTotal += p.EquipmentValueCurrent()
		}
	}

	if tAlive > 5 {
		tAlive = 5
	}
	if ctAlive > 5 {
		ctAlive = 5
	}

	if d.collector != nil {
		d.collector.RecordRoundStart(tAlive, ctAlive, false, d.state.MapName)
	}

	if d.state.SwingTracker != nil && d.state.SwingTracker.IsEnabled() {
		startTick := gs.IngameTick()
		d.state.SwingTracker.ResetRoundWithClock(d.state.RoundNumber, tAlive, ctAlive, d.state.MapName, startTick, eco.DefaultRoundTimeSeconds)

		tAvgEquip := 0.0
		ctAvgEquip := 0.0
		if tAlive > 0 {
			tAvgEquip = float64(tEquipTotal) / float64(tAlive)
		}
		if ctAlive > 0 {
			ctAvgEquip = float64(ctEquipTotal) / float64(ctAlive)
		}
		d.state.SwingTracker.SetEconomyFromValues(tAvgEquip, ctAvgEquip)
	}
}

func (d *DemoParser) resetRoundState() {
	d.state.Round = make(map[uint64]*model.RoundStats)
	d.state.RoundHasKill = false
	d.state.TradeDetector.Reset()
	d.state.RoundDecided = false
	d.state.RoundDecidedAt = 0
	d.state.BombPlanted = false
	d.state.RoundActive = false
}

// FinalizeRound applies round-end residual swing and round-scoped flags without
// folding into match totals. winnerENUM is 2 (T) or 3 (CT).
func (d *DemoParser) FinalizeRound(winnerENUM int) {
	if !d.state.RoundActive {
		return
	}
	if d.parser.GameState().IsWarmupPeriod() {
		return
	}

	d.lastWinner = winnerENUM
	ctx := d.buildRoundEndContextFromWinner(common.Team(winnerENUM))
	d.processRoundEndTrades()
	d.processMultiKills()
	d.processSurvivalStats(ctx)
	d.processClutchDetection(ctx)
	d.processRoundEndSwing(ctx)
}

func (d *DemoParser) buildRoundEndContextFromWinner(winner common.Team) *roundEndContext {
	gs := d.parser.GameState()
	roundDuration := d.timeInRound()
	timeRemaining := 0.0
	if roundDuration < 115.0 {
		timeRemaining = 115.0 - roundDuration
	}
	return &roundEndContext{
		gs:            gs,
		winnerTeam:    winner,
		roundDuration: roundDuration,
		timeRemaining: timeRemaining,
	}
}

// SnapshotRound deep-copies the current round map for CSC to stash on the round
// object. Safe to call after FinalizeRound.
func (d *DemoParser) SnapshotRound() *RoundSnapshot {
	src := d.state.Round
	dst := make(map[uint64]*model.RoundStats, len(src))
	for id, rs := range src {
		dst[id] = cloneRoundStats(rs)
	}

	gs := d.parser.GameState()
	tAlive, ctAlive := d.state.CountAlivePlayers(gs.Participants().Playing())
	var pending probability.PendingRound
	if d.collector != nil {
		pending = d.collector.TakePending()
	}
	return &RoundSnapshot{
		RoundNumber: d.state.RoundNumber,
		Winner:      common.Team(d.lastWinner),
		MapName:     d.state.MapName,
		TAlive:      tAlive,
		CTAlive:     ctAlive,
		BombPlanted: d.state.BombPlanted,
		Pending:     pending,
		Stats:       dst,
	}
}

func cloneRoundStats(src *model.RoundStats) *model.RoundStats {
	if src == nil {
		return nil
	}
	dst := *src
	if src.KillTimes != nil {
		dst.KillTimes = append([]float64(nil), src.KillTimes...)
	}
	if src.SwingContributions != nil {
		dst.SwingContributions = append([]model.SwingContribution(nil), src.SwingContributions...)
	}
	return &dst
}

// FoldSnapshots folds surviving round snapshots into match totals, then sets
// RoundsCounted to the number of snapshots (the post-dedup round count).
func (d *DemoParser) FoldSnapshots(snaps []*RoundSnapshot) {
	for _, snap := range snaps {
		if snap == nil {
			continue
		}
		d.foldSnapshot(snap)
	}
	d.state.RoundsCounted = len(snaps)
}

// FinalizeStats computes derived ratings after FoldSnapshots.
func (d *DemoParser) FinalizeStats() {
	d.computeDerivedStats()
}

func (d *DemoParser) foldSnapshot(snap *RoundSnapshot) {
	for steamID, roundStats := range snap.Stats {
		if roundStats == nil {
			continue
		}
		player := d.playerForFold(steamID)
		eco.FoldRound(player, roundStats, snap.RoundNumber)
		updater := eco.NewSideStatsUpdater(player, roundStats)
		updater.UpdateCommonRoundStats()
		updater.UpdateSideStats()
		player.RoundsPlayed++
	}
	if d.collector != nil {
		d.collector.CommitPending(snap.Pending, snap.TAlive, snap.CTAlive, snap.BombPlanted, fact.Side(snap.Winner), snap.MapName)
	}
}

func (d *DemoParser) playerForFold(steamID uint64) *model.PlayerStats {
	if p, ok := d.state.Players[steamID]; ok && p != nil {
		return p
	}
	p := &model.PlayerStats{SteamID: fmt.Sprintf("%d", steamID)}
	d.state.Players[steamID] = p
	return p
}
