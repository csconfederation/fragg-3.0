package fold

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/csconfederation/fragg-3.0/engine/row"
)

// sum adds src's counters to dst: the integer and float fields of the per-round family structs,
// except `fold:"-"` ones, which are round-only. A field of another kind without that tag is a bug
// in the row model and panics.
func sum(dst, src any) {
	dv, sv := reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem()
	for _, path := range plan(dv.Type()) {
		d, s := dv.FieldByIndex(path), sv.FieldByIndex(path)
		if d.CanInt() {
			d.SetInt(d.Int() + s.Int())
		} else {
			d.SetFloat(d.Float() + s.Float())
		}
	}
}

var plans sync.Map // reflect.Type -> [][]int

func plan(t reflect.Type) [][]int {
	if p, ok := plans.Load(t); ok {
		return p.([][]int)
	}
	p := summable(t, nil)
	plans.Store(t, p)
	return p
}

// summable is the paths to t's summed fields. A Player's sums are its families': not its identity or
// the fold's own rates, ratings, projections and buys.
func summable(t reflect.Type, prefix []int) [][]int {
	var out [][]int
	for i := range t.NumField() {
		f := t.Field(i)
		path := append(append([]int(nil), prefix...), i)
		switch {
		case f.Tag.Get("fold") == "-", foldOnly[f.Type]:
		case f.Anonymous && f.Type.Kind() == reflect.Struct:
			out = append(out, summable(f.Type, path)...)
		case f.Type.Kind() == reflect.Int, f.Type.Kind() == reflect.Float64:
			out = append(out, path)
		default:
			panic(fmt.Sprintf("fold: %s.%s (%s) can't be summed; tag it fold:\"-\"", t, f.Name, f.Type))
		}
	}
	return out
}

// The Player's parts the fold computes rather than sums.
var foldOnly = map[reflect.Type]bool{
	reflect.TypeFor[row.ID]():         true,
	reflect.TypeFor[row.Rates]():      true,
	reflect.TypeFor[row.Ratings]():    true,
	reflect.TypeFor[row.Projection](): true,
	reflect.TypeFor[row.Buys]():       true,
}
