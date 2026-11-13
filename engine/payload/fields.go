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
