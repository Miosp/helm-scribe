package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParse_BasicScalars(t *testing.T) {
	data, err := os.ReadFile("../testdata/basic.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, _, err := Parse(data)
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

	nodes, _, err := Parse(data)
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

	nodes, _, err := Parse(data)
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
	nodes, _, err := Parse(yaml)
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
	nodes, _, err := Parse(yaml)
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
	_, _, err := Parse([]byte(":\n  :\n    - [invalid"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParse_NonMappingRoot(t *testing.T) {
	_, _, err := Parse([]byte("just a string"))
	if err == nil {
		t.Error("expected error for non-mapping root")
	}
}

func TestParse_ArrayValues(t *testing.T) {
	input := []byte("# Allowed hosts\nhosts:\n  - example.com\n  - test.com\n# Empty list\ntags: []\n")
	nodes, _, err := Parse(input)
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
	nodes, _, err := Parse(input)
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

func TestParse_TypeOverride(t *testing.T) {
	data, err := os.ReadFile("../testdata/types.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, warnings, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		typ      string
		nullable bool
		items    int
	}{
		{"nullable_name", "string", false, 0},
		{"optional_label", "string", true, 0},
		{"tags", "string[]", false, 0},
		{"hosts", "object[]", false, 4},
		{"unknown", "null", false, 0},
	}

	if len(nodes) != len(tests) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(tests))
	}

	for i, tt := range tests {
		n := nodes[i]
		if n.Path != tt.path {
			t.Errorf("[%d] path: got %q, want %q", i, n.Path, tt.path)
		}
		if n.Type != tt.typ {
			t.Errorf("[%d] type: got %q, want %q", i, n.Type, tt.typ)
		}
		if n.Nullable != tt.nullable {
			t.Errorf("[%d] nullable: got %v, want %v", i, n.Nullable, tt.nullable)
		}
		if len(n.Items) != tt.items {
			t.Errorf("[%d] items: got %d, want %d", i, len(n.Items), tt.items)
		}
	}

	hasNullWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "unknown") && strings.Contains(w, "null") {
			hasNullWarning = true
		}
	}
	if !hasNullWarning {
		t.Errorf("expected null warning for 'unknown', got warnings: %v", warnings)
	}
}

func TestParse_Phase2Annotations(t *testing.T) {
	data, err := os.ReadFile("../testdata/phase2.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, _, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 6 {
		t.Fatalf("got %d nodes, want 6", len(nodes))
	}

	// @enum
	if len(nodes[0].Enum) != 3 || nodes[0].Enum[0] != "Always" {
		t.Errorf("pullPolicy enum: got %v", nodes[0].Enum)
	}

	// @min/@max
	if nodes[1].Min == nil || *nodes[1].Min != 1 {
		t.Errorf("port min: got %v", nodes[1].Min)
	}
	if nodes[1].Max == nil || *nodes[1].Max != 65535 {
		t.Errorf("port max: got %v", nodes[1].Max)
	}

	// @pattern
	if nodes[2].Pattern != "^[a-z][a-z0-9-]*$" {
		t.Errorf("appName pattern: got %q", nodes[2].Pattern)
	}

	// @deprecated
	if nodes[3].Deprecated != "Use newSetting instead" {
		t.Errorf("oldSetting deprecated: got %q", nodes[3].Deprecated)
	}

	// @example
	if nodes[4].Example != "my-custom-app" {
		t.Errorf("displayName example: got %q", nodes[4].Example)
	}

	// @default override
	if nodes[5].DefaultOverride == nil || *nodes[5].DefaultOverride != "See values.yaml" {
		t.Errorf("extraConfig defaultOverride: got %v", nodes[5].DefaultOverride)
	}
}
