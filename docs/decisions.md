# Decisions

The numbers cited in code comments (`D12`, `D33`, ...). A PR that implements a decision updates its
row. Status: **done** (in the parser), **planned** (decided, not yet built), **held** or
**rejected** (the old rule stays), **process** (no code).

| # | Rule | Status | Why |
|---|---|---|---|
| D1 | Damage is the game's `HealthDamageTaken` | done | matches the game's own damage; ends the old over-count of fatal shotgun hits |
| D2 | Ship the rewrite once validated, outside the season; superseded by D64 | process | one parser per season |
| D3 | The old parser port is an oracle only; production goes old to rewrite in one cutover | process | one validated delivery |
| D4 | The worker stays a separate repo and deploy unit | process | the worker is the rollback unit |
| D5 | Delete the eco-rating CLI in the final PR | planned | no caller |
| D6 | No version metadata in the payload; reversed by D50 | superseded | no consumer then |
| D7 | Clean payload: every stat Stats stores and nothing else | done | remove structural cruft |
| D8 | A demo that stops before the match ends is Unfinished (422) | done | incomplete maps must not reach player stats |
| D9 | A map whose rounds don't cover it, but that ended, is ScoreOnly: teams and score, no player or round rows | done | the score is right, player stats would be partial |
| D10 | Bomb damage in a half's final round counts when the bomb ends the round | done | it was lost before |
| D11 | The file name's match ID wins over the header's; log both | done | headers carry wrong IDs |
| D12 | A replayed round supersedes the rounds from its number up; a skipped number leaves a gap (ScoreOnly) | done | in-demo restores happen |
| D13 | The pistol rating uses rounds 1 and 13, no overtime | done | the old open counter hit other rounds |
| D14 | The bomb's kill counts with its damage (D10) | done | same reason as D10 |
| D15 | The deciding round waits for the bomb's facts before it commits | replaced by D59 | CS2 sends the explosion's damage after `RoundEnd` |
| D16 | Keep and compute every stat field, even unread ones; validate them | process | a later view may use them |
| D17 | `ecoStatsOK` leaves the payload; Stats writes `true` for engine imports | done | marks maps without eco stats |
| D18 | A score-only import never replaces a match that has player rows | process | the richer record wins |
| D19 | The header match ID is compared in Stats, from a top-level payload key | done | the worker never sees the file name |
| D20 | `atd` divides by deaths, 0 with none | done | survived rounds no longer count 0 |
| D21 | The two buy-gap kill stats keep their meaning and are renamed | done | the old names read backwards |
| D22 | One trade detector with a 5 s window | done | the 4 s and 5 s windows disagreed |
| D23 | A clutch entered by a team kill counts | done | matches the other pipeline |
| D24 | Keep working-state fields stored and document them as such | process | they aren't stats |
| D25 | One definition per shared concept | done | no double tracking of trades or deaths |
| D26 | The `eco` prefix stays only on swing and economy-weighted stats | done | other stats aren't eco |
| D27 | The round end waits for the bomb's facts in explosion-ended rounds; replaced by D59 | done | CS2 sends the bomb's deaths after `RoundEnd` |
| D28 | Rebuild the stat layer: pure calculators over committed rounds, one row model, one fold | done | the engine ported two old pipelines |
| D29 | Match the old parser unless its logic was wrong; timing and ordering changes are acceptable | process | parity by default |
| D30 | Scope growth (renames, new columns, approved fixes) in one release is accepted | process | each change was decided alone |
| D31 | Every stat counts the same kills and damage | done | one source |
| D32 | A clutch needs an enemy faced | done | the last player never saw one otherwise |
| D33 | A mid-round joiner counts as alive | done | they are in the round |
| D34 | No flash assist on your own kill; one assist counts once in `eac` | done | double count |
| D35 | A flash assist earns KAST | done | it is an assist |
| D36 | Assist only when an enemy made the kill | rejected | today's rule is public |
| D37 | 4v5/5v4 starts from the first enemy kill | rejected | today's rule is public |
| D38 | Lurker is the T farthest from the team; one close moment doesn't disqualify | done | see the lurker rule |
| D39 | Team pistol columns use rounds 1 and 13 | done | matches D13 |
| D40 | Bots stay out of match averages | done | no bot rows in the archive |
| D41 | Trade denial counts at window expiry | done | the window ended |
| D42 | A same-tick trade is a fast trade | done | it is the fastest |
| D43 | The HLTV rating uses exact rounds survived | done | no approximation |
| D44 | An alive disconnect isn't a survival | done | the player left |
| D45 | Swing-loss sharing ignores dead players with no swing | held | parity until decided |
| D46 | A planter or defuser keeps the 30 win-share bonus with no team damage | done | the bonus is for the objective |
| D47 | Average time to death keeps its fraction | done | Stats takes a decimal |
| D48 | The opening-kill rating term scales with rounds played | held | parity until decided |
| D49 | Counter width (conditional, not needed) | process | no case found |
| D50 | The engine version is in the payload and on `Match`; reverses D6 | planned | v5 and v6 rows differ |
| D51 | Column renames ship in this release; no alias period | planned | no API stability contract |
| D52 | The parser cutover waits for the whole stat-layer rebuild | process | one production cutover |
| D53 | The lifecycle rebuild is in scope | process | the old parser was its only spec |
| D54 | Drop `distanceToTeammates` | done | working sum of the old lurk rule, never a distance |
| D55 | D44 leaves eco `timeAlive` unchanged; D47's decimal `atd` is accepted | done | the rulings covered survival and the fraction only |
| D56 | The map ends at the clinch or the game's own end, whichever comes first; a surrender or admin end is Ended with a reason; a disagreement is logged (not logged yet) | done | the "ended, not by our rule" 500 trips Core's Stats breaker |
| D57 | The payload carries `outcome` and `reason`; a 422 body is JSON `{outcome, reason, detail}` | planned | Core sees only a status code and text |
| D58 | The knife flag goes on the round being opened; a knife while live is a restart that commits nothing | done | the flag lands on the previous round today |
| D59 | One commit rule for every round, the deciding round included; the final round closes at the win panel | done | removes the deciding-round special case and the bomb hold |
| D60 | `Partial` is `ScoreOnly`; `lifecycle.md` defines map, match and series against Core's terms | done | Core's `STATS_PARTIAL` is a different thing |
| D61 | A restore into a new recording stays ScoreOnly; no stitching of recordings | process | Core sends only the latest recording |
| D62 | Go-live is the first open where both first members have 800 money; the `Warmup` flag is not read | done | parity; add a `!Warmup` gate only if a count shows a difference |
| D63 | A tie is Ended with reason `tie`, never a rejection | done (the reason comes with D57) | the parser can't know the match type; Combine ties are legal |
| D64 | The match-model and parser cutovers share one window after 2026-12-01 | process | one rehearsal, one production change |
