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

func TestParse_SectionInlineNoBlankLine(t *testing.T) {
	yaml := []byte("# @section Inline\n# Description\nkey: val\n")
	nodes, err := Parse(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Section != "Inline" {
		t.Errorf("section: got %q, want %q", nodes[0].Section, "Inline")
	}
	if nodes[0].Description != "Description" {
		t.Errorf("description: got %q", nodes[0].Description)
	}
}

func TestParse_MultipleSectionsBetweenKeys(t *testing.T) {
	yaml := []byte("# @section First\n# @section Second\n\nkey: val\n")
	nodes, err := Parse(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Section != "Second" {
		t.Errorf("section: got %q, want %q (last one should win)", nodes[0].Section, "Second")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":\n  :\n    - [invalid"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParse_NonMappingRoot(t *testing.T) {
	_, err := Parse([]byte("just a string"))
	if err == nil {
		t.Error("expected error for non-mapping root")
	}
}

func TestParse_ArrayValues(t *testing.T) {
	input := []byte("# Allowed hosts\nhosts:\n  - example.com\n  - test.com\n# Empty list\ntags: []\n")
	nodes, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	if nodes[0].Type != "array" {
		t.Errorf("hosts type: got %q, want %q", nodes[0].Type, "array")
	}
	items, ok := nodes[0].Default.([]interface{})
	if !ok {
		t.Fatalf("hosts default: expected []interface{}, got %T", nodes[0].Default)
	}
	if len(items) != 2 {
		t.Errorf("hosts default: got %d items, want 2", len(items))
	}

	if nodes[1].Type != "array" {
		t.Errorf("tags type: got %q, want %q", nodes[1].Type, "array")
	}
	emptyItems, ok := nodes[1].Default.([]interface{})
	if !ok {
		t.Fatalf("tags default: expected []interface{}, got %T", nodes[1].Default)
	}
	if len(emptyItems) != 0 {
		t.Errorf("tags default: got %d items, want 0", len(emptyItems))
	}
}

func TestParse_NullValues(t *testing.T) {
	input := []byte("# Explicit null\na: null\n# Tilde null\nb: ~\n# Empty value\nc:\n")
	nodes, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	for i, n := range nodes {
		if n.Type != "null" {
			t.Errorf("[%d] type: got %q, want %q", i, n.Type, "null")
		}
		if n.Default != nil {
			t.Errorf("[%d] default: got %v, want nil", i, n.Default)
		}
	}
}
