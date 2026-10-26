# Eco-Rating Calculation Reference

This document describes how **probability swing** is computed during demo parsing and how it feeds **final rating**. The design is HLTV 3.0–style: kills and deaths are captured through win-probability impact, not as separate K/D terms in the rating formula.

Source of truth for implementation:

- `rating/rating.go` — final rating formula
- `rating/weights.go` — baselines and multipliers
- `rating/swing/` — ledger, pool allocator, residual swing, config
- `rating/probability/` — win probability engine and lookup tables
- `engine/eco/collector.go`, `engine/eco/swing_tracker.go` — live event tracking and ledger application; alive counts and round state come from `engine/round`

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

**Map factor:** `map_T_win_rate / map_baseline`, where `map_baseline` (0.489) is the all-maps T win rate the base table already carries (see [Map T-side win rates](#map-t-side-win-rates) below).

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
| `PlayerSwing` | Running per-player totals for the round |
| `TeamPositive` / `TeamNegative` | Team-level accounting |
| `ResidualApplied` | Whether end-of-round residual was applied |
| `DebugWarnings` | Zero-sum validation failures |

Each ledger event records:

- Raw delta and pool sizes
- Positive allocations (killer credit, helpers, objective primary, residual win)
- Negative allocations (victim death, team responsibility, objective loss, residual loss)
- A `SwingReason` tag per allocation (e.g. `kill_final_hit`, `round_residual_loss`)

At round end, `ValidateRoundSwingLedger` checks that `|sum(PlayerSwing)| ≤ ZeroSumTolerance` (0.005).

### Live parsing flow

1. **Event occurs** (kill, plant, defuse, trade reallocation)
2. **`PoolAllocator`** computes pools and player weights
3. **Ledger applies** the allocations
4. **`applyLedger`** in `engine/eco/collector.go` updates `RoundStats.ProbabilitySwing`
5. **Round end:** residual swing applied, ledger validated, stats rolled up

With `SurvivalCreditEnabled` false (the default), survival credit is off and conversion value is handled by end-of-round residual swing.

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
killer_credit = KillFinalHitBase
              + KillDamageShareWeight × (killer_damage / total_damage)

if trade kill: killer_credit ×= TradeKillMultiplier   // default 0.80

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

| Config field | Default | Old value |
|------------|---------|-----------|
| `KillFinalHitBase` | 0.35 | 0.60 |
| `KillDamageShareWeight` | 0.45 | 0.25 |
| `TradeKillMultiplier` | 0.80 | 0.70 penalty subtract |

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

**Disabled by default** (`SurvivalCreditEnabled` false).

When enabled, prior advantage creators receive up to `SurvivalCreditMaxShare` (default 5%) of the kill’s raw delta. Conversion value is otherwise handled by end-of-round residual swing.

### Bomb plant

```
plant_delta = win_prob_after_plant(T) - win_prob_before_plant(T)
positive_pool = negative_pool = |plant_delta|
```

**Positive pool (T side):**

```
planter_share = plant_delta × PlantPlanterShare     // default 45%
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
defuser_share = defuse_delta × DefuseDefuserShare   // default 60%
support_pool  = defuse_delta × (1 - defuser_share)          // default 40%
```

Support contributors scored by retake kills, damage, flash assists, alive-at-defuse. Fallback to 100% defuser if no supporters.

**Negative pool (T side):** allocated by prior negative swing + 0.25 if alive at defuse time.

### Bomb explode

No extra swing attribution; outcome is already reflected in planted-state probabilities.

### Exit frags

After bomb defuse, bomb explode, or team elimination, `RoundDecided` is set. Kills after that point are tagged as exit frags for stats.

Exit frag swing behavior is controlled by `ExitFrags`:

| Value | Behavior |
|-------|----------|
| `zero` (default) | No probability swing |
| `tiny` | Max 5% of normal raw delta |

The elimination kill itself still receives normal swing — `RoundDecided` is set in a separate handler that runs after the main kill handler.

### End-of-round residual swing

When a round ends, remaining uncredited probability is distributed so the outcome is fully represented:

```
residual = min(1.0 - current_winner_win_prob, ResidualMax)   // default cap 0.35
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
      + SavePenaltyWeight if alive at round end   // default 0.35
minimum score = 1.0
```

This credits entry players who force saves and penalizes saving players on lost rounds. Controlled by `ResidualEnabled` (default true).

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

Implemented in `rating.ComputeFinalRating`; Stats stores it as `swing_rating` (the side ratings as the T and CT rows' `swing_rating`):

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

### Optional KPR/DPR modifier (off)

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
| KPR/DPR | No (the modifier is off) |
| `EcoKillValue` / `EcoDeathValue` | No (separate export stats) |
| Multi-kills, clutches | No (multi-kills feed the HLTV ratings) |
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

**DuelSwing** (not in final rating; stored as `ecoDuelSwing`):

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

Measured on 3,131 CSC S19–S20 league (BO1/BO3, playoffs) and Combine maps: 68,872 rounds and 483,374
kills, with `engine/cmd/probtables` (Core-planning#305). Each cell's sample (wins / total) is the
comment beside it in the source. `EnforceMonotonicity` runs after the base table loads.

### Base win probabilities — no bomb planted

T-side win probability by alive counts. Rows = T alive, columns = CT alive.

| T \ CT | 5 | 4 | 3 | 2 | 1 | 0 |
|--------|------|------|------|------|------|------|
| **5** | 0.489 | 0.708 | 0.891 | 0.978 | 0.999 | 0.990 |
| **4** | 0.298 | 0.511 | 0.751 | 0.929 | 0.993 | 0.990 |
| **3** | 0.130 | 0.280 | 0.515 | 0.785 | 0.965 | 0.990 |
| **2** | 0.028 | 0.086 | 0.224 | 0.483 | 0.813 | 0.990 |
| **1** | 0.002 | 0.008 | 0.037 | 0.130 | 0.401 | 0.990 |
| **0** | 0.010 | 0.010 | 0.010 | 0.010 | 0.010 | 0.500 |

Notes:

- `5v0`, `4v0`, etc.: forced — all CTs dead, T wins.
- `0v5`–`0v1`: forced — all Ts dead, CT wins.
- `0v0`: draw state, should not occur.

### Base win probabilities — bomb planted

| T \ CT | 5 | 4 | 3 | 2 | 1 | 0 |
|--------|------|------|------|------|------|------|
| **5** | 0.785 | 0.856 | 0.946 | 0.994 | 0.998 | 0.990 |
| **4** | 0.512 | 0.663 | 0.856 | 0.971 | 0.998 | 0.990 |
| **3** | 0.237 | 0.415 | 0.657 | 0.886 | 0.989 | 0.990 |
| **2** | 0.071 | 0.146 | 0.341 | 0.645 | 0.915 | 0.990 |
| **1** | 0.003 | 0.027 | 0.087 | 0.274 | 0.627 | 0.990 |
| **0** | 0.000 | 0.005 | 0.007 | 0.040 | 0.220 | 1.000 |

Notes:

- `5v0`–`1v0`: forced — all CTs dead with bomb planted.
- `0v5`–`0v1`: measured — the defuse race after the last T dies.
- `0v0_planted`: forced — bomb explodes, T wins.

### Duel win rates

Attacker win probability by equipment category. Rows = attacker, columns = defender. Mirror
matchups are fixed at 0.500.

| Attacker ↓ / Defender → | Starter pistol | Upgraded pistol | SMG | Rifle | AWP |
|-------------------------|---------------:|----------------:|----:|------:|----:|
| **Starter pistol** | 0.500 | 0.523 | 0.265 | 0.242 | 0.258 |
| **Upgraded pistol** | 0.477 | 0.500 | 0.366 | 0.354 | 0.345 |
| **SMG** | 0.735 | 0.634 | 0.500 | 0.427 | 0.399 |
| **Rifle** | 0.758 | 0.646 | 0.573 | 0.500 | 0.462 |
| **AWP** | 0.742 | 0.655 | 0.601 | 0.538 | 0.500 |

These values drive `GetDuelWinRate()` and therefore killer/victim **economy weights** on each kill (weights in the zero-sum allocator, not pool multipliers).

### Map T-side win rates

Round-level T win rate per map. Used as `map_factor = value / map_baseline`; `MapBaseline` = 0.489 (33703 / 68872, all maps).

| Map | T win rate | Sample (T wins / total) | Map factor (÷ 0.489) |
|-----|------------|-------------------------|---------------------|
| de_ancient | 0.489 | 5466 / 11179 | 1.000 |
| de_anubis | 0.526 | 4776 / 9086 | 1.076 |
| de_dust2 | 0.509 | 4895 / 9626 | 1.041 |
| de_inferno | 0.505 | 4838 / 9576 | 1.033 |
| de_mirage | 0.465 | 5268 / 11324 | 0.951 |
| de_nuke | 0.463 | 4451 / 9613 | 0.947 |
| de_overpass | 0.473 | 3597 / 7597 | 0.967 |
| de_train | 0.473 | 412 / 871 | 0.967 |

Maps not in this table default to `MapBaseline` (factor **1.00**).

---

## Config

### Swing settings

`swing.DefaultConfig()` (`rating/swing/config.go`); nothing overrides it.

| Field | Default | Meaning |
|-------|---------|---------|
| `ExitFrags` | `"zero"` | Exit frag swing: `zero` or `tiny` (5%) |
| `ResidualEnabled` | `true` | Apply end-of-round residual swing |
| `ResidualMax` | `0.35` | Cap on residual correction per round |
| `SavePenaltyWeight` | `0.35` | Penalty weight for alive losers at round end |
| `PlantPlanterShare` | `0.45` | Planter’s share of plant positive pool |
| `DefuseDefuserShare` | `0.60` | Defuser’s share of defuse positive pool |
| `KillFinalHitBase` | `0.35` | Base killer credit before damage share |
| `KillDamageShareWeight` | `0.45` | Damage share component of killer credit |
| `TradeKillMultiplier` | `0.80` | Trade kill credit multiplier |
| `SurvivalCreditEnabled` | `false` | Man-advantage survival credit |
| `SurvivalCreditMaxShare` | `0.05` | Max survival credit per event (if enabled) |
| `ZeroSumTolerance` | `0.005` | Ledger validation tolerance |

### Rating modifier

The KPR/DPR modifier (Part 5) is off: `engine/eco` calls `Derive(p, false)`, so the final rating is swing + ADR + KAST.
