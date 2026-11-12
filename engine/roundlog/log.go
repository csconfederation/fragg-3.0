package roundlog

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

// Version is the fixture format.
const Version = 1

// Event is a match.Event as stored. Final, the lifecycle's round end processing, has no fact. A
// Frame's Sample is the lurk-cadence snapshot, beside the fact.
type Event struct{ match.Event }

type eventJSON struct {
	Kind    string          `json:"kind"`
	Fact    json.RawMessage `json:"fact,omitempty"`
	Sample  *fact.Snapshot  `json:"sample,omitempty"`
	Roster  *match.Roster   `json:"roster,omitempty"`
	Live    bool            `json:"live,omitempty"`
	Dropped bool            `json:"dropped,omitempty"`
	Join    *match.Member   `json:"join,omitempty"`
	Win     bool            `json:"win,omitempty"`
	Winner  string          `json:"winner,omitempty"`
}

const finalKind = "Final"

func (e Event) MarshalJSON() ([]byte, error) {
	j := eventJSON{Kind: finalKind, Sample: e.Sample, Live: e.Live, Dropped: e.Dropped, Join: e.Join, Win: e.Win,
		Winner: e.Winner}
	if len(e.Roster.T) > 0 || len(e.Roster.CT) > 0 {
		j.Roster = &e.Roster
	}
	if e.Fact != nil {
		kind, v, err := kindOf(e.Fact)
		if err != nil {
			return nil, err
		}
		j.Kind = kind
		if j.Fact, err = marshal(v); err != nil {
			return nil, err
		}
	} else if !e.Final {
		return nil, fmt.Errorf("event without a fact that isn't Final")
	}
	return marshal(j)
}

func (e *Event) UnmarshalJSON(b []byte) error {
	var j eventJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*e = Event{match.Event{Sample: j.Sample, Live: j.Live, Dropped: j.Dropped, Join: j.Join, Win: j.Win, Winner: j.Winner}}
	if j.Roster != nil {
		e.Roster = *j.Roster
	}
	if j.Kind == finalKind {
		e.Final = true
		return nil
	}
	f, err := factOf(j.Kind, j.Fact)
	e.Fact = f
	return err
}

// File is one fixture: a committed round's deriver input and the rows the collectors made from it.
type File struct {
	Version int    `json:"version"`
	Demo    string `json:"demo"` // the demo's file name
	Sha256  string `json:"sha256"`
	Map     string `json:"map"` // ServerInfo's map at the open
	Round   Round  `json:"round"`
	// Open is the snapshot and roster of the round's open.
	Open   fact.Snapshot `json:"open"`
	Roster match.Roster  `json:"roster"`
	// Events are every event the deriver got from the open until the next one, in order: the
	// round's buffer, plus what it doesn't buffer (Final, found PlayerInfo, non-live facts).
	Events []Event `json:"events"`
	// Commit is the index of the first event after the commit; EndTick is the commit's tick.
	Commit  int             `json:"commit"`
	EndTick int             `json:"endTick"`
	Rows    json.RawMessage `json:"rows"`
}

// Round is the lifecycle's round at its open.
type Round struct {
	Num       int            `json:"num"`
	StartTick int            `json:"startTick"`
	Pistol    bool           `json:"pistol"`
	FreezeEnd bool           `json:"freezeEnd"`
	InitT     int            `json:"initT"`
	InitCT    int            `json:"initCT"`
	TClan     string         `json:"tClan"`
	CTClan    string         `json:"ctClan"`
	Members   []match.Member `json:"members"` // by Steam64
}

// marshal is json.Marshal without HTML escaping, so names stay readable; either way is stable.
func marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Encode is the fixture's JSON, uncompressed.
func Encode(f *File) ([]byte, error) { return marshal(f) }

// Decode parses a fixture's JSON.
func Decode(b []byte) (*File, error) {
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Version != Version {
		return nil, fmt.Errorf("fixture version %d, want %d", f.Version, Version)
	}
	return &f, nil
}

// Gzip compresses b; the header carries no time or name, so the bytes depend only on b.
func Gzip(b []byte) ([]byte, error) {
	var out bytes.Buffer
	zw, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Gunzip decompresses a fixture file.
func Gunzip(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(zr)
}
