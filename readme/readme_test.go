package readme

import (
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/model"
)

// assertRowContains checks that a table row exists containing all given substrings.
func assertRowContains(t *testing.T, table string, substrings ...string) {
	t.Helper()
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		allFound := true
		for _, s := range substrings {
			if !strings.Contains(line, s) {
				allFound = false
				break
			}
		}
		if allFound {
			return
		}
	}
	t.Errorf("no row containing %v in:\n%s", substrings, table)
}

func TestGenerate_BasicTable(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "replicaCount", Description: "Number of replicas", Type: "integer", Default: 1, Section: "Common"},
		{Path: "fullnameOverride", Description: "Override full name", Type: "string", Default: "", Section: "Common"},
	}

	result := Generate(nodes, DefaultOptions())

	if !strings.Contains(result, "## Common") {
		t.Error("missing section header")
	}
	assertRowContains(t, result, "`replicaCount`", "Number of replicas", "`1`")
	assertRowContains(t, result, "`fullnameOverride`", "Override full name")
}

func TestGenerate_MultipleSections(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "replicas", Description: "Replicas", Default: 1, Section: "Common"},
		{Path: "port", Description: "Port", Default: 80, Section: "Network"},
	}

	result := Generate(nodes, DefaultOptions())

	commonIdx := strings.Index(result, "## Common")
	networkIdx := strings.Index(result, "## Network")
	if commonIdx == -1 || networkIdx == -1 {
		t.Fatalf("missing section headers, got:\n%s", result)
	}
	if commonIdx >= networkIdx {
		t.Error("sections in wrong order")
	}
}

func TestGenerate_NestedObjectFlattening(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Path:        "image",
			Description: "Container image config",
			Type:        "object",
			Section:     "Image",
			Children: []*model.ValueNode{
				{Path: "image.repository", Description: "Repo", Default: "nginx", Section: "Image"},
				{Path: "image.tag", Description: "Tag", Default: "latest", Section: "Image"},
			},
		},
	}

	result := Generate(nodes, DefaultOptions())

	assertRowContains(t, result, "`image.repository`", "Repo")
	assertRowContains(t, result, "`image.tag`", "Tag")
}

func TestGenerate_Truncation(t *testing.T) {
	longDefault := strings.Repeat("a", 100)
	nodes := []*model.ValueNode{
		{Path: "key", Description: "Desc", Default: longDefault, Section: "S"},
	}

	opts := DefaultOptions()
	opts.TruncateLength = 80
	result := Generate(nodes, opts)

	if !strings.Contains(result, "See values.yaml") {
		t.Errorf("expected truncation, got:\n%s", result)
	}
}

func TestGenerate_NoSection(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "key", Description: "Desc", Default: "val"},
	}

	result := Generate(nodes, DefaultOptions())

	if strings.Contains(result, "##") {
		t.Errorf("unexpected section header for unsectioned nodes, got:\n%s", result)
	}
	assertRowContains(t, result, "`key`", "Desc")
}

func TestInsertIntoReadme_BetweenMarkers(t *testing.T) {
	existing := "# My Chart\n\nSome intro text\n\n<!-- helm-scribe:start -->\nold content here\n<!-- helm-scribe:end -->\n\n## Other manual content\n"

	table := "| Key | Description | Default |\n|-----|-------------|--------|\n| `key` | Desc | `val` |\n"

	result, err := InsertIntoReadme(existing, table)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "<!-- helm-scribe:start -->") {
		t.Error("missing start marker")
	}
	if !strings.Contains(result, "<!-- helm-scribe:end -->") {
		t.Error("missing end marker")
	}
	if strings.Contains(result, "old content here") {
		t.Error("old content not replaced")
	}
	if !strings.Contains(result, "| `key` |") {
		t.Errorf("new content missing, got:\n%s", result)
	}
	if !strings.Contains(result, "## Other manual content") {
		t.Error("manual content after markers was lost")
	}
}

