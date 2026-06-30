package main

import (
	"os"
	"testing"

	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/schema"
)

func TestK8sEndToEndSchemaValidates(t *testing.T) {
	data, err := os.ReadFile("testdata/k8s/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, warnings, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}

	schemaBytes, err := schema.Generate(nodes)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	sch := compileE2ESchema(t, schemaBytes)

	// A valid ResourceRequirements value (Quantity values are strings).
	valid := unmarshalE2E(t, `{
		"resources": {"limits": {"cpu": "500m", "memory": "128Mi"}, "requests": {"cpu": "250m"}},
		"securityContext": null
	}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid Kubernetes document rejected: %v", err)
	}

	// resources.limits must be an object, not a string.
	invalid := unmarshalE2E(t, `{"resources": {"limits": "not-an-object"}}`)
	if err := sch.Validate(invalid); err == nil {
		t.Error("expected validation error for malformed resources.limits")
	}
}
