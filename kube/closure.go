package kube

import (
	"encoding/json"
	"fmt"
	"strings"
)

const definitionsRefPrefix = "#/definitions/"

// Closure returns the given definition keys plus every definition reachable
// through $ref, keyed by definition name. A visited set makes it safe for
// self-referential types.
func Closure(keys ...string) (map[string]json.RawMessage, error) {
	all := Definitions()
	out := make(map[string]json.RawMessage)
	var visit func(key string) error
	visit = func(key string) error {
		if _, seen := out[key]; seen {
			return nil
		}
		raw, ok := all[key]
		if !ok {
			return fmt.Errorf("unknown Kubernetes definition %q", key)
		}
		out[key] = raw
		for _, ref := range collectRefs(raw) {
			target := strings.TrimPrefix(ref, definitionsRefPrefix)
			if target == ref {
				continue // not an internal definitions ref
			}
			if err := visit(target); err != nil {
				return err
			}
		}
		return nil
	}
	for _, k := range keys {
		if err := visit(k); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// collectRefs walks an arbitrary JSON value and returns every "$ref" string.
func collectRefs(raw json.RawMessage) []string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	var refs []string
	var walk func(node any)
	walk = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			for k, child := range n {
				if k == "$ref" {
					if s, ok := child.(string); ok {
						refs = append(refs, s)
					}
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range n {
				walk(child)
			}
		}
	}
	walk(v)
	return refs
}
