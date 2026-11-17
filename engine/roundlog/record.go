package roundlog

import (
	"fmt"
	"reflect"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
	"github.com/csconfederation/fragg-3.0/engine/round"
)

// Recorder is a match.Stats that passes everything to next (the Deriver) and keeps the input of
// the rounds Want selects, from each open until the next.
type Recorder struct {
	Next match.Stats
	Want func(num int) bool
	// Logs are the recorded rounds by their lifecycle round, a File each without Demo, Sha256 or Rows.
	Logs    map[*match.Round]*File
	mapName string
	cur     *File
	key     *match.Round
}

func (r *Recorder) Open(mr *match.Round, s fact.Snapshot, roster match.Roster) error {
	r.cur, r.key = nil, mr
	if r.Want(mr.Num) {
		r.cur = &File{Version: Version, Map: r.mapName, Round: roundOf(mr), Open: s, Roster: roster, Commit: -1}
		if r.Logs == nil {
			r.Logs = map[*match.Round]*File{}
		}
		r.Logs[mr] = r.cur
	}
	return r.Next.Open(mr, s, roster)
}

func (r *Recorder) Event(e match.Event) error {
	if si, ok := e.Fact.(fact.ServerInfo); ok && !e.Dropped {
		r.mapName = si.Map
	}
	if r.cur != nil {
		if h, ok := e.Fact.(fact.Hurt); ok { // the buffer keeps only these; so does the fixture
			h.Snapshot = fact.Snapshot{Tick: h.Tick, RoundsPlayed: h.RoundsPlayed, Warmup: h.Warmup}
			e.Fact = h
		}
		r.cur.Events = append(r.cur.Events, Event{e})
	}
	return r.Next.Event(e)
}

func (r *Recorder) Commit(mr *match.Round, tick int) error {
	if r.cur != nil && r.key == mr {
		r.cur.Commit, r.cur.EndTick = len(r.cur.Events), tick
	}
	return r.Next.Commit(mr, tick)
}

// Check replays the fixture f made from the handed-over round got, and returns an error unless it
// rebuilds the same round, rows included. f.Rows is set from the replay.
func Check(f *File, got *round.Round, want *Rows) error {
	rp, err := Replay(f)
	if err != nil {
		return fmt.Errorf("replay: %w", err)
	}
	a, b := *got, *rp.Round
	a.Key, b.Key = nil, nil
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("the replayed round differs from the original: %s", firstDiff(reflect.ValueOf(a), reflect.ValueOf(b), "round"))
	}
	wb, err := marshal(want)
	if err != nil {
		return err
	}
	gb, err := marshal(rp.Rows)
	if err != nil {
		return err
	}
	if string(wb) != string(gb) {
		return fmt.Errorf("the replayed rows differ from the match's")
	}
	f.Rows = wb
	return nil
}

// firstDiff names the first field where two equal-typed values differ, to point a failure.
func firstDiff(a, b reflect.Value, path string) string {
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if a.Type().Field(i).IsExported() && !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
				return firstDiff(a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name)
			}
		}
	case reflect.Slice:
		if a.Len() != b.Len() {
			return fmt.Sprintf("%s length %d vs %d", path, a.Len(), b.Len())
		}
		for i := 0; i < a.Len(); i++ {
			if !reflect.DeepEqual(a.Index(i).Interface(), b.Index(i).Interface()) {
				return firstDiff(a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i))
			}
		}
	case reflect.Interface, reflect.Pointer:
		if !a.IsNil() && !b.IsNil() && a.Elem().Type() == b.Elem().Type() {
			return firstDiff(a.Elem(), b.Elem(), path)
		}
	}
	return path
}
