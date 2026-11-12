package payload

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// field is a struct field with a JSON name, found through the embedded structs.
type field struct {
	name  string
	index []int
}

// jsonFields lists t's fields by JSON name, in order, flattening embedded structs as encoding/json
// does. It skips `json:"-"` fields.
func jsonFields(t reflect.Type) []field {
	var out []field
	for i := range t.NumField() {
		f := t.Field(i)
		tag, hasTag := f.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" && tag == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				for _, sub := range jsonFields(ft) {
					out = append(out, field{sub.name, append([]int{i}, sub.index...)})
				}
				continue
			}
		}
		if !hasTag || name == "" {
			name = f.Name
		}
		out = append(out, field{name, []int{i}})
	}
	return out
}

// byName indexes a type's fields by JSON name; a name found twice is a bug in the row model.
func byName(t reflect.Type) map[string][]int {
	m := map[string][]int{}
	for _, f := range jsonFields(t) {
		if _, dup := m[f.name]; dup {
			panic(fmt.Sprintf("%s: JSON name %q found twice", t, f.name))
		}
		m[f.name] = f.index
	}
	return m
}

// An index is the field paths of one type, cached: a demo has tens of thousands of rows.
var indexes sync.Map // reflect.Type -> map[string][]int

func index(t reflect.Type) map[string][]int {
	if m, ok := indexes.Load(t); ok {
		return m.(map[string][]int)
	}
	m := byName(t)
	indexes.Store(t, m)
	return m
}

// A copyPlan pairs each field of dst with the source field of the same JSON name.
type copyPlan [][2][]int

var plans sync.Map // [2]reflect.Type -> copyPlan

// copyRow copies src's fields onto dst, which are pointers to structs. Every JSON name must be on
// both sides, and an integer must fit in dst's type, so a dropped or narrowed field fails here and
// not as a silent value change. Until the fold produces row types itself, this is how the collectors'
// rows become the payload's.
func copyRow(dst, src any) {
	dv, sv := reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem()
	key := [2]reflect.Type{dv.Type(), sv.Type()}
	p, ok := plans.Load(key)
	if !ok {
		p = newPlan(dv.Type(), sv.Type())
		plans.Store(key, p)
	}
	for _, pair := range p.(copyPlan) {
		d, s := dv.FieldByIndex(pair[0]), sv.FieldByIndex(pair[1])
		if d.Kind() == reflect.Int || d.Kind() == reflect.Int64 {
			d.SetInt(intOf(s))
		} else {
			d.Set(s)
		}
	}
}

func intOf(v reflect.Value) int64 {
	if v.CanInt() {
		return v.Int()
	}
	return int64(v.Uint())
}

func newPlan(dt, st reflect.Type) copyPlan {
	di, si := index(dt), index(st)
	var p copyPlan
	for name, d := range di {
		s, ok := si[name]
		if !ok {
			panic(fmt.Sprintf("%s has %q, %s doesn't", dt, name, st))
		}
		df, sf := dt.FieldByIndex(d).Type, st.FieldByIndex(s).Type
		switch {
		case df == sf:
		case (df.Kind() == reflect.Int || df.Kind() == reflect.Int64) && sf.Bits() <= 32 && isInt(sf):
		default:
			panic(fmt.Sprintf("%q: %s doesn't hold %s", name, df, sf))
		}
		p = append(p, [2][]int{d, s})
	}
	for name := range si {
		if _, ok := di[name]; !ok {
			panic(fmt.Sprintf("%s has %q, %s doesn't", st, name, dt))
		}
	}
	return p
}

func isInt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return true
	}
	return false
}
