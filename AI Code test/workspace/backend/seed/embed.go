// Package seed embeds the sample tenant configurations inserted on first start
// (when the tenant table is empty) and reused by the engine tests.
package seed

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed *.json
var files embed.FS

// Tenant is one seed file: a slug plus a full tenant config.
type Tenant struct {
	Slug   string          `json:"slug"`
	Config json.RawMessage `json:"config"`
}

// Load returns every seed tenant, ordered by file name.
func Load() ([]Tenant, error) {
	names, err := fs.Glob(files, "*.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	out := make([]Tenant, 0, len(names))
	for _, name := range names {
		raw, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var t Tenant
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, fmt.Errorf("seed %s: %w", name, err)
		}
		out = append(out, t)
	}
	return out, nil
}
