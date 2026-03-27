// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package parser provides CS2 demo file parsing functionality.
// This file implements a Logger for detailed parsing output with player filtering
// and formatted event logging (kills, deaths, trades, clutches, etc.).
package parser

import (
	"bytes"
	"log"
)

// ParserLogger defines the interface for demo parsing logging.
// This allows for a no-op implementation when logging is disabled.
type ParserLogger interface {
	GetOutput() string
	ClearOutput()
	SetPlayerFilter(players []string)
	AddPlayerFilter(player string)
	ClearPlayerFilter()
	SetEnabled(enabled bool)
	LogKill(round int, killer, victim string, killerEquip, victimEquip int, killValue float64)
	LogDeath(round int, victim, killer string, victimEquip, killerEquip int, deathPenalty float64)
	LogRoundStart(round int)
	LogRoundEnd(round int)
	LogTrade(round int, trader, tradedPlayer, originalKiller string)
	LogOpeningKill(round int, killer, victim string)
	LogMultiKill(round int, player string, kills int)
	LogPlayerSummary(name string, kills, deaths, damage int, ecoKillValue, ecoDeathValue, finalRating float64)
	LogBombPlant(round int, planter string)
	LogBombDefuse(round int, defuser string)
	LogKnifeRound()
	LogWarmup()
	Printf(format string, v ...interface{})
}

// noOpLogger is a no-op implementation that does nothing.
// Used when logging is disabled to avoid any overhead.
type noOpLogger struct{}

func (n *noOpLogger) GetOutput() string                { return "" }
func (n *noOpLogger) ClearOutput()                     {}
func (n *noOpLogger) SetPlayerFilter(players []string) {}
func (n *noOpLogger) AddPlayerFilter(player string)    {}
func (n *noOpLogger) ClearPlayerFilter()               {}
func (n *noOpLogger) SetEnabled(enabled bool)          {}
func (n *noOpLogger) LogKill(round int, killer, victim string, killerEquip, victimEquip int, killValue float64) {
}
func (n *noOpLogger) LogDeath(round int, victim, killer string, victimEquip, killerEquip int, deathPenalty float64) {
}
func (n *noOpLogger) LogRoundStart(round int)                                         {}
func (n *noOpLogger) LogRoundEnd(round int)                                           {}
func (n *noOpLogger) LogTrade(round int, trader, tradedPlayer, originalKiller string) {}
func (n *noOpLogger) LogOpeningKill(round int, killer, victim string)                 {}
func (n *noOpLogger) LogMultiKill(round int, player string, kills int)                {}
func (n *noOpLogger) LogPlayerSummary(name string, kills, deaths, damage int, ecoKillValue, ecoDeathValue, finalRating float64) {
}
func (n *noOpLogger) LogBombPlant(round int, planter string)  {}
func (n *noOpLogger) LogBombDefuse(round int, defuser string) {}
func (n *noOpLogger) LogKnifeRound()                          {}
func (n *noOpLogger) LogWarmup()                              {}
func (n *noOpLogger) Printf(format string, v ...interface{})  {}

// sharedNoOpLogger is a singleton no-op logger to avoid allocations.
var sharedNoOpLogger = &noOpLogger{}

// Logger provides formatted logging for demo parsing events.
// It supports player filtering to focus output on specific players.
type Logger struct {
	enabled      bool            // Whether logging is active
	logger       *log.Logger     // Underlying logger instance
	buffer       *bytes.Buffer   // Buffer to capture log output
	playerFilter map[string]bool // Set of player names to filter (empty = log all)
}

// NewLogger creates a new Logger with the specified enabled state.
// Output is captured to an internal buffer for later retrieval.
// When disabled, returns a no-op logger to avoid any overhead.
func NewLogger(enabled bool) ParserLogger {
	if !enabled {
		return sharedNoOpLogger
	}
	buf := &bytes.Buffer{}
	return &Logger{
		enabled:      true,
		logger:       log.New(buf, "", 0),
		buffer:       buf,
		playerFilter: make(map[string]bool),
	}
}

// GetOutput returns all captured log output as a string.
func (l *Logger) GetOutput() string {
	return l.buffer.String()
}

// ClearOutput resets the log buffer, discarding all captured output.
func (l *Logger) ClearOutput() {
	l.buffer.Reset()
}

