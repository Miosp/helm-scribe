package kube

import (
	"strings"
	"testing"
)

func TestResolveCoreType(t *testing.T) {
	key, err := Resolve("core/v1/ResourceRequirements")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "io.k8s.api.core.v1.ResourceRequirements" {
		t.Errorf("got %q", key)
	}
}

func TestResolveNormalizesGroupSuffix(t *testing.T) {
	// networking.k8s.io normalizes to package segment "networking".
	key, err := Resolve("networking.k8s.io/v1/Ingress")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "io.k8s.api.networking.v1.Ingress" {
		t.Errorf("got %q", key)
	}
}

func TestResolveUnknownType(t *testing.T) {
	_, err := Resolve("core/v1/NotARealKind")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "core/v1/NotARealKind") {
		t.Errorf("error should name the input, got %q", err.Error())
	}
}

func TestResolveMalformedInput(t *testing.T) {
	_, err := Resolve("Container")
	if err == nil {
		t.Fatal("expected error for input that is not group/version/Kind")
	}
}
