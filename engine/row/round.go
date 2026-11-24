package row

import "github.com/csconfederation/fragg-3.0/engine/fact"

// Round is a committed round: its meta, bomb fields and player rows. The payload lists the player
// rows itself.
type Round struct {
	Meta
	Bomb
	Players map[uint64]*Player  `json:"-"`
	Teams   map[fact.Side]*Team `json:"-"`
}

type Meta struct {
	RoundNum           int     `json:"roundNum"`
	StartingTick       int     `json:"startingTick"`
	EndingTick         int     `json:"endingTick"`
	InitTerroristCount int     `json:"initTerroristCount"`
	InitCTCount        int     `json:"initCTerroristCount"`
	WinnerClanName     string  `json:"winnerClanName"`
	WinnerENUM         int     `json:"winnerENUM"` // the side that won: 2 (T) or 3 (CT)
	IntegrityCheck     bool    `json:"integrityCheck"`
	ServerNormalizer   int     `json:"serverNormalizer"`
	ServerImpactPoints float64 `json:"serverImpactPoints"`
	KnifeRound         bool    `json:"knifeRound"`
	RoundEndReason     string  `json:"roundEndReason"`
}

type Bomb struct {
	Planter           uint64 `json:"planter,string"`
	Defuser           uint64 `json:"defuser,string"`
	EndDueToBombEvent bool   `json:"endDueToBombEvent"`
	WinTeamDmg        int    `json:"winTeamDmg"`
	BombStartTick     int    `json:"bombStartTick"`
}