// SetPlayerFilter sets the list of player names to include in logging.
// Only events involving these players will be logged.
func (l *Logger) SetPlayerFilter(players []string) {
	l.playerFilter = make(map[string]bool)
	for _, p := range players {
		l.playerFilter[p] = true
	}
}

// AddPlayerFilter adds a single player to the filter list.
func (l *Logger) AddPlayerFilter(player string) {
	l.playerFilter[player] = true
}

// ClearPlayerFilter removes all player filters, allowing all events to be logged.
func (l *Logger) ClearPlayerFilter() {
	l.playerFilter = make(map[string]bool)
}

// shouldLog returns true if logging is enabled and any of the given players
// pass the filter (or if no filter is set).
func (l *Logger) shouldLog(players ...string) bool {
	if !l.enabled {
		return false
	}
	if len(l.playerFilter) == 0 {
		return true
	}
	for _, p := range players {
		if l.playerFilter[p] {
			return true
		}
	}
	return false
}

// LogKill logs a kill event with equipment values and economic impact.
func (l *Logger) LogKill(round int, killer, victim string, killerEquip, victimEquip int, killValue float64) {
	if !l.shouldLog(killer, victim) {
		return
	}

	ratio := float64(victimEquip) / float64(max(killerEquip, 100))
	ecoType := getEcoType(ratio)

	l.logger.Printf("┌─ KILL ─────────────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s killed %s", round, killer, victim)
	l.logger.Printf("│ Equipment: $%d vs $%d (ratio: %.2f)", killerEquip, victimEquip, ratio)
	l.logger.Printf("│ Type: %s", ecoType)
	l.logger.Printf("│ Kill Value: %.2fx", killValue)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogDeath logs a death event with equipment values and penalty calculation.
func (l *Logger) LogDeath(round int, victim, killer string, victimEquip, killerEquip int, deathPenalty float64) {
	if !l.shouldLog(victim, killer) {
		return
	}

	ratio := float64(victimEquip) / float64(max(killerEquip, 100))
	ecoType := getDeathType(ratio)

	l.logger.Printf("┌─ DEATH ────────────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s died to %s", round, victim, killer)
	l.logger.Printf("│ Equipment: $%d vs $%d (ratio: %.2f)", victimEquip, killerEquip, ratio)
	l.logger.Printf("│ Type: %s", ecoType)
	l.logger.Printf("│ Death Penalty: %.2fx", deathPenalty)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogRoundStart logs the beginning of a new round.
func (l *Logger) LogRoundStart(round int) {
	if !l.enabled {
		return
	}
	l.logger.Printf("")
	l.logger.Printf("══════════════════════════════════════════════════════════")
	l.logger.Printf("                      ROUND %d START", round)
	l.logger.Printf("══════════════════════════════════════════════════════════")
}

// LogRoundEnd logs the end of a round.
func (l *Logger) LogRoundEnd(round int) {
	if !l.enabled {
		return
	}
	l.logger.Printf("══════════════════════════════════════════════════════════")
	l.logger.Printf("                      ROUND %d END", round)
	l.logger.Printf("══════════════════════════════════════════════════════════")
	l.logger.Printf("")
}

// LogTrade logs a trade kill (avenging a teammate's death within the trade window).
func (l *Logger) LogTrade(round int, trader, tradedPlayer, originalKiller string) {
	if !l.shouldLog(trader, tradedPlayer, originalKiller) {
		return
	}
	l.logger.Printf("┌─ TRADE ────────────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s traded %s (killed %s)", round, trader, tradedPlayer, originalKiller)
	l.logger.Printf("│ %s gets KAST credit, %s gets TradeDenial", tradedPlayer, trader)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogOpeningKill logs the first kill of a round.
func (l *Logger) LogOpeningKill(round int, killer, victim string) {
	if !l.shouldLog(killer, victim) {
		return
	}
	l.logger.Printf("┌─ OPENING KILL ─────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s got the opening kill on %s", round, killer, victim)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogMultiKill logs a multi-kill round (2K, 3K, 4K, or ACE).
func (l *Logger) LogMultiKill(round int, player string, kills int) {
	if !l.shouldLog(player) {
		return
	}
	killType := ""
	switch kills {
	case 2:
		killType = "Double Kill"
	case 3:
		killType = "Triple Kill"
	case 4:
		killType = "Quad Kill"
	case 5:
		killType = "ACE"
	}
	l.logger.Printf("┌─ MULTI-KILL ───────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s got a %s (%d kills)", round, player, killType, kills)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogPlayerSummary logs end-of-game statistics for a player.
func (l *Logger) LogPlayerSummary(name string, kills, deaths, damage int, ecoKillValue, ecoDeathValue, finalRating float64) {
	if !l.shouldLog(name) {
		return
	}
	l.logger.Printf("┌─ PLAYER SUMMARY: %s", name)
	l.logger.Printf("│ K/D: %d/%d | Damage: %d", kills, deaths, damage)
	l.logger.Printf("│ Eco Kill Value: %.2f | Eco Death Value: %.2f", ecoKillValue, ecoDeathValue)
	l.logger.Printf("│ Final Rating: %.2f", finalRating)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogBombPlant logs a bomb plant event.
func (l *Logger) LogBombPlant(round int, planter string) {
	if !l.shouldLog(planter) {
		return
	}
	l.logger.Printf("┌─ BOMB PLANT ───────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s planted the bomb", round, planter)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogBombDefuse logs a bomb defuse event.
func (l *Logger) LogBombDefuse(round int, defuser string) {
	if !l.shouldLog(defuser) {
		return
	}
	l.logger.Printf("┌─ BOMB DEFUSE ──────────────────────────────────────────")
	l.logger.Printf("│ Round %d: %s defused the bomb", round, defuser)
	l.logger.Printf("└────────────────────────────────────────────────────────")
}

// LogKnifeRound logs detection of a knife round (stats not tracked).
func (l *Logger) LogKnifeRound() {
	if !l.enabled {
		return
	}
	l.logger.Printf("⚔️  KNIFE ROUND DETECTED - Skipping stats tracking")
}

// LogWarmup logs detection of warmup period (stats not tracked).
func (l *Logger) LogWarmup() {
	if !l.enabled {
		return
	}
	l.logger.Printf("🔥 WARMUP DETECTED - Skipping stats tracking")
}

// getEcoType returns a descriptive string for the economic advantage of a kill.
func getEcoType(ratio float64) string {
	if ratio > 4.0 {
		return "🎯 PISTOL VS RIFLE (1.80x bonus)"
	} else if ratio > 2.0 {
		return "💰 ECO VS FORCE/FULL (1.50x bonus)"
	} else if ratio > 1.3 {
		return "📈 FORCE VS FULL BUY (1.25x bonus)"
	} else if ratio > 1.1 {
		return "📊 SLIGHT DISADVANTAGE (1.10x bonus)"
	} else if ratio > 0.9 {
		return "⚖️  EQUAL FIGHT (1.00x)"
	} else if ratio > 0.75 {
		return "📉 SLIGHT ADVANTAGE (0.95x)"
	} else if ratio > 0.5 {
		return "🔽 EQUIPMENT ADVANTAGE (0.85x)"
	} else {
		return "🔫 RIFLE VS PISTOL (0.70x reduced)"
	}
}

// getDeathType returns a descriptive string for the economic context of a death.
func getDeathType(ratio float64) string {
	if ratio > 4.0 {
		return "💀 DIED TO PISTOL WITH RIFLE (1.60x penalty)"
	} else if ratio > 2.0 {
		return "😬 DIED TO ECO (1.40x penalty)"
	} else if ratio > 1.3 {
		return "😤 FULL BUY DIED TO FORCE (1.20x penalty)"
	} else if ratio > 1.1 {
		return "📉 SLIGHT ADVANTAGE DEATH (1.10x penalty)"
	} else if ratio > 0.9 {
		return "⚖️  EQUAL FIGHT DEATH (1.00x)"
	} else if ratio > 0.75 {
		return "📊 SLIGHT DISADVANTAGE DEATH (0.95x)"
	} else if ratio > 0.5 {
		return "📈 DIED TO BETTER EQUIPPED (0.85x)"
	} else {
		return "🛡️  ECO VS RIFLE DEATH (0.70x reduced)"
	}
}

// Disable turns off logging.
func (l *Logger) Disable() {
	l.enabled = false
}

// Enable turns on logging.
func (l *Logger) Enable() {
	l.enabled = true
}

// SetEnabled sets the logging state.
func (l *Logger) SetEnabled(enabled bool) {
	l.enabled = enabled
}

// Printf logs a formatted message if logging is enabled.
func (l *Logger) Printf(format string, v ...interface{}) {
	if l.enabled {
		l.logger.Printf(format, v...)
	}
}
