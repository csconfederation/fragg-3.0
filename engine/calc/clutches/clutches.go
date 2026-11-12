// Package clutches is the clutches family: the won clutch by size (cl_N), the clutch attempts by
// size, and the team's clutch (design note section 3). The slots are the Deriver's: one per side,
// the last member alive and the enemies left when they became it (corpus finding 6). A clutch won
// is the winning side's slot, for CSC; an attempt is either side's, for eco (D23).
package clutches

import (
	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/round"
	"github.com/csconfederation/fragg-3.0/engine/row"
)

// Calc writes each player's Clutches and each team's TeamClutches.
func Calc(r *round.Round, out *row.Round) {
	win, won := r.WinAt(r.Commit)
	if !won {
		return
	}
	slots := []struct {
		side fact.Side
		c    round.Clutch
	}{{fact.SideT, r.End.TClutch}, {fact.SideCT, r.End.CTClutch}}

	// Attempts are eco's: every slot with enemies left, held by a human with a row. Eco has no bot rows.
	for _, s := range slots {
		if p := out.Players[s.c.Steam]; p != nil && s.c.Steam != 0 && s.c.Size > 0 {
			attempt(&p.Clutches, s.c.Size)
		}
	}
	// A clutch won is the winner's slot, for a clutcher who isn't a bot. A size outside 1v0 to 1v5
	// has no bonus in CSC, so it is no clutch; the bonus index doesn't reach 6.
	for _, s := range slots {
		if win.Side != s.side || s.c.Steam == 0 {
			continue
		}
		if s.c.Size < 0 || s.c.Size > 5 {
			break
		}
		if t := out.Teams[r.SideOf(win.Clan)]; t != nil { // the winner's team row
			t.Clutches = 1
		}
		if p := out.Players[s.c.Steam]; p != nil {
			switch s.c.Size {
			case 1:
				p.Cl1 = 1
			case 2:
				p.Cl2 = 1
			case 3:
				p.Cl3 = 1
			case 4:
				p.Cl4 = 1
			case 5:
				p.Cl5 = 1
			}
		}
		break
	}
}

func attempt(c *row.Clutches, size int) {
	switch size {
	case 1:
		c.Clutch1v1Attempts = 1
	case 2:
		c.Clutch1v2Attempts = 1
	case 3:
		c.Clutch1v3Attempts = 1
	case 4:
		c.Clutch1v4Attempts = 1
	case 5:
		c.Clutch1v5Attempts = 1
	}
}
