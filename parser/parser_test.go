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
	}{
		{"replicaCount", "Number of replicas", "integer"},
		{"fullnameOverride", "Override the full name", "string"},
		{"debug", "Enable debug mode", "boolean"},
		{"cpuLimit", "Resource CPU limit", "number"},
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

func TestParse_NestedObjects(t *testing.T) {
	data, err := os.ReadFile("../testdata/nested.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 1 {
		t.Fatalf("got %d top-level nodes, want 1", len(nodes))
	}

	img := nodes[0]
	if img.Path != "image" {
		t.Errorf("path: got %q", img.Path)
	}
	if img.Description != "Container image configuration" {
		t.Errorf("description: got %q", img.Description)
	}
	if len(img.Children) != 2 {
		t.Fatalf("got %d children, want 2", len(img.Children))
	}

	repo := img.Children[0]
	if repo.Path != "image.repository" {
		t.Errorf("child path: got %q", repo.Path)
	}
	if repo.Type != "string" {
		t.Errorf("child type: got %q", repo.Type)
	}
}

func TestParse_SectionsAndSkip(t *testing.T) {
	data, err := os.ReadFile("../testdata/sections.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	// reconcileInterval is skipped, so 3 nodes
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}

	if nodes[0].Section != "Common parameters" {
		t.Errorf("[0] section: got %q", nodes[0].Section)
	}
	if nodes[1].Section != "Common parameters" {
		t.Errorf("[1] section: got %q", nodes[1].Section)
	}
	if nodes[2].Section != "Network parameters" {
		t.Errorf("[2] section: got %q", nodes[2].Section)
	}
}
