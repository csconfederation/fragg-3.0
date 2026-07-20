// Package csc is a faithful port of the demoScrape2 (CSC) demo parser so that
// eco-rating can emit demoScrape2-compatible statistics under their original
// field names. The rating and per-round calculations in this package are kept
// intentionally identical to demoScrape2 so downstream CSC consumers see the
// same numbers they always have. Eco-rating's own (additive) metrics are merged
// in separately by the export package.
package csc

import "errors"

// ErrNoValidRounds indicates the demo contained no valid, complete rounds.
// It mirrors demoScrape2 v0.3.2 so the demo worker's classifier can map an
// unprocessable-but-not-broken demo to HTTP 422 instead of 5xx.
var ErrNoValidRounds = errors.New("demoscrape2: demo contains no valid rounds (ErrNoValidRounds)")

// debugLogger is a no-op logger implementing the subset of the logrus API used
// by the ported CSC pipeline. Debug output is discarded to keep the drop-in
// parser silent inside the demo worker (demoScrape2 ran these at debug level).
type debugLogger struct{}

func (debugLogger) Debug(args ...interface{})                 {}
func (debugLogger) Debugf(format string, args ...interface{}) {}

var log = debugLogger{}
