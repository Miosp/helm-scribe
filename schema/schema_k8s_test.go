package schema

import (
	"encoding/json"
	"testing"

	"github.com/miosp/helm-scribe/model"
)

func generate(t *testing.T, nodes []*model.ValueNode) map[string]any {
	t.Helper()
	b, err := Generate(nodes)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return s
}

func TestK8sFieldEmitsRef(t *testing.T) {
	nodes := []*model.ValueNode{{
		Key: "resources", Path: "resources", Type: "k8s:core/v1/ResourceRequirements",
		K8sRef: "io.k8s.api.core.v1.ResourceRequirements",
	}}
	s := generate(t, nodes)
	props := s["properties"].(map[string]any)
	res := props["resources"].(map[string]any)
	if res["$ref"] != "#/definitions/io.k8s.api.core.v1.ResourceRequirements" {
		t.Errorf("got %v", res["$ref"])
	}
	defs, ok := s["definitions"].(map[string]any)
	if !ok {
		t.Fatal("expected definitions block")
	}
	if _, ok := defs["io.k8s.api.core.v1.ResourceRequirements"]; !ok {
		t.Error("definitions must include the referenced type")
	}
	if _, ok := defs["io.k8s.apimachinery.pkg.api.resource.Quantity"]; !ok {
		t.Error("definitions must include transitively referenced Quantity")
	}
}

func TestK8sArrayFieldEmitsItemsRef(t *testing.T) {
	nodes := []*model.ValueNode{{
		Key: "containers", Path: "containers", Type: "k8s:core/v1/Container[]",
		K8sRef: "io.k8s.api.core.v1.Container",
	}}
	s := generate(t, nodes)
	props := s["properties"].(map[string]any)
	c := props["containers"].(map[string]any)
	if c["type"] != "array" {
		t.Errorf("expected array, got %v", c["type"])
	}
	items := c["items"].(map[string]any)
	if items["$ref"] != "#/definitions/io.k8s.api.core.v1.Container" {
		t.Errorf("got %v", items["$ref"])
	}
}

func TestK8sNullableFieldUsesAnyOf(t *testing.T) {
	nodes := []*model.ValueNode{{
		Key: "sc", Path: "sc", Type: "k8s:core/v1/PodSecurityContext",
		K8sRef: "io.k8s.api.core.v1.PodSecurityContext", Nullable: true,
	}}
	s := generate(t, nodes)
	props := s["properties"].(map[string]any)
	sc := props["sc"].(map[string]any)
	anyOf, ok := sc["anyOf"].([]any)
	if !ok || len(anyOf) != 2 {
		t.Fatalf("expected anyOf with 2 entries, got %v", sc["anyOf"])
	}
}

func TestNoK8sMeansNoDefinitions(t *testing.T) {
	nodes := []*model.ValueNode{{Key: "name", Path: "name", Type: "string", Default: "x"}}
	s := generate(t, nodes)
	if _, ok := s["definitions"]; ok {
		t.Error("definitions block must be absent when no Kubernetes types are used")
	}
}

func TestK8sNullableArrayFieldUsesTypeArray(t *testing.T) {
	nodes := []*model.ValueNode{{
		Key: "sidecars", Path: "sidecars", Type: "k8s:core/v1/Container[]",
		K8sRef: "io.k8s.api.core.v1.Container", Nullable: true,
	}}
	s := generate(t, nodes)
	props := s["properties"].(map[string]any)
	sc := props["sidecars"].(map[string]any)
	types, ok := sc["type"].([]any)
	if !ok {
		t.Fatalf("expected type to be a JSON array (nullable array), got %T: %v", sc["type"], sc["type"])
	}
	strTypes := make([]string, len(types))
	for i, tv := range types {
		strTypes[i] = tv.(string)
	}
	if strTypes[0] != "array" && strTypes[1] != "array" {
		t.Errorf("expected type to include \"array\", got %v", strTypes)
	}
	items := sc["items"].(map[string]any)
	if items["$ref"] != "#/definitions/io.k8s.api.core.v1.Container" {
		t.Errorf("got items $ref %v", items["$ref"])
	}
}

func TestK8sNullableItemsArrayWrapsItemsInAnyOf(t *testing.T) {
	nodes := []*model.ValueNode{{
		Key: "sidecars", Path: "sidecars", Type: "k8s:core/v1/Container[]",
		K8sRef: "io.k8s.api.core.v1.Container", ItemNullable: true,
	}}
	s := generate(t, nodes)
	props := s["properties"].(map[string]any)
	sc := props["sidecars"].(map[string]any)
	if sc["type"] != "array" {
		t.Errorf("expected array type, got %v", sc["type"])
	}
	items := sc["items"].(map[string]any)
	anyOf, ok := items["anyOf"].([]any)
	if !ok || len(anyOf) != 2 {
		t.Fatalf("expected items.anyOf with 2 entries (ref + null), got %v", items)
	}
}

func TestK8sNodeDoesNotCollectChildRefs(t *testing.T) {
	// Parent is a Kubernetes-typed node (Quantity). Its transitive closure is
	// itself only ({Quantity}); it does NOT reference Pod. The child carries an
	// unrelated K8sRef (Pod). Because a k8s-typed node short-circuits nodeSchema
	// to a bare $ref, its children's schemas are never emitted, so their K8sRefs
	// must not be collected into the definitions closure.
	// Without the recursion guard this test fails: Pod appears in definitions.
	parent := &model.ValueNode{
		Key: "qty", Path: "qty", Type: "k8s:core/v1/Quantity",
		K8sRef: "io.k8s.apimachinery.pkg.api.resource.Quantity",
		Children: []*model.ValueNode{{
			Key:    "child",
			Path:   "qty.child",
			Type:   "k8s:core/v1/Pod",
			K8sRef: "io.k8s.api.core.v1.Pod",
		}},
	}
	s := generate(t, []*model.ValueNode{parent})
	defs := s["definitions"].(map[string]any)
	if _, ok := defs["io.k8s.apimachinery.pkg.api.resource.Quantity"]; !ok {
		t.Error("parent Quantity definition must be present")
	}
	if _, ok := defs["io.k8s.api.core.v1.Pod"]; ok {
		t.Error("child Pod ref must NOT be pulled into definitions; k8s node children must not be collected")
	}
}
