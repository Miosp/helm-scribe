package parser

import (
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/model"
)

func findNode(t *testing.T, data string, path string) *model.ValueNode {
	t.Helper()
	nodes, _, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, n := range nodes {
		if n.Path == path {
			return n
		}
	}
	t.Fatalf("node %q not found", path)
	return nil
}

func TestK8sTypeSetsRef(t *testing.T) {
	data := "# @type k8s:core/v1/ResourceRequirements\nresources: {}\n"
	n := findNode(t, data, "resources")
	if n.K8sRef != "io.k8s.api.core.v1.ResourceRequirements" {
		t.Errorf("K8sRef = %q", n.K8sRef)
	}
}

func TestK8sArrayType(t *testing.T) {
	data := "# @type k8s:core/v1/Container[]\ncontainers: []\n"
	n := findNode(t, data, "containers")
	if n.K8sRef != "io.k8s.api.core.v1.Container" {
		t.Errorf("K8sRef = %q", n.K8sRef)
	}
	if !strings.HasSuffix(n.Type, "[]") {
		t.Errorf("expected array Type, got %q", n.Type)
	}
}

func TestK8sNullableType(t *testing.T) {
	data := "# @type k8s:core/v1/PodSecurityContext?\nsecurityContext:\n"
	n := findNode(t, data, "securityContext")
	if n.K8sRef != "io.k8s.api.core.v1.PodSecurityContext" {
		t.Errorf("K8sRef = %q", n.K8sRef)
	}
	if !n.Nullable {
		t.Error("expected Nullable true")
	}
}

func TestK8sUnknownTypeWarns(t *testing.T) {
	data := "# @type k8s:core/v1/Bogus\nx: {}\n"
	nodes, warnings, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if nodes[0].K8sRef != "" {
		t.Error("unknown type must not set K8sRef")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Bogus") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning naming the bad type, got %v", warnings)
	}
}

func TestK8sTypeIgnoresConflictingConstraints(t *testing.T) {
	data := "# @type k8s:core/v1/ResourceRequirements\n# @min 1\nresources: {}\n"
	nodes, warnings, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if nodes[0].Min != nil {
		t.Error("@min must be cleared on a Kubernetes-typed field")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "ignored") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning about ignored constraint, got %v", warnings)
	}
}
