# Eco-Rating Calculation Reference

This document describes how **probability swing** is computed during demo parsing and how it feeds **final rating**. The design is HLTV 3.0–style: kills and deaths are captured through win-probability impact, not as separate K/D terms in the rating formula.

Source of truth for implementation:

- `rating/rating.go` — final rating formula
- `rating/weights.go` — baselines and multipliers
- `rating/swing/` — ledger, pool allocator, residual swing, config
- `rating/probability/` — win probability engine and lookup tables
- `parser/handlers.go`, `parser/swing_tracker.go` — live event tracking and ledger application

---

## End-to-end pipeline

```
Round start → build RoundState (alive counts, economy, map)
           → create RoundSwingLedger
           → events (kills, plants, defuses, trades)
           → compute win-prob before/after each event
           → split delta into equal positive + negative pools
           → allocate pools to players via normalized weights
           → record allocations in ledger
           → at round end: apply residual swing + validate zero-sum
           → apply ledger allocations to RoundStats.ProbabilitySwing
           → roll up to match totals
           → after all rounds: compute FinalRating
```

1. Each round, a **probability engine** tracks game state and estimates T-side win probability.
2. Each meaningful event produces a signed **raw delta** = change in win probability.
3. That delta becomes two equal **pools** (positive for the benefiting side, negative for the hurt side).
4. Pools are **distributed** to players using normalized weights (economy affects weights, not pool size).
5. Every allocation is recorded in a per-round **ledger** before updating player stats.
6. At round end, **residual swing** credits uncaptured conversion value (e.g. entry → save → win).
7. Per-round swings are summed into `player.ProbabilitySwing`.
8. **Final rating** combines average swing per round with ADR, KAST, and optionally KPR/DPR.

The final rating formula is unchanged. Only the swing attribution model changed.

---

## Part 1: Win probability engine

Before any swing can be calculated, the engine answers: *“What is this team’s chance to win right now?”*

### Round state inputs

| Input | Source | Effect |
|-------|--------|--------|
| `TAlive`, `CTAlive` | Players alive at round start / after each death | Primary lookup key (e.g. `5v4_none`) |
| `BombPlanted` | Bomb plant event | Switches to `*_planted` table; time adjustments apply |
| `BombDefused` | Defuse event | Round effectively over for CT |
| `TEconomy`, `CTEconomy` | Average team equipment value at round start | Tier difference adjusts probability |
| `Map` | Demo map name | Map-specific T-side bias |
| `TimeRemaining` | Only when bomb is planted | Lower time → more T advantage |

Equipment is categorized into tiers:

| Category | Equipment value |
|----------|-----------------|
| Starter pistol | $0–999 |
| Upgraded pistol | $1000–1999 |
| SMG | $2000–3499 |
| Rifle | $3500–4749 |
| AWP | $4750+ |

### Win probability formula

For a given side:

```
T_win_prob = base_table[T_alive v CT_alive, bomb_status]
           × economy_multiplier(T_econ - CT_econ)
           × map_factor
           × time_multiplier   (only if bomb planted)

side_win_prob = T_win_prob           (if T)
              = 1 - T_win_prob       (if CT)

clamped to [0.01, 0.99]
```

**Economy multiplier** (T econ tier minus CT econ tier):

| Diff | Multiplier |
|------|------------|
| +4 | 1.20 |
| +3 | 1.15 |
| +2 | 1.10 |
| +1 | 1.05 |
| 0 | 1.00 |
| −1 | 0.95 |
| −2 | 0.90 |
| −3 | 0.85 |
| −4 | 0.80 |

