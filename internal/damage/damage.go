//go:build !damage_v5

// Package damage picks the HP a PlayerHurt counts. Production uses the
// game-accurate HealthDamageTaken (Core-planning#292 D1). Building with
// -tags damage_v5 restores v5's value, only for the v5→v6 delta report (#300),
// so the ~+1 HP doesn't hide the other upstream changes.
package damage

import "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"

func Taken(e events.PlayerHurt) int { return e.HealthDamageTaken }
