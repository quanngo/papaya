package tenantconfig

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"claimconfig/component/models"
)

// Diff lists every difference between two configs, walking their JSON form:
//   - objects: compared key by key;
//   - arrays of scalars (enabled types, channels, documents…): compared as
//     sets, so order changes are not reported;
//   - arrays of objects: matched by the identity field from each module's
//     DiffKeys (tiers by role, custom fields by key), otherwise by position.
//
// Absent, null, "" and empty arrays/objects count as the same "no value".
func Diff(a, b *models.TenantConfig) []models.Change {
	keyed := map[string]string{}
	for _, m := range Modules {
		for path, key := range m.DiffKeys() {
			keyed[path] = key
		}
	}
	d := differ{keyed: keyed, out: []models.Change{}}
	d.walk("", "", toJSONValue(a), toJSONValue(b))
	return d.out
}

type differ struct {
	keyed map[string]string
	out   []models.Change
}

func toJSONValue(v any) any {
	raw, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}

// walk compares l and r at path; pattern is path without element identities,
// used to look up DiffKeys.
func (d *differ) walk(path, pattern string, l, r any) {
	l, r = normalize(l), normalize(r)
	switch {
	case l == nil && r == nil:
		return
	case l == nil:
		d.add(path, "added", nil, r)
		return
	case r == nil:
		d.add(path, "removed", l, nil)
		return
	}

	lm, lIsMap := l.(map[string]any)
	rm, rIsMap := r.(map[string]any)
	if lIsMap && rIsMap {
		keys := map[string]bool{}
		for k := range lm {
			keys[k] = true
		}
		for k := range rm {
			keys[k] = true
		}
		for _, k := range sortedKeys(keys) {
			d.walk(join(path, k), join(pattern, k), lm[k], rm[k])
		}
		return
	}

	ls, lIsList := l.([]any)
	rs, rIsList := r.([]any)
	if lIsList && rIsList {
		switch {
		case allScalars(ls) && allScalars(rs):
			d.diffSet(path, ls, rs)
		case d.keyed[pattern] != "":
			d.diffKeyed(path, pattern, d.keyed[pattern], ls, rs)
		default:
			n := len(ls)
			if len(rs) > n {
				n = len(rs)
			}
			for i := 0; i < n; i++ {
				d.walk(fmt.Sprintf("%s[%d]", path, i), pattern+"[]", at(ls, i), at(rs, i))
			}
		}
		return
	}

	if !reflect.DeepEqual(l, r) {
		d.add(path, "changed", l, r)
	}
}

func (d *differ) diffSet(path string, l, r []any) {
	for _, v := range l {
		if !hasValue(r, v) {
			d.add(path, "removed", v, nil)
		}
	}
	for _, v := range r {
		if !hasValue(l, v) {
			d.add(path, "added", nil, v)
		}
	}
}

func (d *differ) diffKeyed(path, pattern, key string, l, r []any) {
	byKey := func(list []any) (map[string]any, []string) {
		m, order := map[string]any{}, []string{}
		for i, el := range list {
			id := fmt.Sprintf("#%d", i)
			if obj, ok := el.(map[string]any); ok && obj[key] != nil {
				id = fmt.Sprint(obj[key])
			}
			if _, dup := m[id]; !dup {
				order = append(order, id)
			}
			m[id] = el
		}
		return m, order
	}
	lm, lOrder := byKey(l)
	rm, rOrder := byKey(r)
	order := append([]string{}, lOrder...)
	for _, id := range rOrder {
		if _, ok := lm[id]; !ok {
			order = append(order, id)
		}
	}
	for _, id := range order {
		d.walk(fmt.Sprintf("%s[%s=%s]", path, key, id), pattern+"[]", lm[id], rm[id])
	}
}

func (d *differ) add(path, kind string, l, r any) {
	module := path
	if i := strings.IndexAny(path, ".["); i >= 0 {
		module = path[:i]
	}
	d.out = append(d.out, models.Change{Path: path, Module: module, Kind: kind, Left: l, Right: r})
}

func normalize(v any) any {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
	case []any:
		if len(t) == 0 {
			return nil
		}
	case map[string]any:
		if len(t) == 0 {
			return nil
		}
	}
	return v
}

func allScalars(list []any) bool {
	for _, v := range list {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

func hasValue(list []any, v any) bool {
	for _, x := range list {
		if reflect.DeepEqual(x, v) {
			return true
		}
	}
	return false
}

func at(list []any, i int) any {
	if i < len(list) {
		return list[i]
	}
	return nil
}

func join(path, k string) string {
	if path == "" {
		return k
	}
	return path + "." + k
}
