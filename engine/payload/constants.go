package payload

import (
	"fmt"
	"reflect"
)

// The scopes of constants, and the rows each applies to.
const (
	scopeRoundPlayer = "round-player" // a player's row in a round
	scopeBot         = "bot"          // the shared Steam64-0 row, in every set
	scopeSideT       = "side-T"       // a T side row
	scopeSideCT      = "side-CT"      // a CT side row
	scopeTeamMatch   = "team-match"   // a team's match row
	scopeRound       = "round"        // a round's meta
)

// constants is the one place the cells the baseline forces are named: scope, then JSON name, then
// value. Every other cell is computed where the baseline has it and left 0 elsewhere. The sets
// were checked against the corpus, and against the code: no collector writes a round-player field
// listed here.
var constants = map[string]map[string]any{
	// 109 fields no collector writes on a round row: the 46 eco-origin fields, the rates, ratings,
	// projections, whole-only rating inputs and buy-scope fields the fold computes, rounds and wlp.
	scopeRoundPlayer: with(zeros(ecoOrigin...), zeros(
		"rounds", "wlp", "utilThrown", "atd", "kast", "killPointAvg", "iiwr", "adr", "drDiff", "KR", "tr",
		"rating", "impactRating",
		"TDamage", "ctDamage", "TImpactPoints", "TWinPoints", "TOK", "TOL", "ctImpactPoints", "ctWinPoints",
		"ctOK", "ctOL", "TKills", "TDeaths", "TKAST", "TKASTRounds", "TADR", "ctKills", "ctDeaths", "ctKAST",
		"ctKASTRounds", "ctADR", "TRounds", "ctRounds", "ctRating", "ctImpactRating", "TRating", "TImpactRating",
		"TADP", "ctADP", "TRF",
		"TTeamsWinPoints", "ctTeamsWinPoints", "TWinPointsNormalizer", "ctWinPointsNormalizer", "teamsWinPoints",
		"winPointsNormalizer",
		"pistolRounds", "pistolRoundsWon", "pistolRoundRating", "ecoBuyRounds", "ecoBuyRoundsWon", "ecoBuyRating",
		"forceBuyRounds", "forceBuyRoundsWon", "forceBuyRating", "fullBuyRounds", "fullBuyRoundsWon", "fullBuyRating",
		"antiEcoRounds", "antiEcoRoundsWon", "antiEcoRating",
	), map[string]any{"health": 100}),
	// eco has no bot rows.
	scopeBot: zeros(ecoOrigin...),
	// Q57: side rows keep 0 for these seven; each side row also keeps the other side's variants at 0.
	scopeSideT:  with(q57(), zeros("ecoCTProbabilitySwing", "hltvCtRating", "ecoCTEcoRating")),
	scopeSideCT: with(q57(), zeros("ecoTProbabilitySwing", "hltvTRating", "ecoTEcoRating")),
	// The team rows' points exist only on the round rows.
	scopeTeamMatch: zeros("winPoints", "impactPoints", "TWinPoints", "ctWinPoints", "TImpactPoints",
		"ctImpactPoints", "normalizer"),
	scopeRound: {"integrityCheck": true, "knifeRound": false, "serverNormalizer": 0, "serverImpactPoints": 0},
}

// The eco stats the fold merges onto the match and side rows (fold/eco.go).
var ecoOrigin = []string{
	"swing_rating", "ecoProbabilitySwing", "ecoProbabilitySwingPerRound", "ecoTProbabilitySwing",
	"ecoCTProbabilitySwing", "ecoKillValue", "ecoDeathValue", "ecoDuelSwing", "ecoDuelSwingPerRound",
	"ecoAdjustedKills", "hltvRating", "hltvCtRating", "hltvTRating", "ecoCTEcoRating", "ecoTEcoRating",
	"ecoSwingDisplayRating", "assistedKills", "clutch1v1Attempts", "clutch1v2Attempts", "clutch1v3Attempts",
	"clutch1v4Attempts", "clutch1v5Attempts", "tradeKills", "tradeDenials", "fastTrades", "openingDeathsTraded",
	"awpOpeningKills", "awpMultiKillRounds", "awpDeaths", "roundsWithAWPKill", "awpKillsPerRound",
	"timeAlivePerRound", "avgTimeToDeath", "avgTimeToKill", "manAdvantageKills", "manDisadvantageDeaths",
	"exitFrags", "knifeKills", "pistolVsRifleKills", "killsVsLowerBuy", "killsVsMuchLowerBuy", "utilityKills",
	"perfectKills", "earlyDeaths", "savedByTeammate", "savedTeammate",
}

func zeros(names ...string) map[string]any {
	m := map[string]any{}
	for _, n := range names {
		m[n] = 0
	}
	return m
}

func q57() map[string]any {
	return zeros("ecoAdjustedKills", "timeAlivePerRound", "avgTimeToDeath", "avgTimeToKill", "killsVsLowerBuy",
		"killsVsMuchLowerBuy", "perfectKills")
}

func with(ms ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range ms {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// setConstants sets dst's cells for scope to the table's values.
func setConstants(dst any, scope string) {
	dv := reflect.ValueOf(dst).Elem()
	idx := index(dv.Type())
	for name, val := range constants[scope] {
		path, ok := idx[name]
		if !ok {
			panic(fmt.Sprintf("constants[%s]: %s has no %q", scope, dv.Type(), name))
		}
		f := dv.FieldByIndex(path)
		switch v := val.(type) {
		case int:
			if f.CanInt() {
				f.SetInt(int64(v))
			} else {
				f.SetFloat(float64(v))
			}
		case bool:
			f.SetBool(v)
		default:
			panic(fmt.Sprintf("constants[%s][%s]: unsupported %T", scope, name, val))
		}
	}
}
