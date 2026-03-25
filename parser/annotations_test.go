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
