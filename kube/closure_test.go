package kube

import "testing"

func TestClosureIncludesReferencedTypes(t *testing.T) {
	defs, err := Closure("io.k8s.api.core.v1.ResourceRequirements")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := defs["io.k8s.api.core.v1.ResourceRequirements"]; !ok {
		t.Error("closure must contain the root type")
	}
	// ResourceRequirements references Quantity (resource limits/requests).
	if _, ok := defs["io.k8s.apimachinery.pkg.api.resource.Quantity"]; !ok {
		t.Error("closure must pull in transitively referenced Quantity")
	}
	// It must NOT drag in unrelated types.
	if _, ok := defs["io.k8s.api.core.v1.Pod"]; ok {
		t.Error("closure must not contain unrelated types")
	}
}

func TestClosureUnknownKey(t *testing.T) {
	_, err := Closure("io.k8s.api.core.v1.DoesNotExist")
	if err == nil {
		t.Fatal("expected error for unknown definition key")
	}
}

func TestClosureTerminatesOnSelfReference(t *testing.T) {
	// JSONSchemaProps is self-referential; closure must not loop forever.
	key := "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps"
	if _, ok := Definitions()[key]; !ok {
		t.Skip("self-referential type not present in this schema version")
	}
	if _, err := Closure(key); err != nil {
		t.Fatalf("closure of self-referential type failed: %v", err)
	}
}
