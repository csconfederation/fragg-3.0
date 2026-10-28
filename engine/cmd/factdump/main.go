// Command factdump prints a demo's facts as JSON lines, for debugging the engine.
//
//	go run ./engine/cmd/factdump DEMO.dem
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/csconfederation/fragg-3.0/engine/fact"
	"github.com/csconfederation/fragg-3.0/engine/ingest"
)

type printer struct{ enc *json.Encoder }

func (p printer) Fact(f fact.Fact) error {
	if fr, ok := f.(fact.Frame); ok { // Sample is a function; print the frame's clock only
		return p.enc.Encode(map[string]any{"Frame": struct{ Tick, Frame int }{fr.Tick, fr.Frame}})
	}
	return p.enc.Encode(map[string]any{strings.TrimPrefix(fmt.Sprintf("%T", f), "fact."): f})
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: factdump DEMO.dem")
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	term, err := ingest.Run(context.Background(), f, printer{json.NewEncoder(os.Stdout)})
	fmt.Fprintf(os.Stderr, "terminal kind %d frame %d err %v; sink error %v\n", term.Kind, term.Frame, term.Err, err)
}
