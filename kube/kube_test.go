package kube

import "testing"

func TestDefinitionsLoad(t *testing.T) {
	defs := Definitions()
	if len(defs) == 0 {
		t.Fatal("expected definitions to be loaded, got none")
	}
	if _, ok := defs["io.k8s.api.core.v1.ResourceRequirements"]; !ok {
		t.Error("expected core/v1 ResourceRequirements definition to be present")
	}
}

func TestSchemaVersionPinned(t *testing.T) {
	if KubeSchemaVersion == "" {
		t.Error("KubeSchemaVersion must name the embedded Kubernetes version")
	}
}
