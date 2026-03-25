package parser

import (
	"os"
	"testing"
)

func TestParse_BasicScalars(t *testing.T) {
	data, err := os.ReadFile("../testdata/basic.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	expected := []struct {
		path        string
		description string
		typ         string
		defVal      interface{}
	}{
		{"replicaCount", "Number of replicas", "integer", 1},
		{"fullnameOverride", "Override the full name", "string", ""},
		{"debug", "Enable debug mode", "boolean", false},
		{"cpuLimit", "Resource CPU limit", "number", 0.5},
	}

	if len(nodes) != len(expected) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(expected))
	}

	for i, e := range expected {
		n := nodes[i]
		if n.Path != e.path {
			t.Errorf("[%d] path: got %q, want %q", i, n.Path, e.path)
		}
		if n.Description != e.description {
			t.Errorf("[%d] description: got %q, want %q", i, n.Description, e.description)
		}
		if n.Type != e.typ {
			t.Errorf("[%d] type: got %q, want %q", i, n.Type, e.typ)
		}
	}
}
