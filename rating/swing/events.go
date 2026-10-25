package swing

import "github.com/csconfederation/fragg-3.0/engine/fact"

// KillEvent represents a kill during the round.
type KillEvent struct {
	TimeInRound         float64
	KillerID            uint64
	VictimID            uint64
	KillerSide          fact.Side
	VictimSide          fact.Side
	KillerEquip         float64
	VictimEquip         float64
	IsTradeKill         bool
	IsHeadshot          bool
	TotalDamageToVictim int
	KillerDamageDealt   int
	DamageContributors  []DamageContributor
	FlashAssists        []FlashAssist
}

// DamageContributor tracks a player's damage contribution to a kill.
type DamageContributor struct {
	PlayerID uint64
	Damage   int
}

// FlashAssist tracks flash assist details.
type FlashAssist struct {
	PlayerID uint64
	Duration float64 // Seconds the victim was flashed
}

// KillSwingResult contains swing metadata for a single kill event.
type KillSwingResult struct {
	RawSwing      float64
	EcoMultiplier float64
}
