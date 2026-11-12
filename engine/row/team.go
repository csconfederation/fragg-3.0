package row

// Team is a team's match stats. The payload adds its name and score.
type Team struct {
	TeamOpenings
	TeamClutches
	TeamSurvival
	TeamEconomy
	TeamObjectives
	TeamMeta
	FromPlayers
}

type TeamOpenings struct {
	FourVFiveW int `json:"fourVFiveW"`
	FourVFiveS int `json:"fourVFiveS"`
	FiveVFourW int `json:"fiveVFourW"`
	FiveVFourS int `json:"fiveVFourS"`
}

type TeamClutches struct {
	Clutches int `json:"clutches"`
}

type TeamSurvival struct {
	Saves int `json:"saves"`
}

// TeamEconomy is the rounds and round wins by buy; pistols are the pistol rounds.
type TeamEconomy struct {
	Pistols    int `json:"pistols"`
	PistolsW   int `json:"pistolsW"`
	EcoBuys    int `json:"ecoBuys"`
	EcoBuysW   int `json:"ecoBuysW"`
	ForceBuys  int `json:"forceBuys"`
	ForceBuysW int `json:"forceBuysW"`
	FullBuys   int `json:"fullBuys"`
	FullBuysW  int `json:"fullBuysW"`
	AntiEcos   int `json:"antiEcos"`
	AntiEcosW  int `json:"antiEcosW"`
}

type TeamObjectives struct {
	WinPoints      float64 `json:"winPoints"`
	ImpactPoints   float64 `json:"impactPoints"`
	TWinPoints     float64 `json:"TWinPoints"`
	CtWinPoints    float64 `json:"ctWinPoints"`
	TImpactPoints  float64 `json:"TImpactPoints"`
	CtImpactPoints float64 `json:"ctImpactPoints"`
}

type TeamMeta struct {
	TR         int `json:"TR"`
	TRW        int `json:"TRW"`
	CtR        int `json:"ctR"`
	CtRW       int `json:"ctRW"`
	Normalizer int `json:"normalizer"`
}

// FromPlayers are summed from the team's player match rows.
type FromPlayers struct {
	Traded int `json:"traded"`
	Fass   int `json:"fass"`
	Ef     int `json:"ef"`
	Ud     int `json:"ud"`
	Util   int `json:"util"`
	Deaths int `json:"deaths"`
}