func TestInsertIntoReadme_NoMarkers(t *testing.T) {
	existing := "# My Chart\n\nSome text\n"
	table := "| Key |\n"

	_, err := InsertIntoReadme(existing, table)
	if err == nil {
		t.Error("expected error for missing markers")
	}
}

func TestFormatDefault_Types(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{"nil", nil, "`null`"},
		{"bool_true", true, "`true`"},
		{"bool_false", false, "`false`"},
		{"integer", 42, "`42`"},
		{"string", "hello", "`\"hello\"`"},
		{"empty_array", []interface{}{}, "`[]`"},
		{"non_empty_array", []interface{}{"a", "b"}, "See values.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDefault(tt.val, 80)
			if got != tt.want {
				t.Errorf("formatDefault(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestGenerate_HeadingLevel(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "key", Description: "Desc", Default: "val", Section: "Sec"},
	}

	t.Run("custom_level_3", func(t *testing.T) {
		opts := DefaultOptions()
		opts.HeadingLevel = 3
		result := Generate(nodes, opts)
		if !strings.Contains(result, "### Sec") {
			t.Errorf("expected ### heading, got:\n%s", result)
		}
	})

	t.Run("out_of_range_falls_back_to_2", func(t *testing.T) {
		opts := DefaultOptions()
		opts.HeadingLevel = 0
		result := Generate(nodes, opts)
		if !strings.Contains(result, "## Sec") {
			t.Errorf("expected ## fallback heading, got:\n%s", result)
		}
	})

	t.Run("above_6_falls_back_to_2", func(t *testing.T) {
		opts := DefaultOptions()
		opts.HeadingLevel = 7
		result := Generate(nodes, opts)
		if !strings.Contains(result, "## Sec") {
			t.Errorf("expected ## fallback heading, got:\n%s", result)
		}
	})
}

func TestGenerate_NoPrettyPrint(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "a", Description: "Short", Default: 1, Section: "S"},
		{Path: "longKeyName", Description: "A longer description here", Default: "value", Section: "S"},
	}

	t.Run("pretty_print_pads_columns", func(t *testing.T) {
		opts := DefaultOptions()
		result := Generate(nodes, opts)
		lines := strings.Split(result, "\n")
		// In pretty-print mode, all data rows should have the same length
		var dataLines []string
		for _, line := range lines {
			if strings.HasPrefix(line, "|") {
				dataLines = append(dataLines, line)
			}
		}
		if len(dataLines) < 3 {
			t.Fatalf("expected at least 3 table lines, got %d", len(dataLines))
		}
		firstLen := len(dataLines[0])
		for i, dl := range dataLines[1:] {
			if len(dl) != firstLen {
				t.Errorf("line %d length %d != header length %d (not aligned)", i+1, len(dl), firstLen)
			}
		}
	})

	t.Run("no_pretty_print_no_padding", func(t *testing.T) {
		opts := DefaultOptions()
		opts.NoPrettyPrint = true
		result := Generate(nodes, opts)
		// Should contain unpadded rows
		assertRowContains(t, result, "`a`", "Short", "`1`")
		// Rows should NOT all be the same length
		lines := strings.Split(result, "\n")
		var dataLines []string
		for _, line := range lines {
			if strings.HasPrefix(line, "|") && !strings.HasPrefix(line, "|--") {
				dataLines = append(dataLines, line)
			}
		}
		if len(dataLines) >= 2 {
			// The short row and long row should differ in length
			if len(dataLines[1]) == len(dataLines[2]) {
				t.Error("expected different row lengths in no-pretty mode")
			}
		}
	})
}

