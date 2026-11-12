package cscstats

import "github.com/csconfederation/fragg-3.0/engine/match"

// Round is the committed round's row as the collector holds it, nil for an unknown round.
func (c *Collector) Round(r *match.Round) *Round { return c.rounds[r] }
