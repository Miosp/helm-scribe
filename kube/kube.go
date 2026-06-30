package kube

import (
	_ "embed"
	"encoding/json"
	"sync"
)

// KubeSchemaVersion names the embedded Kubernetes JSON Schema version.
// To target a different version, change this constant and replace
// kube/definitions.json with the matching upstream _definitions.json.
const KubeSchemaVersion = "v1.32.1"

//go:embed definitions.json
var rawDefinitions []byte

var (
	defsOnce sync.Once
	defs     map[string]json.RawMessage
)

// Definitions returns the embedded Kubernetes definitions keyed by their
// fully-qualified name (e.g. "io.k8s.api.core.v1.Container").
func Definitions() map[string]json.RawMessage {
	defsOnce.Do(func() {
		var doc struct {
			Definitions map[string]json.RawMessage `json:"definitions"`
		}
		if err := json.Unmarshal(rawDefinitions, &doc); err != nil {
			panic("kube: embedded definitions.json is invalid: " + err.Error())
		}
		defs = doc.Definitions
	})
	return defs
}
