package readme

import (
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/model"
)

func TestGenerate_BasicTable(t *testing.T) {
	nodes := []*model.ValueNode{
		{Path: "replicaCount", Description: "Number of replicas", Type: "integer", Default: 1, Section: "Common"},
		{Path: "fullnameOverride", Description: "Override full name", Type: "string", Default: "", Section: "Common"},
	}

	result := Generate(nodes, DefaultOptions())

	if !strings.Contains(result, "## Common") {
		t.Error("missing section header")
	}
	if !strings.Contains(result, "| `replicaCount` | Number of replicas | `1` |") {
		t.Errorf("missing replicaCount row, got:\n%s", result)
	}
	if !strings.Contains(result, `| `+"`"+`fullnameOverride`+"`"+` | Override full name | `+"`"+`""`+"`"+` |`) {
		t.Errorf("missing fullnameOverride row, got:\n%s", result)
	}
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

	if !strings.Contains(result, "| `image.repository` |") {
		t.Errorf("missing flattened child, got:\n%s", result)
	}
	if !strings.Contains(result, "| `image.tag` |") {
		t.Errorf("missing flattened child, got:\n%s", result)
	}
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
	if !strings.Contains(result, "| `key` |") {
		t.Errorf("missing row, got:\n%s", result)
	}
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
