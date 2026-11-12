// Package roundlog stores a committed round's deriver input as a gzipped JSON fixture, with the
// rows today's collectors produce from it, and replays the fixture through the Deriver and the
// collectors (stat-layer rebuild PR 2). Facts are tagged by kind: {"kind": "Kill", "fact": {...}}.
package roundlog

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/csconfederation/fragg-3.0/engine/fact"
)

// kinds is every fact type, by name. A test checks it against engine/fact.
var kinds = func() map[string]reflect.Type {
	m := map[string]reflect.Type{}
	for _, f := range []fact.Fact{fact.Header{}, fact.ServerInfo{}, fact.RoundStart{}, fact.Frame{}, fact.FreezeEnd{},
		fact.RoundEnd{}, fact.ScoreUpdated{}, fact.RoundEndOfficial{}, fact.Kill{}, fact.Hurt{}, fact.Flashed{},
		fact.BombPlanted{}, fact.BombDefused{}, fact.BombExplode{}, fact.GrenadeThrow{}, fact.TeamChange{},
		fact.PlayerInfo{}, fact.Connect{}, fact.Disconnect{}, fact.Footstep{}, fact.GamePhase{},
		fact.WinPanelMatch{}} {
		m[reflect.TypeOf(f).Name()] = reflect.TypeOf(f)
	}
	return m
}()

// frame is a Frame without its Sample function.
type frame struct{ Tick, Frame int }

// kindOf names f and returns what encodes it.
func kindOf(f fact.Fact) (string, any, error) {
	name := reflect.TypeOf(f).Name()
	if _, ok := kinds[name]; !ok {
		return "", nil, fmt.Errorf("fact kind %s has no entry in the kind table", name)
	}
	if fr, ok := f.(fact.Frame); ok {
		return name, frame{fr.Tick, fr.Frame}, nil
	}
	return name, f, nil
}

// factOf decodes a tagged fact.
func factOf(kind string, raw json.RawMessage) (fact.Fact, error) {
	t, ok := kinds[kind]
	if !ok {
		return nil, fmt.Errorf("unknown fact kind %q", kind)
	}
	if kind == "Frame" {
		var fr frame
		if err := json.Unmarshal(raw, &fr); err != nil {
			return nil, err
		}
		return fact.Frame{Tick: fr.Tick, Frame: fr.Frame}, nil
	}
	v := reflect.New(t)
	if err := json.Unmarshal(raw, v.Interface()); err != nil {
		return nil, fmt.Errorf("fact %s: %w", kind, err)
	}
	return v.Elem().Interface().(fact.Fact), nil
}
