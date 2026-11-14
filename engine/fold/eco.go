package fold

import (
	"github.com/csconfederation/fragg-3.0/engine/row"
	"github.com/csconfederation/fragg-3.0/rating"
)

// eco computes the eco rates and ratings of one player's rows. The eco sums are the family sums
// (swing, kill values, times and the eco kills, damage and KAST); a side row carries its side's.
// A bot's row gets values too, which the payload's constants then zero: eco has no bot rows.
func (m *Match) eco(id uint64, tallies map[*row.Player]*tally) {
	whole, t, ct := m.Whole[id], m.T[id], m.CT[id]
	w := tallies[whole]

	// Whole row: the swing rating is the side rating's formula over every round.
	whole.SwingRating = swingRating(whole)
	whole.EcoProbabilitySwingPerRound = perRound(whole.EcoProbabilitySwing, whole.Rounds)
	whole.EcoDuelSwing = whole.EcoKillValue - whole.EcoDeathValue
	whole.EcoDuelSwingPerRound = perRound(whole.EcoDuelSwing, whole.Rounds)
	whole.EcoSwingDisplayRating = swingDisplayRating(whole.EcoProbabilitySwing, whole.Rounds)
	whole.AWPKillsPerRound = perRound(float64(whole.EcoAWPKills), whole.Rounds)
	rounds := float64(whole.Rounds)
	whole.TimeAlivePerRound = whole.EcoTimeAlive / rounds
	if w.died > 0 {
		whole.AvgTimeToDeath = whole.EcoDeathTime / float64(w.died)
	}
	if whole.KillsWithTTK > 0 {
		whole.AvgTimeToKill = whole.TimeToKillSum / float64(whole.KillsWithTTK)
	}
	whole.HLTVRating = rating.ComputeHLTVRating(rating.HLTVInput{RoundsPlayed: whole.Rounds, Kills: whole.EcoKills,
		Survivals: w.survived, MultiKills: w.multi})

	for _, s := range []*row.Player{t, ct} {
		if s == nil {
			continue
		}
		st := tallies[s]
		s.SwingRating = swingRating(s)
		s.EcoProbabilitySwingPerRound = perRound(s.EcoProbabilitySwing, s.Rounds)
		s.EcoDuelSwing = s.EcoKillValue - s.EcoDeathValue
		s.EcoDuelSwingPerRound = perRound(s.EcoDuelSwing, s.Rounds)
		s.EcoSwingDisplayRating = swingDisplayRating(s.EcoProbabilitySwing, s.Rounds)
		s.AWPKillsPerRound = perRound(float64(s.EcoAWPKills), s.Rounds)
		s.HLTVRating = rating.ComputeSideHLTVRating(s.Rounds, s.EcoKills, st.survived, st.multi)
	}
	// A side row also names its own side's variants.
	if t != nil {
		t.EcoTProbabilitySwing, t.HLTVTRating, t.EcoTEcoRating = t.EcoProbabilitySwing, t.HLTVRating, t.SwingRating
	}
	if ct != nil {
		ct.EcoCTProbabilitySwing, ct.HLTVCtRating, ct.EcoCTEcoRating = ct.EcoProbabilitySwing, ct.HLTVRating, ct.SwingRating
	}
}

// swingRating is the eco rating over a row's rounds: damage, KAST and probability swing.
func swingRating(p *row.Player) float64 {
	return rating.ComputeSideRating(p.Rounds, p.EcoKills, p.Deaths, p.EcoDamage, p.EcoProbabilitySwing, p.EcoKastRounds, false)
}

func perRound(total float64, rounds int) float64 {
	if rounds <= 0 {
		return 0
	}
	return total / float64(rounds)
}

// swingDisplayRating scales probability swing into the display rating used by
// ecoSwingDisplayRating (0% = 1.0, +4% = 1.4, -3% = 0.7), clamped to [0.5, 1.5].
func swingDisplayRating(probSwing float64, rounds int) float64 {
	if rounds <= 0 {
		return 0
	}
	rating := 1.0 + (probSwing/float64(rounds))*10.0
	if rating < 0.5 {
		return 0.5
	}
	if rating > 1.5 {
		return 1.5
	}
	return rating
}