func TestGenerate_TypeColumn(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "name", Description: "App name", Type: "string", Default: "app", Section: "S"},
		{Path: "port", Description: "Port", Type: "integer", Default: 80, Section: "S"},
		{Path: "label", Description: "Label", Type: "string", Nullable: true, Default: nil, Section: "S"},
		{Path: "tags", Description: "Tags", Type: "string[]", Default: []interface{}{}, Section: "S"},
	}

	opts := DefaultOptions()
	opts.TypeColumn = true
	result := Generate(nodes, opts)

	if !strings.Contains(result, "| Type ") {
		t.Errorf("missing Type column header, got:\n%s", result)
	}

	assertRowContains(t, result, "`name`", "`string`", "App name")
	assertRowContains(t, result, "`port`", "`integer`", "Port")
	assertRowContains(t, result, "`label`", "`string?`", "Label")
	assertRowContains(t, result, "`tags`", "`string[]`", "Tags")
}

func TestGenerate_TypeColumnDisabled(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "name", Description: "App name", Type: "string", Default: "app", Section: "S"},
	}

	opts := DefaultOptions()
	result := Generate(nodes, opts)

	if strings.Contains(result, "| Type ") {
		t.Error("Type column should not appear when disabled")
	}
}

func TestGenerate_DeprecatedPrefix(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "old", Description: "Old setting", Type: "boolean", Default: true, Section: "S",
			Deprecated: "Use newSetting instead"},
	}

	result := Generate(nodes, DefaultOptions())
	assertRowContains(t, result, "`old`", "(DEPRECATED) Old setting")
}

func TestGenerate_DefaultOverride(t *testing.T) {
	override := "See values.yaml"
	nodes := []*model.ValueNode{
		{Path: "config", Description: "Config", Type: "object", Default: nil, Section: "S",
			DefaultOverride: &override},
	}

	result := Generate(nodes, DefaultOptions())
	assertRowContains(t, result, "`config`", "`See values.yaml`")
	if strings.Contains(result, "`null`") {
		t.Errorf("default override should replace null display, got:\n%s", result)
	}
}

func TestGenerate_Phase2AnnotationsNotRendered(t *testing.T) {
	min, max := float64(1), float64(100)
	nodes := []*model.ValueNode{
		{Path: "policy", Description: "Pull policy", Type: "string", Default: "Always",
			Enum: []string{"Always", "IfNotPresent", "Never"}, Section: "S"},
		{Path: "port", Description: "Port", Type: "integer", Default: 80,
			Min: &min, Max: &max, Section: "S"},
		{Path: "name", Description: "Name", Type: "string", Default: "app",
			Pattern: "^[a-z]+$", Section: "S"},
		{Path: "display", Description: "Display", Type: "string", Default: "",
			Example: "my-app", Section: "S"},
	}

	result := Generate(nodes, DefaultOptions())

	// These annotations should NOT appear in the README table
	if strings.Contains(result, "Always, IfNotPresent") || strings.Contains(result, "@enum") {
		t.Errorf("enum values should not be rendered in README table")
	}
	if strings.Contains(result, "@min") || strings.Contains(result, "@max") {
		t.Errorf("min/max annotations should not be rendered in README table")
	}
	if strings.Contains(result, "@pattern") || strings.Contains(result, "^[a-z]+$") {
		t.Errorf("pattern should not be rendered in README table")
	}
	if strings.Contains(result, "@example") || strings.Contains(result, "my-app") {
		t.Errorf("example should not be rendered in README table")
	}

	// But the descriptions and defaults should still be there
	assertRowContains(t, result, "`policy`", "Pull policy", "`\"Always\"`")
	assertRowContains(t, result, "`port`", "Port", "`80`")
	assertRowContains(t, result, "`name`", "Name", "`\"app\"`")
	assertRowContains(t, result, "`display`", "Display")
}

func TestInsertIntoReadme_OnlyStartMarker(t *testing.T) {
	existing := "# Chart\n<!-- helm-scribe:start -->\ncontent\n"
	_, err := InsertIntoReadme(existing, "new")
	if err == nil {
		t.Error("expected error when only start marker present")
	}
}

func TestInsertIntoReadme_OnlyEndMarker(t *testing.T) {
	existing := "# Chart\n<!-- helm-scribe:end -->\n"
	_, err := InsertIntoReadme(existing, "new")
	if err == nil {
		t.Error("expected error when only end marker present")
	}
}
