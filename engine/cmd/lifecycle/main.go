// Command lifecycle runs ingest and match over a demo and prints the result as one JSON line, for
// the parity check against the oracle.
//
//	go run ./engine/cmd/lifecycle ROOT REL_PATH
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/csconfederation/fragg-3.0/engine/ingest"
	"github.com/csconfederation/fragg-3.0/engine/match"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: lifecycle ROOT REL_PATH")
		os.Exit(2)
	}
	out := map[string]any{"path": os.Args[2]}
	defer func() { json.NewEncoder(os.Stdout).Encode(out) }()
	f, err := os.Open(filepath.Join(os.Args[1], os.Args[2]))
	if err != nil {
		out["error"] = err.Error()
		return
	}
	defer f.Close()
	m := match.New()
	term, sinkErr := ingest.Run(context.Background(), f, m)
	out["terminal"] = term.Kind
	if sinkErr != nil {
		out["sinkError"] = sinkErr.Error()
	}
	res, err := m.Finish(term)
	if err != nil {
		out["error"] = err.Error()
		return
	}
	out["outcome"] = res.Outcome.String()
	out["result"] = res
}
