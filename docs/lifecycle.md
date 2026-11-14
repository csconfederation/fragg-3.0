# Map lifecycle

How `engine/match` turns facts into a map's status: when it goes live, which rounds commit, how it
ends. Decision numbers (`D12`) are in [decisions.md](decisions.md).

Rules marked **planned** are decided but not yet in the parser; see [Not yet built](#not-yet-built).

## Terms (D60)

| Term | Parser | Stats | Core |
|---|---|---|---|
| **map** | one game on one map, one demo (`engine/match` calls it a match) | `Match` (`{id}`, or `{id}_{k}` in a BO3) | `MatchStats` (one map of a series) |
| **series** | none | none | `Matches` (BO1 or BO3) |
| **ScoreOnly** | an ended map whose rounds don't cover it (was `Partial`) | `fullStatsOK = false` | accepted map; Core can't tell it from a full one |
| none | none | none | `STATS_PARTIAL`: a series with some map rejected. Not ScoreOnly |

## Rules

| # | Rule |
|---|---|
| 1 | **Go-live (D62).** A map goes live at the first round open where the first listed T and the first listed CT member each have 800 money (money + spent). Going live sets both scores to 0. The `Warmup` flag isn't read |
| 2 | **Round open.** A round opens at a frame within 64 ticks of `RoundStart` (an early open) or at `FreezeEnd`. `FreezeEnd` always reopens and replaces an early open. A fact belongs to the open round |
| 3 | **Knife (D58).** A round whose freeze ends with both first members on 0 money is a knife round. The flag goes on the round being opened. A knife round never commits or scores. A knife while live is a restart: the map waits for the next 800/800 open, which resets the scores |
| 4 | **Round commit (D59, planned).** One rule for every round, the deciding round included. A round commits at its closing event if its number agrees at open, `RoundEnd` and close, and its winner is one of its two teams. The closing event is `RoundEndOfficial`; for the final round it is the first of `RoundEndOfficial`, a game end after its `RoundEnd`, or the stream's end. A round with no `RoundEnd`, or one replaced by a new open before it closes, is rejected with its facts |
| 5 | **Restores and replays (D12).** A committing round `n` removes every committed round numbered `n` or higher. A replay that skips a number leaves a gap, and the map is ScoreOnly (D9) |
| 6 | **Score.** A live `ScoreUpdated` sets both teams' scores. A round with none gives its winner +1. Knife rounds don't score |
| 7 | **Team names.** `"[tag] Name"` becomes `Name`. A 31-character clan matches the known team whose name contains it. With no clan names, `StartedT` / `StartedCT`, swapped by half. Core matches these names exactly |
| 8 | **Clinch (MR12).** 13 rounds with the loser under 12; in overtime, the first to win 4 of a 6-round block. `fact.MR = 12` |
| 9 | **End (D56, planned).** The map ends at the clinch (rule 8) or at the game's own end (`GamePhase` Ended or `WinPanelMatch`), whichever comes first. A surrender or admin end is a game end. The game's end can arrive before the final round's `RoundEnd`: that round still ends, scores and closes (rule 4) before the map is Ended. Reason: `clinch`, else `tie` at a level score, else `game_end`. If the two signals disagree, the parser logs it and keeps the first end. Rounds after the end are ignored |
| 10 | **Ties (D63, planned).** A tie is a game end at a level score. It is Ended with reason `tie`. The parser can't tell the match type: a Combine can tie 15-15 after one overtime; a regulation tie means a broken server config |
| 11 | **Members.** At open: the listed players on each side, minus late connects, with a reconnected player's last record. An alive player who joins mid-round becomes a member (D33) |
| 12 | **Outcome.** The table below; the first matching row wins |

## Outcomes (D8, D9, D56, D57)

| # | Outcome | When | `reason` (D57, planned) | Worker | Stats | Core |
|---|---|---|---|---|---|---|
| 1 | error | an invariant broke | none | 500 | 502 | transient: retried, counts toward the global breaker |
| 2 | invalid file | not a CS2 demo | none | 400 | 422 | rejected, as row 4 |
| 3 | error | a decode, failed or stopped stream **before** the end (rule 9) | none | 500 | 502 | as row 1 |
| 4 | NoRounds | no committed non-knife round | `no_rounds` | 422 | 422 | map rejected: BO1 `STATS_FAILED`, BO3 with another map accepted `STATS_PARTIAL` |
| 5 | Unfinished | not ended; the stream ended clean or truncated | `stopped` | 422 | 422 | as row 4 |
| 6 | ScoreOnly | ended, but the rounds don't cover 1..n or the scores don't sum to them | `first_round`, `missing_round`, `score_sum` | 200, score-only | `Match` + zeroed `TeamStats` | accepted, COMPLETED |
| 7 | Ended | ended, and the rounds cover the map | `clinch`, `game_end`, `tie` | 200, full | all rows | accepted; regulation scores, names and map synced nightly |

## Not yet built

Today's parser differs from the rules above in these places; each PR that closes one updates this list.

- **D56:** a game end without a clinch is an error ("the game ended, but not by our match-end rule", a 500), except the tie fallback, which commits the last round at a level score. A decode failure after a game end with no clinch is a 500.
- **D59:** the deciding round commits at its `RoundEnd`, except a bomb-ended one, which waits for the bomb's facts (the D15/D27 hold).
- **D57:** the payload has no `outcome` or `reason`; a 422 body is a string.

The status fixtures in `engine/roundlog/testdata/status` pin today's behavior for each.
