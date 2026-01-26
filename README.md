# Eco-Rating Parser

> **DISCLAIMER**: Comments throughout this codebase were generated with AI assistance to help users find and understand code for reference while building FraGG 3.0. There may be mistakes in the comments. Please verify accuracy.

A CS2 demo parser that calculates advanced player performance ratings based on probability-based impact metrics, economic context, and round swing analysis.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Adding New Stats](#adding-new-stats)
- [Rating System](#rating-system)
- [Key Concepts](#key-concepts)

---

## Overview

This parser processes CS2 demo files and computes comprehensive player statistics including:

- **Probability Swing**: How much each action affected win probability
- **Economic Impact**: Equipment-adjusted kill values
- **HLTV Rating**: Standard HLTV 2.0 rating for comparison
- **Round Swing**: Per-round impact score
- **140+ tracked statistics**: Opening kills, trades, clutches, utility, AWP stats, etc.

### Usage

```bash
# Single demo
eco-rating -demo=path/to/demo.dem

# Cumulative mode (batch process from cloud bucket)
eco-rating -cumulative -tier=contender
```

---

## Architecture

```
eco-rating/
├── main.go                 # Entry point, CLI handling
├── config/                 # Configuration loading
├── bucket/                 # Cloud storage client
├── downloader/             # Demo download & extraction
├── parser/                 # Demo parsing (core logic)
│   ├── parser.go           # Main DemoParser struct
│   ├── handlers.go         # Event handlers (kills, damage, rounds)
│   ├── round.go            # MatchState management
│   ├── round_swing.go      # Round swing calculation
│   ├── side_stats.go       # T/CT side stat updates
│   ├── trade_detector.go   # Trade kill detection
│   ├── swing_tracker.go    # Probability swing tracking
│   └── damage_tracker.go   # Damage attribution
├── model/                  # Data structures
│   ├── player_stats.go     # PlayerStats struct (all tracked stats)
│   ├── round_stats.go      # RoundStats struct (per-round data)
│   └── round_context_builder.go
├── rating/                 # Rating calculations
│   ├── rating.go           # Final rating computation
│   ├── weights.go          # ALL constants and weights
│   ├── economy.go          # Economic kill/death values
│   ├── hltv.go             # HLTV 2.0 rating calculation
│   ├── probability/        # Win probability engine
│   └── swing/              # Swing calculation & attribution
├── output/                 # Statistics aggregation
│   └── aggregator.go       # Multi-game stat aggregation
└── export/                 # Export to CSV/JSON
```

---

## Adding New Stats

### Step 1: Add the Field to PlayerStats

Edit `model/player_stats.go` to add your new stat:

```go
type PlayerStats struct {
    // ... existing fields ...
    
    // Your new stat
    MyNewStat     int     `json:"my_new_stat"`
    MyNewStatPct  float64 `json:"my_new_stat_pct"`  // If it needs a percentage
}
```

### Step 2: Add to RoundStats (if tracked per-round)

If your stat is tracked per-round, add it to `model/round_stats.go`:

```go
type RoundStats struct {
    // ... existing fields ...
    
    MyNewStatThisRound int
}
```

### Step 3: Track the Stat in Event Handlers

Edit `parser/handlers.go` to track your stat during parsing. Find the appropriate handler:

- **Kill events**: `handleKill()` or create a new `processMyNewStat()` function
- **Damage events**: `handlePlayerHurt()`
- **Round events**: `handleRoundEnd()`
- **Bomb events**: `handleBombPlanted()`, `handleBombDefused()`

Example - tracking a new kill-related stat:

```go
// In handlers.go, add to processKillerStats or create new function
func (d *DemoParser) processMyNewStat(ctx *killContext) {
    if someCondition {
        attacker := d.state.ensurePlayer(ctx.attacker)
        round := d.state.ensureRound(ctx.attacker)
        
        attacker.MyNewStat++
        round.MyNewStatThisRound++
    }
}

// Call it from handleKill()
func (d *DemoParser) handleKill(e events.Kill) {
    // ... existing code ...
    d.processMyNewStat(ctx)
}
```

### Step 4: Calculate Derived Metrics

If your stat needs a per-round rate or percentage, add it to `parser/parser.go` in `computeDerivedStats()`:

```go
func (d *DemoParser) computeDerivedStats() {
    for _, p := range d.state.Players {
        if p.RoundsPlayed > 0 {
            rounds := float64(p.RoundsPlayed)
            // ... existing calculations ...
            
            // Your new derived metric
            p.MyNewStatPct = float64(p.MyNewStat) / rounds
        }
    }
}
```

### Step 5: Add to Aggregator (for cumulative mode)

Edit `output/aggregator.go`:

1. Add field to `AggregatedStats` struct
2. Add accumulation in `AddGame()`:
   ```go
   agg.MyNewStat += p.MyNewStat
   ```
3. Add derived calculation in `Finalize()` if needed

### Step 6: Add to Export

Edit `export/file.go`:

1. Add column to `getSingleGameHeader()`:
   ```go
   return []string{
       // ... existing headers ...
       "My New Stat", "My New Stat Pct",
   }
   ```

2. Add value to `getSingleGameRow()`:
   ```go
   return []string{
       // ... existing values ...
       strconv.Itoa(p.MyNewStat),
       formatFloat(p.MyNewStatPct),
   }
   ```

3. Repeat for `getAggregatedHeader()` and `getAggregatedRow()` if used in cumulative mode.

---

## Rating System

The final rating uses a **probability-based system** that measures how much each player's actions affected their team's win probability.

### Final Rating Formula

The eco-rating is computed in `rating/rating.go`:

```go
rating = 1.0                          // Baseline
       + adrContrib                   // ADR above/below 77
       + kastContrib                  // KAST above/below 72%
       + probSwingContrib             // Probability swing (core metric)
```

### Key Constants (rating/weights.go)

```go
ProbSwingContribMultiplier = 2.5  // How much probability swing affects rating
ADRContribAbove = 0.005           // Bonus per ADR point above 77
ADRContribBelow = 0.004           // Penalty per ADR point below 77
KASTContribAbove = 0.20           // Bonus per KAST % above 72%
KASTContribBelow = 0.25           // Penalty per KAST % below 72%
```

### Probability Swing (Core Metric)

The probability engine (`rating/probability/`) calculates win probability based on:
- Players alive on each team
- Equipment values
- Bomb status
- Time remaining

Each action (kill, death, bomb plant/defuse) creates a swing:
1. **Before action**: Calculate win probability (e.g., 45%)
2. **After action**: Calculate new probability (e.g., 55%)
3. **Swing**: The delta (+10%)

This is accumulated per player and becomes the primary rating driver.

## Key Concepts

### KAST
**K**ill, **A**ssist, **S**urvive, or **T**raded. Percentage of rounds where player contributed.

### Trade
A kill that avenges a teammate's death within 5 seconds.

### Probability Swing  
Win probability delta from player actions. A kill that moves win probability from 30% to 50% = +20% swing.

### Economic Impact
Kill value adjusted for equipment advantage. Killing a rifle player with a pistol is worth 1.8x; killing a pistol player with a rifle is worth 0.7x.

---

## Files Quick Reference

| File | Purpose |
|------|---------|
| `model/player_stats.go` | Add new stat fields |
| `model/round_stats.go` | Add per-round tracking fields |
| `parser/handlers.go` | Track stats during parsing |
| `parser/parser.go` | Calculate derived metrics |
| `output/aggregator.go` | Accumulate stats across games |
| `export/file.go` | Add to CSV export |
| `rating/weights.go` | Constants (mostly legacy round swing) |
| `rating/rating.go` | Final rating formula |
| `rating/economy.go` | Economic kill/death values |

---

## Questions?

Review the inline comments in each file. Comments were generated with AI assistance to help explain the code, though there may be mistakes.
