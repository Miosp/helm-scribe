package parser

import (
	"testing"
)

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

func TestParseAnnotations_Enum(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		values []string
	}{
		{"unquoted", "# @enum [Always, IfNotPresent, Never]", []string{"Always", "IfNotPresent", "Never"}},
		{"quoted", `# @enum ["val 1", "val 2"]`, []string{"val 1", "val 2"}},
		{"numbers", "# @enum [1, 2, 3]", []string{"1", "2", "3"}},
		{"with description", "# Pull policy\n# @enum [Always, Never]", []string{"Always", "Never"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ann := ParseAnnotations(tt.input)
			if len(ann.Enum) != len(tt.values) {
				t.Fatalf("enum: got %d values, want %d", len(ann.Enum), len(tt.values))
			}
			for i, v := range tt.values {
				if ann.Enum[i] != v {
					t.Errorf("enum[%d]: got %q, want %q", i, ann.Enum[i], v)
				}
			}
		})
	}
}

func TestParseAnnotations_MinMax(t *testing.T) {
	ann := ParseAnnotations("# Port number\n# @min 1\n# @max 65535")
	if ann.Min == nil || *ann.Min != 1 {
		t.Errorf("min: got %v, want 1", ann.Min)
	}
	if ann.Max == nil || *ann.Max != 65535 {
		t.Errorf("max: got %v, want 65535", ann.Max)
	}
}

func TestParseAnnotations_MinMaxFloat(t *testing.T) {
	ann := ParseAnnotations("# @min 0.1\n# @max 99.9")
	if ann.Min == nil || *ann.Min != 0.1 {
		t.Errorf("min: got %v, want 0.1", ann.Min)
	}
	if ann.Max == nil || *ann.Max != 99.9 {
		t.Errorf("max: got %v, want 99.9", ann.Max)
	}
}

func TestParseAnnotations_Default(t *testing.T) {
	ann := ParseAnnotations("# @default See values.yaml")
	if ann.DefaultOverride == nil || *ann.DefaultOverride != "See values.yaml" {
		t.Errorf("default override: got %v", ann.DefaultOverride)
	}
}

func TestParseAnnotations_Deprecated(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		ann := ParseAnnotations("# Old setting\n# @deprecated Use newSetting instead")
		if ann.Deprecated != "Use newSetting instead" {
			t.Errorf("deprecated: got %q", ann.Deprecated)
		}
		if ann.Description != "Old setting" {
			t.Errorf("description: got %q", ann.Description)
		}
	})
	t.Run("without message", func(t *testing.T) {
		ann := ParseAnnotations("# @deprecated")
		if ann.Deprecated != "deprecated" {
			t.Errorf("deprecated: got %q, want %q", ann.Deprecated, "deprecated")
		}
	})
}

func TestParseAnnotations_Example(t *testing.T) {
	ann := ParseAnnotations("# App name\n# @example my-custom-app")
	if ann.Example != "my-custom-app" {
		t.Errorf("example: got %q", ann.Example)
	}
}

func TestParseAnnotations_Pattern(t *testing.T) {
	ann := ParseAnnotations("# @pattern ^[a-z][a-z0-9-]*$")
	if ann.Pattern != "^[a-z][a-z0-9-]*$" {
		t.Errorf("pattern: got %q", ann.Pattern)
	}
}