**Map factor:** `map_T_win_rate / 0.50` (see [Map T-side win rates](#map-t-side-win-rates) below).

**Time multiplier** (bomb planted only):

| Time remaining | Multiplier |
|----------------|------------|
| ≤ 5s | ×1.15 |
| ≤ 10s | ×1.08 |
| ≤ 20s | ×1.03 |
| > 20s | ×1.00 |

If a state key is missing from the lookup tables, the engine falls back to a ratio-based estimate (see `rating/probability/tables.go`).

---

## Part 2: Round swing ledger and zero-sum accounting

### Core principle

Instead of creating independent positive and negative swing values, each event produces **one probability delta** that is split into two equal pools:

```
raw_delta = win_prob_after - win_prob_before
pool = |raw_delta|

positive_pool = pool   → distributed to benefiting side
negative_pool = pool   → distributed to hurt side

sum(all player swing in round) ≈ 0
```

Economy, damage share, and trade status affect **allocation weights**, not the total pool size. This prevents swing mass from being created or destroyed across a round.

### Ledger data model (`rating/swing/ledger.go`)

Each round maintains a `RoundSwingLedger`:

| Field | Purpose |
|-------|---------|
| `Events[]` | Every swing allocation before it hits player stats |
| `PlayerSwing` | Running per-player totals for the round |
| `PositiveTotal` / `NegativeTotal` | Team-level accounting |
| `ResidualApplied` | Whether end-of-round residual was applied |
| `ExitFragsIgnored` | Count of exit frags skipped |
| `DebugWarnings` | Zero-sum validation failures |

Each ledger event records:

- Raw delta and pool sizes
- Positive allocations (killer credit, helpers, objective primary, residual win)
- Negative allocations (victim death, team responsibility, objective loss, residual loss)
- A `SwingReason` tag per allocation (e.g. `kill_final_hit`, `round_residual_loss`)

At round end, `ValidateRoundSwingLedger` checks that `|sum(PlayerSwing)| ≤ swing_zero_sum_tolerance` (default 0.005).

### Live parsing flow

1. **Event occurs** (kill, plant, defuse, trade reallocation)
2. **`PoolAllocator`** computes pools and player weights
3. **Ledger records** the event
4. **`applyLedgerAllocations`** in `parser/handlers.go` updates `RoundStats.ProbabilitySwing`
5. **Round end:** residual swing applied, ledger validated, stats rolled up

When `swing_survival_credit_enabled: false`, survival credit is off and conversion value is handled by end-of-round residual swing.

---

## Part 3: Probability swing per event

**Swing** = change in the acting side’s win probability caused by an event.

During live demo parsing, swings flow through `SwingTracker.RecordKill` / `RecordBombPlant` / `RecordBombDefuse` → ledger → `applyLedgerAllocations`.

### Kills — raw delta and pools

```
raw_delta = win_prob_after_kill(killer's side) - win_prob_before_kill
if raw_delta < 0: raw_delta = 0   // kills never reduce killer-side probability

pool = |raw_delta|
positive_pool = pool
negative_pool = pool
```

The state update is: one player dies on the victim’s side (`TAlive` or `CTAlive` decrements).

### Kills — positive pool (benefiting side)

Duel win rate comes from equipment matchup tables (see [Duel win rates](#duel-win-rates) below).

**Killer weight** (economy as weight, not pool multiplier):

```
killer_credit = swing_kill_final_hit_base
              + swing_kill_damage_share_weight × (killer_damage / total_damage)

if trade kill: killer_credit ×= swing_trade_kill_multiplier   // default 0.80

killer_weight = killer_credit × killer_eco_weight
killer_eco_weight = 0.50 / duel_win_rate   (capped at 2.0)
```

**Helper weights** from the remaining shareable fraction `(1 - killer_credit)`:

```
damage_helper_weight = shareable × 0.75 × (contributor_damage / total_damage)
flash_helper_weight  = shareable × 0.25 × min(flash_duration / 3.0, 1.0)
```

All positive-side weights (killer + helpers) are **normalized** to sum to `positive_pool`. The pool is never inflated by economy.

Default attribution constants:

| Config key | Default | Old value |
|------------|---------|-----------|
| `swing_kill_final_hit_base` | 0.35 | 0.60 |
| `swing_kill_damage_share_weight` | 0.45 | 0.25 |
| `swing_trade_kill_multiplier` | 0.80 | 0.70 penalty subtract |

### Kills — negative pool (hurt side)

**Victim weight:**

```
victim_eco_weight = (1 - duel_win_rate) / 0.50   (capped at 2.0)
victim_weight = 1.0 × victim_eco_weight
```

**Low-health redistribution** (does not delete negative swing):

```
victim_hp = 100 - prior_damage_from_others
health_factor = 0.50 + 0.50 × (victim_hp / 100)
  100 HP → full weight
   50 HP → 75% weight
    1 HP → 50% weight

victim_weight ×= health_factor
```

The removed victim share is **redistributed** to alive teammates on the losing side (each gets a base weight of 0.25). Weights are normalized over the full losing side so `negative_pool` is fully allocated.

### Kills — trade death reallocation

When a teammate trades within 5 seconds / 320 ticks, the original death penalty is **redistributed**, not refunded as new positive mass:

```
refund = |original_death_penalty| × 0.30
victim gets +refund
remaining teammates split -refund equally
net swing change = 0
```

Trade death reallocation runs through `SwingTracker.ApplyTradeReallocation` and the ledger.

### Kills — man-advantage survival credit

**Disabled by default** (`swing_survival_credit_enabled: false`).

When enabled, prior advantage creators receive up to `swing_survival_credit_max_share` (default 5%) of the kill’s raw delta. Conversion value is otherwise handled by end-of-round residual swing.

### Bomb plant

```
plant_delta = win_prob_after_plant(T) - win_prob_before_plant(T)
positive_pool = negative_pool = |plant_delta|
```

**Positive pool (T side):**

```
planter_share = plant_delta × swing_plant_planter_share     // default 45%
support_pool  = plant_delta × (1 - planter_share)           // default 55%
```

Support contributors scored by prior kills, damage, flash assists, and alive-at-plant. If no supporters qualify, planter receives 100%.

**Negative pool (CT side):** allocated by prior negative swing + 0.25 if alive at plant time.

### Bomb defuse

```
defuse_delta = win_prob_after_defuse(CT) - win_prob_before_defuse(CT)
positive_pool = negative_pool = |defuse_delta|
```

**Positive pool (CT side):**

```
defuser_share = defuse_delta × swing_defuse_defuser_share   // default 60%
support_pool  = defuse_delta × (1 - defuser_share)          // default 40%
```

Support contributors scored by retake kills, damage, flash assists, alive-at-defuse. Fallback to 100% defuser if no supporters.

**Negative pool (T side):** allocated by prior negative swing + 0.25 if alive at defuse time.

### Bomb explode

No extra swing attribution; outcome is already reflected in planted-state probabilities.

### Exit frags

After bomb defuse, bomb explode, or team elimination, `RoundDecided` is set. Kills after that point are tagged as exit frags for stats.

Exit frag swing behavior is controlled by `swing_exit_frags`:

| Value | Behavior |
|-------|----------|
| `zero` (default) | No probability swing |
| `tiny` | Max 5% of normal raw delta |

The elimination kill itself still receives normal swing — `RoundDecided` is set in a separate handler that runs after the main kill handler.

### End-of-round residual swing

When a round ends, remaining uncredited probability is distributed so the outcome is fully represented:

```
residual = min(1.0 - current_winner_win_prob, swing_residual_max)   // default cap 0.35
```

**Winning side** (positive residual), weighted by:

```
score = round_positive_swing × 1.0
      + (round_damage / max_round_damage) × 0.25
      + flash_assists × 0.15
      + bomb_plant × 0.25
```

If all scores are zero, split evenly among alive winners.

**Losing side** (negative residual), weighted by:

```
score = |round_negative_swing| × 1.0
      + swing_save_penalty_weight if alive at round end   // default 0.35
minimum score = 1.0
```

This credits entry players who force saves and penalizes saving players on lost rounds. Controlled by `swing_residual_enabled` (default true).

### What does NOT add swing

- Exit frags (default: zero swing)
- Round saves alone (no penalty unless residual swing applies)
- Assists (unless via damage/flash contribution sharing on a kill)
- `EcoKillValue` / `EcoDeathValue`: tracked separately for export; economy shapes allocation weights on kills

---

## Part 4: Accumulation to player totals

Each round, per-player `RoundStats.ProbabilitySwing` accumulates via ledger allocations:

| Source | Sign | SwingReason |
|--------|------|-------------|
| Killer final-hit credit | + | `kill_final_hit` |
| Damage contributor share | + | `kill_damage_share` |
| Flash assist share | + | `flash_assist` |
| Victim death penalty | − | `victim_death` |
| Teammate responsibility share | − | redistributed death/trade |
| Bomb plant (planter) | + | `bomb_plant` |
| Bomb plant (support) | + | `plant_support` |
| Bomb defuse (defuser) | + | `bomb_defuse` |
| Bomb defuse (support) | + | `defuse_support` |
| Round-end residual (win) | + | `round_residual_win` |
| Round-end residual (loss) | − | `round_residual_loss` |
| Trade reallocation | ± | `trade_refund` |
| Survival credit (if enabled) | + | `survival_credit` |
| Exit frag (ignored) | 0 | `exit_frag_ignored` |

Debug breakdown fields on `RoundStats` (not used in final rating):

- `KillSwing`, `DeathSwing`, `AssistSwing`, `ObjectiveSwing`, `ResidualSwing`, `ExitFragSwingIgnored`

At round end:

```
player.ProbabilitySwing += round.ProbabilitySwing
player.TProbabilitySwing  or  CTProbabilitySwing  (by side played that round)
```

After all rounds:

```
ProbabilitySwingPerRound = ProbabilitySwing / RoundsPlayed
```

---

## Part 5: Final rating formula

Implemented in `rating.ComputeFinalRating`:

```
adr  = total_damage / rounds_played
kast = kast_rounds / rounds_played    // fraction 0–1, baseline 72%

prob_swing_contrib = ProbabilitySwingPerRound × 2.5

adr_contrib  = (adr - 75) × 0.01   if adr ≥ 75
             = (adr - 75) × 0.012  if adr < 75

kast_contrib = (kast - 0.72) × 0.30   if kast ≥ 0.72
             = (kast - 0.72) × 0.40   if kast < 0.72

final_rating = 1.0 + adr_contrib + kast_contrib + prob_swing_contrib + kpr_dpr_adj

final_rating clamped to [0.20, 3.00]
```

### Rating constants (`rating/weights.go`)

| Constant | Value | Meaning |
|----------|-------|---------|
| `RatingBaseline` | 1.0 | Starting point |
| `BaselineADR` | 75.0 | Average ADR |
| `ADRContribAbove` | 0.01 | Per ADR point above baseline |
| `ADRContribBelow` | 0.012 | Per ADR point below baseline |
| `BaselineKAST` | 0.72 | Average KAST (72%) |
| `KASTContribAbove` | 0.30 | Per KAST point above baseline |
| `KASTContribBelow` | 0.40 | Per KAST point below baseline |
| `ProbSwingContribMultiplier` | 2.5 | Swing per round → rating |
| `MinRating` | 0.20 | Floor |
| `MaxRating` | 3.00 | Ceiling |
| `BaselineKPR` | 0.72 | KPR baseline (optional modifier) |
| `BaselineDPR` | 0.68 | DPR baseline (optional modifier) |

### Optional KPR/DPR modifier (`kdpr_modifier` in config)

When enabled:

```
kpr = kills / rounds
dpr = deaths / rounds

kpr_adj = exp_adjust(kpr - 0.72, max=0.1, k=5)
dpr_adj = exp_adjust(0.68 - dpr, max=0.1, k=5)   // lower DPR is good

kpr_dpr_adj = kpr_adj + dpr_adj    // range roughly -0.20 to +0.20

where exp_adjust(diff, max, k) = sign(diff) × max × (1 - e^(-k × |diff|))
```

KPR/DPR is a small secondary nudge. Kills/deaths are **not** directly in the rating beyond swing and this optional modifier.

### What is NOT in final rating

| Metric | Used in FinalRating? |
|--------|---------------------|
| Probability swing | Yes (primary driver) |
| ADR | Yes |
| KAST | Yes |
| KPR/DPR | Only if `kdpr_modifier` enabled |
| `SwingRating` | No (export/display only) |
| `EcoKillValue` / `EcoDeathValue` | No (separate export stats) |
| Multi-kills, clutches | No (passed to side rating but unused there) |
| HLTV rating | No (parallel comparison stat) |

### Side ratings (`TEcoRating`, `CTEcoRating`)

Same formula as final rating, but scoped to T or CT rounds:

```
adr, kast, prob_swing_per_round, kpr/dpr computed from side-specific totals
```

---

## Part 6: Worked example

**Setup:** 30 rounds, `ProbabilitySwing = +0.90` total → `+0.03` per round.

**One kill example:** 4v4, no bomb, T kills CT. Killer dealt 10 damage, teammate dealt 90 damage (finishing tap).

- Win prob before (T): 47.3%
- Win prob after (4v3): 54.5%
- `raw_delta = +0.072`
- `pool = 0.072` (same for positive and negative sides)

Killer had rifle vs pistol (25% duel win rate):

```
killer_eco_weight = 0.50 / 0.25 = 2.0
victim_eco_weight = 0.75 / 0.50 = 1.5

killer_credit = 0.35 + 0.45 × 0.10 = 0.395
killer_weight = 0.395 × 2.0 = 0.79

helper_weight (90 dmg) ≈ 0.605 × 0.75 × 0.90 = 0.409
(normalized over positive side → killer ~39%, helper ~61% of pool)

killer_swing  ≈ 0.072 × 0.39 = +0.028
helper_swing  ≈ 0.072 × 0.61 = +0.044
```

Victim side (full HP, no teammates to redistribute to):

```
victim_contribution ≈ -0.072   (full negative pool to victim)
```

Round is zero-sum: `+0.028 + 0.044 - 0.072 ≈ 0`

**Save scenario:** T entries twice, CTs save, T wins. Before round end T win prob = 82%.

```
residual = min(1.0 - 0.82, 0.35) = 0.18
T contributors split +0.18 by contribution score
CT saving players split -0.18 by responsibility score
```

**Rating impact of that player’s overall swing:**

```
prob_swing_contrib = 0.03 × 2.5 = +0.075 to rating
```

If ADR = 80 and KAST = 75%:

```
adr_contrib  = (80 - 75) × 0.01  = +0.05
kast_contrib = (0.75 - 0.72) × 0.30 = +0.009
final_rating ≈ 1.0 + 0.05 + 0.009 + 0.075 = 1.134
(+ small KPR/DPR adj if enabled)
```

---

## Part 7: Derived metrics (informational)

**SwingRating** (stored on player, not used in final rating):

```
SwingRating = 1.0 + (ProbabilitySwingPerRound × 10.0)
clamped to [0.5, 1.5]

Interpretation:
  0% avg swing/round  → 1.00
  +4% avg swing/round → 1.40
  -3% avg swing/round → 0.70
```

**DuelSwing** (also not in final rating):

```
DuelSwing = EcoKillValue - EcoDeathValue   // legacy-style eco point totals
```

---

## Part 8: Summary — what affects final rating

| Factor | How it enters |
|--------|---------------|
| Kill impact | Zero-sum pool split by damage share, final-hit credit, eco weights |
| Death impact | Negative pool share with health redistribution to teammates |
| Damage without kill | Via contributor share on someone else’s kill swing |
| Flash assists | Via flash share on kill swing (25% of shareable pool) |
| Bomb plant/defuse | Shared probability delta (planter/defuser + support, opposing side penalized) |
| Round conversion | End-of-round residual swing (entries forcing saves, save penalties) |
| Exit frags | Zero swing by default |
| Man-advantage survival | Disabled by default; use residual instead |
| ADR | Linear adjustment vs baseline 75 |
| KAST | Asymmetric linear adjustment vs baseline 72% |
| KPR/DPR | Optional exponential nudge (config) |
| Equipment tier at round start | Shapes win-prob lookups for all events |
| Map | Scales all win-prob calculations |
| Bomb timer | Scales win-prob when bomb is planted |

### Expected rating shifts vs old model

| Player type | Typical swing change |
|-------------|---------------------|
| Exit fraggers | Down |
| Save-heavy players on lost rounds | Down (residual penalty) |
| Low-damage kill finishers | Down |
| Planters/defusers with little else | Down (shared credit) |
| Entry players forcing saves | Up (residual credit) |
| Heavy damage before teammate finish | Up |
| Retake contributors enabling defuse | Up |

---

## Probability tables (`rating/probability/tables_data.go`)

Empirically derived from **54,584 rounds** and **390,522 kills** across competitive CS2 demos.

### Base win probabilities — no bomb planted

T-side win probability by alive counts. Rows = T alive, columns = CT alive.

| T \ CT | 5 | 4 | 3 | 2 | 1 | 0 |
|--------|------|------|------|------|------|------|
| **5** | 0.494 | 0.525 | 0.715 | 0.902 | 0.989 | 0.990 |
| **4** | 0.475 | 0.473 | 0.545 | 0.752 | 0.951 | 0.990 |
| **3** | 0.305 | 0.393 | 0.429 | 0.554 | 0.829 | 0.990 |
| **2** | 0.124 | 0.231 | 0.299 | 0.337 | 0.511 | 0.990 |
| **1** | 0.018 | 0.100 | 0.299 | 0.558 | 0.462 | 0.990 |
| **0** | 0.010 | 0.010 | 0.010 | 0.010 | 0.010 | 0.500 |

Notes:

- `5v0`, `4v0`, etc.: forced — all CTs dead, T wins.
- `0v5`–`0v1`: forced — all Ts dead, CT wins.
- `0v0`: draw state, should not occur.

Sample sizes (wins / total from source comments):

| Key | Wins / Total |
|-----|--------------|
| 5v5_none | 39679 / 80398 |
| 5v4_none | 17158 / 32701 |
| 5v3_none | 11957 / 16730 |
| 5v2_none | 6695 / 7425 |
| 5v1_none | 2522 / 2549 |
| 4v5_none | 16027 / 33736 |
| 4v4_none | 9036 / 19099 |
| 4v3_none | 7495 / 13763 |
| 4v2_none | 5951 / 7910 |
| 4v1_none | 3001 / 3154 |
| 3v5_none | 6235 / 20406 |
| 3v4_none | 6048 / 15379 |
| 3v3_none | 5565 / 12978 |
| 3v2_none | 4720 / 8518 |
| 3v1_none | 3165 / 3816 |
| 2v5_none | 1408 / 11385 |
| 2v4_none | 2587 / 11180 |
| 2v3_none | 3190 / 10687 |
| 2v2_none | 2627 / 7786 |
| 2v1_none | 1886 / 3689 |
| 1v5_none | 93 / 5257 |
| 1v4_none | 415 / 4162 |
| 1v3_none | 978 / 3271 |
| 1v2_none | 1229 / 2203 |
| 1v1_none | 666 / 1442 |

### Base win probabilities — bomb planted

| T \ CT | 5 | 4 | 3 | 2 | 1 | 0 |
|--------|------|------|------|------|------|------|
| **5** | 0.794 | 0.751 | 0.825 | 0.954 | 0.995 | 0.990 |
| **4** | 0.742 | 0.733 | 0.741 | 0.860 | 0.970 | 0.990 |
| **3** | 0.543 | 0.669 | 0.683 | 0.729 | 0.860 | 0.990 |
| **2** | 0.241 | 0.467 | 0.650 | 0.662 | 0.499 | 0.990 |
| **1** | 0.052 | 0.162 | 0.500 | 0.932 | 0.549 | 0.990 |
| **0** | 0.000 | 0.004 | 0.005 | 0.020 | 0.118 | 1.000 |

Notes:

- `5v0`–`1v0`: forced — all CTs dead with bomb planted.
- `0v0_planted`: bomb explodes, T wins (249 / 249).

Sample sizes (wins / total from source comments):

| Key | Wins / Total |
|-----|--------------|
| 5v5_planted | 910 / 1146 |
| 5v4_planted | 1508 / 2007 |
| 5v3_planted | 2779 / 3369 |
| 5v2_planted | 3904 / 4091 |
| 5v1_planted | 2963 / 2978 |
| 4v5_planted | 991 / 1335 |
| 4v4_planted | 2061 / 2813 |
| 4v3_planted | 3585 / 4841 |
| 4v2_planted | 5538 / 6437 |
| 4v1_planted | 4096 / 4224 |
| 4v0_planted | 11 / 11 |
| 3v5_planted | 649 / 1195 |
| 3v4_planted | 2130 / 3184 |
| 3v3_planted | 4104 / 6011 |
| 3v2_planted | 6038 / 8284 |
| 3v1_planted | 4717 / 5486 |
| 3v0_planted | 17 / 17 |
| 2v5_planted | 201 / 835 |
| 2v4_planted | 1173 / 2510 |
| 2v3_planted | 3520 / 5418 |
| 2v2_planted | 5316 / 8034 |
| 2v1_planted | 2328 / 4663 |
| 2v0_planted | 27 / 27 |
| 1v5_planted | 23 / 441 |
| 1v4_planted | 214 / 1321 |
| 1v3_planted | 1395 / 2789 |
| 1v2_planted | 3787 / 4063 |
| 1v1_planted | 825 / 1503 |
| 1v0_planted | 52 / 52 |
| 0v5_planted | 0 / 99 |
| 0v4_planted | 2 / 517 |
| 0v3_planted | 8 / 1478 |
| 0v2_planted | 56 / 2746 |
| 0v1_planted | 309 / 2620 |

### Duel win rates

Attacker win probability by equipment category. Rows = attacker, columns = defender.

| Attacker ↓ / Defender → | Starter pistol | Upgraded pistol | SMG | Rifle | AWP |
|-------------------------|---------------:|----------------:|----:|------:|----:|
| **Starter pistol** | 0.500 | 0.520 | 0.257 | 0.248 | 0.270 |
| **Upgraded pistol** | 0.480 | 0.500 | 0.348 | 0.360 | 0.349 |
| **SMG** | 0.743 | 0.652 | 0.500 | 0.431 | 0.401 |
| **Rifle** | 0.752 | 0.640 | 0.569 | 0.500 | 0.468 |
| **AWP** | 0.730 | 0.651 | 0.599 | 0.532 | 0.500 |

Full key → value mapping (as stored in code):

| Key | Win rate | Sample (wins / total) |
|-----|----------|------------------------|
| starter_pistol_vs_starter_pistol | 0.500 | 30039 / 30039 |
| starter_pistol_vs_upgraded_pistol | 0.520 | 3335 / 6415 |
| starter_pistol_vs_smg | 0.257 | 2338 / 9109 |
| starter_pistol_vs_rifle | 0.248 | 3748 / 15102 |
| starter_pistol_vs_awp | 0.270 | 6037 / 22368 |
| upgraded_pistol_vs_starter_pistol | 0.480 | 3080 / 6415 |
| upgraded_pistol_vs_upgraded_pistol | 0.500 | 334 / 334 |
| upgraded_pistol_vs_smg | 0.348 | 872 / 2508 |
| upgraded_pistol_vs_rifle | 0.360 | 2532 / 7040 |
| upgraded_pistol_vs_awp | 0.349 | 7108 / 20393 |
| smg_vs_starter_pistol | 0.743 | 6771 / 9109 |
| smg_vs_upgraded_pistol | 0.652 | 1636 / 2508 |
| smg_vs_smg | 0.500 | 4738 / 4738 |
| smg_vs_rifle | 0.431 | 8134 / 18879 |
| smg_vs_awp | 0.401 | 14262 / 35511 |
| rifle_vs_starter_pistol | 0.752 | 11354 / 15102 |
| rifle_vs_upgraded_pistol | 0.640 | 4508 / 7040 |
| rifle_vs_smg | 0.569 | 10745 / 18879 |
| rifle_vs_rifle | 0.500 | 24356 / 24356 |
| rifle_vs_awp | 0.468 | 48771 / 104315 |
| awp_vs_starter_pistol | 0.730 | 16331 / 22368 |
| awp_vs_upgraded_pistol | 0.651 | 13285 / 20393 |
| awp_vs_smg | 0.599 | 21249 / 35511 |
| awp_vs_rifle | 0.532 | 55544 / 104315 |
| awp_vs_awp | 0.500 | 89415 / 89415 |

These values drive `GetDuelWinRate()` and therefore killer/victim **economy weights** on each kill (weights in the zero-sum allocator, not pool multipliers).

### Map T-side win rates

Empirically derived round-level T win rate per map. Used as `map_factor = value / 0.50`.

| Map | T win rate | Sample (T wins / total) | Map factor (÷ 0.50) |
|-----|------------|-------------------------|---------------------|
| de_ancient | 0.513 | 4120 / 8027 | 1.026 |
| de_anubis | 0.564 | 2623 / 4652 | 1.128 |
| de_dust2 | 0.519 | 3822 / 7366 | 1.038 |
| de_inferno | 0.512 | 3848 / 7517 | 1.024 |
| de_mirage | 0.498 | 3711 / 7457 | 0.996 |
| de_nuke | 0.480 | 3832 / 7984 | 0.960 |
| de_overpass | 0.488 | 2665 / 5457 | 0.976 |
| de_train | 0.456 | 2793 / 6124 | 0.912 |

Maps not in this table default to **0.500** T win rate (factor **1.00**).

### Swing normalization constants

From the bottom of `tables_data.go` — used for `SwingRating` display, not final rating:

| Constant | Value | Meaning |
|----------|-------|---------|
| `SwingToRatingScale` | 10.0 | Multiplier on avg swing per round |
| `SwingRatingBaseline` | 1.0 | Zero swing → 1.0 |
| `MinSwingRating` | 0.40 | Floor (integration helper) |
| `MaxSwingRating` | 1.80 | Ceiling (integration helper) |

Parser clamp for exported `SwingRating`: **[0.5, 1.5]**.

---

## Config

### Swing parser settings (`config.json`)

All swing settings have defaults in `config.DefaultConfig()` / `rating/swing.DefaultConfig()`.

| Key | Default | Meaning |
|-----|---------|---------|
| `swing_exit_frags` | `"zero"` | Exit frag swing: `zero` or `tiny` (5%) |
| `swing_residual_enabled` | `true` | Apply end-of-round residual swing |
| `swing_residual_max` | `0.35` | Cap on residual correction per round |
| `swing_save_penalty_weight` | `0.35` | Penalty weight for alive losers at round end |
| `swing_plant_planter_share` | `0.45` | Planter’s share of plant positive pool |
| `swing_defuse_defuser_share` | `0.60` | Defuser’s share of defuse positive pool |
| `swing_kill_final_hit_base` | `0.35` | Base killer credit before damage share |
| `swing_kill_damage_share_weight` | `0.45` | Damage share component of killer credit |
| `swing_trade_kill_multiplier` | `0.80` | Trade kill credit multiplier |
| `swing_survival_credit_enabled` | `false` | Man-advantage survival credit |
| `swing_survival_credit_max_share` | `0.05` | Max survival credit per event (if enabled) |
| `swing_zero_sum_tolerance` | `0.005` | Ledger validation tolerance |

Example block:

```json
{
  "swing_exit_frags": "zero",
  "swing_residual_enabled": true,
  "swing_residual_max": 0.35,
  "swing_save_penalty_weight": 0.35,
  "swing_plant_planter_share": 0.45,
  "swing_defuse_defuser_share": 0.60,
  "swing_kill_final_hit_base": 0.35,
  "swing_kill_damage_share_weight": 0.45,
  "swing_trade_kill_multiplier": 0.80,
  "swing_survival_credit_enabled": false,
  "swing_zero_sum_tolerance": 0.005
}
```

### Rating modifier

In `config.json`, `"kdpr_modifier": true` adds the KPR/DPR adjustment described in Part 5. With it off, final rating is purely swing + ADR + KAST.
