// Command payload runs the engine (ingest, match, CSC stats) over a demo and writes the oracle-
// shaped payload, gzipped, for the parity check. It prints one JSON status line.
//
//	go run ./engine/cmd/payload DEMO.dem OUT.json.gz
package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/csconfederation/fragg-3.0/engine/cscstats"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: payload DEMO.dem OUT.json.gz")
		os.Exit(2)
	}
	status := map[string]any{"demo": os.Args[1]}
	defer func() { json.NewEncoder(os.Stdout).Encode(status) }()
	f, err := os.Open(os.Args[1])
	if err != nil {
		status["error"] = err.Error()
		return
	}
	defer f.Close()
	stats := cscstats.New()
	m := match.New(stats)
	term, _ := ingest.Run(context.Background(), f, m)
	res, err := m.Finish(term)
	if err != nil {
		status["error"] = err.Error()
		return
	}
	status["outcome"] = res.Outcome.String()
	b, err := json.Marshal(stats.Game(res))
	if err != nil {
		status["error"] = "encode: " + err.Error()
		return
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		status["error"] = err.Error()
		return
	}
	defer out.Close()
	zw := gzip.NewWriter(out)
	defer zw.Close()
	zw.Write(b)
}