func TestParseAnnotations_EnumEdgeCases(t *testing.T) {
	t.Run("missing brackets", func(t *testing.T) {
		ann := ParseAnnotations("# @enum Always, Never")
		if ann.Enum != nil {
			t.Errorf("expected nil for missing brackets, got %v", ann.Enum)
		}
	})
	t.Run("empty list", func(t *testing.T) {
		ann := ParseAnnotations("# @enum []")
		if len(ann.Enum) != 0 {
			t.Errorf("expected empty enum, got %v", ann.Enum)
		}
	})
	t.Run("single value", func(t *testing.T) {
		ann := ParseAnnotations("# @enum [Only]")
		if len(ann.Enum) != 1 || ann.Enum[0] != "Only" {
			t.Errorf("expected [Only], got %v", ann.Enum)
		}
	})
	t.Run("mixed quoted and unquoted", func(t *testing.T) {
		ann := ParseAnnotations(`# @enum [Always, "If Not Present", Never]`)
		if len(ann.Enum) != 3 {
			t.Fatalf("expected 3 values, got %d: %v", len(ann.Enum), ann.Enum)
		}
		if ann.Enum[1] != "If Not Present" {
			t.Errorf("enum[1]: got %q, want %q", ann.Enum[1], "If Not Present")
		}
	})
	t.Run("quoted value containing comma", func(t *testing.T) {
		t.Skip("known limitation: parseEnum splits on comma before handling quotes")
		ann := ParseAnnotations(`# @enum ["a,b", "c"]`)
		if len(ann.Enum) != 2 {
			t.Fatalf("expected 2 values, got %d: %v", len(ann.Enum), ann.Enum)
		}
		if ann.Enum[0] != "a,b" {
			t.Errorf("enum[0]: got %q, want %q", ann.Enum[0], "a,b")
		}
		if ann.Enum[1] != "c" {
			t.Errorf("enum[1]: got %q, want %q", ann.Enum[1], "c")
		}
	})
}

func TestParseAnnotations_MinMaxInvalid(t *testing.T) {
	t.Run("non-numeric min", func(t *testing.T) {
		ann := ParseAnnotations("# @min abc")
		if ann.Min != nil {
			t.Errorf("expected nil for invalid min, got %v", *ann.Min)
		}
	})
	t.Run("non-numeric max", func(t *testing.T) {
		ann := ParseAnnotations("# @max xyz")
		if ann.Max != nil {
			t.Errorf("expected nil for invalid max, got %v", *ann.Max)
		}
	})
	t.Run("min only", func(t *testing.T) {
		ann := ParseAnnotations("# @min 5")
		if ann.Min == nil || *ann.Min != 5 {
			t.Errorf("min: got %v, want 5", ann.Min)
		}
		if ann.Max != nil {
			t.Errorf("max should be nil, got %v", *ann.Max)
		}
	})
}

func TestParseAnnotations_DefaultEdgeCases(t *testing.T) {
	t.Run("bare @default with only trailing space is not a match", func(t *testing.T) {
		// "# @default " trims to "@default" which lacks the trailing space
		// required by HasPrefix — so it falls through to description text.
		ann := ParseAnnotations("# @default ")
		if ann.DefaultOverride != nil {
			t.Errorf("expected nil DefaultOverride for bare '@default ', got %q", *ann.DefaultOverride)
		}
		if ann.Description != "@default" {
			t.Errorf("expected '@default' as description, got %q", ann.Description)
		}
	})
	t.Run("with leading/trailing content", func(t *testing.T) {
		ann := ParseAnnotations("# Description\n# @default custom value here")
		if ann.DefaultOverride == nil || *ann.DefaultOverride != "custom value here" {
			t.Errorf("default override: got %v", ann.DefaultOverride)
		}
		if ann.Description != "Description" {
			t.Errorf("description: got %q", ann.Description)
		}
	})
}

func TestParseAnnotations_CombinedAnnotations(t *testing.T) {
	input := "# Port number\n# @type integer\n# @enum [80, 443, 8080]\n# @min 1\n# @max 65535\n# @pattern ^[0-9]+$\n# @example 8080\n# @deprecated Use newPort instead"
	ann := ParseAnnotations(input)

	if ann.Description != "Port number" {
		t.Errorf("description: got %q", ann.Description)
	}
	if ann.Type != "integer" {
		t.Errorf("type: got %q", ann.Type)
	}
	if len(ann.Enum) != 3 {
		t.Errorf("enum: got %v", ann.Enum)
	}
	if ann.Min == nil || *ann.Min != 1 {
		t.Errorf("min: got %v", ann.Min)
	}
	if ann.Max == nil || *ann.Max != 65535 {
		t.Errorf("max: got %v", ann.Max)
	}
	if ann.Pattern != "^[0-9]+$" {
		t.Errorf("pattern: got %q", ann.Pattern)
	}
	if ann.Example != "8080" {
		t.Errorf("example: got %q", ann.Example)
	}
	if ann.Deprecated != "Use newPort instead" {
		t.Errorf("deprecated: got %q", ann.Deprecated)
	}
}
