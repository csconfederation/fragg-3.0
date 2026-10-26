# fragg

fragg's engine parses a CS2 demo into the payload csgo-demo-worker sends CSC-Stats: the match
lifecycle, CSC's stats and the eco/swing ratings, folded into one payload (Core-planning#292).
csgo-demo-worker calls `payload.Run`; nothing else in this repo is an entry point.

## Packages

| Package | Role |
|---|---|
| `engine/ingest` | Reads the demo (demoinfocs v6) and emits facts (`engine/fact`) |
| `engine/match` | The match lifecycle: live state, round opens, wins, commits, outcome |
| `engine/round` | Shared round state and one definition per shared concept: alive counts, phase, opening, clutches, deaths, assists, trades, flashes. Collectors read its `Context` |
| `engine/cscstats` | CSC's stats per round |
| `engine/eco` | Eco stats and the probability swing (`rating/swing`, `rating/probability`) |
| `engine/fold` | Folds both into match, T and CT rows and derives the ratings |
| `engine/payload` | `Run`: the whole pipeline, the payload and its error sentinels |
| `rating`, `model` | Rating formulas and weights; eco's per-round and per-player structs |

Rating formulas: [docs/rating.md](docs/rating.md).

## Commands

```bash
go run ./engine/cmd/payload DEMO.dem OUT.json.gz   # the payload, and one JSON status line
go run ./engine/cmd/factdump DEMO.dem              # the facts as JSON lines, for debugging
```

## Adding a stat

1. If the demo doesn't carry it yet, add it to a fact in `engine/fact` and read it in `engine/ingest`.
2. If CSC and eco both use the concept, decide it once in `engine/round` and expose it on `Context`.
3. Count it in `engine/cscstats` or `engine/eco`, and fold it in `engine/fold`.
4. Add the payload field (`engine/cscstats/types.go`), then the Stats column (CSC-Stats).

## Tests

`go test ./...` runs everywhere. The demo tests need `FRAGG_CI_DEMOS`: the CI demos, listed by
sha256 in `testdata/ci/demos.json`, live in the public `cscdemos` bucket under `ci/`. To limit
bucket traffic, CI parses them only on a merge to `main` or a manual run.

```bash
scripts/fetch-ci-demos.sh /tmp/ci-demos
FRAGG_CI_DEMOS=/tmp/ci-demos go test ./engine/...
FRAGG_CI_DEMOS=/tmp/ci-demos go test ./engine/payload -run TestCIGoldens -update   # after an approved output change
```

Floats compare within 1e-9.
