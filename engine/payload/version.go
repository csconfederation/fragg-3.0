package payload

import "runtime/debug"

const module = "github.com/csconfederation/fragg-3.0"

// engineVersion is the fragg build that made the payload (D50). A worker build has fragg as a
// dependency, so it is the module version, whose pseudo-version carries the commit. fragg's own
// build (cmd/payload) gets a stamped main-module version (pseudo-version, "+dirty" if modified);
// without one it is "devel-<commit>" from the VCS settings, and without build info (go test) "devel".
var engineVersion = func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	for _, d := range bi.Deps {
		if d.Path == module {
			if d.Replace != nil {
				d = d.Replace
			}
			if d.Version == "" { // a local replace has no version
				return "devel"
			}
			return d.Version
		}
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	v := "devel"
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			v += "-" + s.Value[:min(len(s.Value), 12)]
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if dirty {
		v += "+dirty"
	}
	return v
}()
