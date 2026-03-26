package parser

import "testing"

func TestParseAnnotations_DescriptionOnly(t *testing.T) {
	input := "# Number of replicas for the deployment"
	ann := ParseAnnotations(input)
	if ann.Description != "Number of replicas for the deployment" {
		t.Errorf("got description %q", ann.Description)
	}
	if ann.Section != "" {
		t.Errorf("got unexpected section %q", ann.Section)
	}
	if ann.Skip {
		t.Error("got unexpected skip")
	}
}

func TestParseAnnotations_MultiLineDescription(t *testing.T) {
	input := "# Number of replicas\n# for the deployment"
	ann := ParseAnnotations(input)
	if ann.Description != "Number of replicas for the deployment" {
		t.Errorf("got description %q", ann.Description)
	}
}

func TestParseAnnotations_LineBreakInDescription(t *testing.T) {
	input := "# First paragraph\n#\n# Second paragraph"
	ann := ParseAnnotations(input)
	expected := "First paragraph\nSecond paragraph"
	if ann.Description != expected {
		t.Errorf("got description %q, want %q", ann.Description, expected)
	}
}

func TestParseAnnotations_Section(t *testing.T) {
	input := "# @section Common parameters"
	ann := ParseAnnotations(input)
	if ann.Section != "Common parameters" {
		t.Errorf("got section %q", ann.Section)
	}
	if ann.Description != "" {
		t.Errorf("got unexpected description %q", ann.Description)
	}
}

func TestParseAnnotations_Skip(t *testing.T) {
	input := "# Internal setting\n# @skip"
	ann := ParseAnnotations(input)
	if !ann.Skip {
		t.Error("expected skip to be true")
	}
	if ann.Description != "Internal setting" {
		t.Errorf("got description %q", ann.Description)
	}
}

func TestParseAnnotations_Empty(t *testing.T) {
	ann := ParseAnnotations("")
	if ann.Description != "" || ann.Section != "" || ann.Skip {
		t.Errorf("expected empty annotation, got %+v", ann)
	}
}

func TestParseAnnotations_Type(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		typ          string
		nullable     bool
		itemNullable bool
	}{
		{"plain type", "# @type string", "string", false, false},
		{"nullable type", "# @type string?", "string", true, false},
		{"array type", "# @type string[]", "string[]", false, false},
		{"nullable array", "# @type string[]?", "string[]", true, false},
		{"array of nullable", "# @type string?[]", "string[]", false, true},
		{"nullable array of nullable", "# @type string?[]?", "string[]", true, true},
		{"integer type", "# @type integer", "integer", false, false},
		{"object type", "# @type object", "object", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ann := ParseAnnotations(tt.input)
			if ann.Type != tt.typ {
				t.Errorf("type: got %q, want %q", ann.Type, tt.typ)
			}
			if ann.Nullable != tt.nullable {
				t.Errorf("nullable: got %v, want %v", ann.Nullable, tt.nullable)
			}
			if ann.ItemNullable != tt.itemNullable {
				t.Errorf("itemNullable: got %v, want %v", ann.ItemNullable, tt.itemNullable)
			}
		})
	}
}

func TestParseAnnotations_Item(t *testing.T) {
	input := "# List of hosts\n# @item host: string\n# @item paths: object[]"
	ann := ParseAnnotations(input)

	if len(ann.Items) != 2 {
		t.Fatalf("items: got %d, want 2", len(ann.Items))
	}
	if ann.Items[0].Path != "host" || ann.Items[0].Type != "string" {
		t.Errorf("item[0]: got %+v", ann.Items[0])
	}
	if ann.Items[1].Path != "paths" || ann.Items[1].Type != "object[]" {
		t.Errorf("item[1]: got %+v", ann.Items[1])
	}
	if ann.Description != "List of hosts" {
		t.Errorf("description: got %q", ann.Description)
	}
}

func TestParseAnnotations_ItemNestedPath(t *testing.T) {
	input := "# @item paths[].path: string\n# @item paths[].pathType: string"
	ann := ParseAnnotations(input)

	if len(ann.Items) != 2 {
		t.Fatalf("items: got %d, want 2", len(ann.Items))
	}
	if ann.Items[0].Path != "paths[].path" {
		t.Errorf("item[0] path: got %q", ann.Items[0].Path)
	}
}
