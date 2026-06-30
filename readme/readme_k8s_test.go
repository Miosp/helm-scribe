package readme

import (
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/model"
)

func k8sNode() *model.ValueNode {
	return &model.ValueNode{
		Key: "resources", Path: "resources",
		Type: "k8s:core/v1/ResourceRequirements",
		K8sRef: "io.k8s.api.core.v1.ResourceRequirements",
		Description: "Compute resources",
	}
}

func TestCompactRowShowsGVKInTypeColumn(t *testing.T) {
	opts := Options{TruncateLength: 80, TypeColumn: true}
	out := Generate([]*model.ValueNode{k8sNode()}, opts)
	if !strings.Contains(out, "core/v1/ResourceRequirements") {
		t.Errorf("type column should show the GVK:\n%s", out)
	}
	// No sub-field rows when expansion is off.
	if strings.Contains(out, "limits") {
		t.Errorf("compact mode must not expand sub-fields:\n%s", out)
	}
}

func TestK8sNodeWithChildrenStaysCompact(t *testing.T) {
	// A k8s-typed field that has a concrete object value in values.yaml will
	// carry inferred Children. flatten must still treat it as a single leaf row,
	// not descend into the inferred structure.
	n := k8sNode()
	n.Children = []*model.ValueNode{
		{Key: "limits", Path: "resources.limits", Type: "object"},
	}
	opts := Options{TruncateLength: 80, TypeColumn: true}
	out := Generate([]*model.ValueNode{n}, opts)
	if !strings.Contains(out, "`resources`") {
		t.Errorf("k8s field row missing:\n%s", out)
	}
	if strings.Contains(out, "`resources.limits`") {
		t.Errorf("flatten must not descend into a k8s field's inferred children:\n%s", out)
	}
}
