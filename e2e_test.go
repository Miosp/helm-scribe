package main

import (
	"os"
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/config"
	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/readme"
)

func TestEndToEnd(t *testing.T) {
	valuesData, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	readmeData, err := os.ReadFile("testdata/e2e/README.md")
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := parser.Parse(valuesData)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	opts := readme.Options{TruncateLength: cfg.TruncateLength}
	table := readme.Generate(nodes, opts)

	result, err := readme.InsertIntoReadme(string(readmeData), table)
	if err != nil {
		t.Fatal(err)
	}

	// Verify sections present
	if !strings.Contains(result, "## Common parameters") {
		t.Error("missing Common parameters section")
	}
	if !strings.Contains(result, "## Image parameters") {
		t.Error("missing Image parameters section")
	}

	// Verify keys present
	if !strings.Contains(result, "| `replicaCount` |") {
		t.Error("missing replicaCount")
	}
	if !strings.Contains(result, "| `image.repository` |") {
		t.Error("missing image.repository")
	}
	if !strings.Contains(result, "| `image.tag` |") {
		t.Error("missing image.tag")
	}

	// Verify skipped key absent
	if strings.Contains(result, "reconcileInterval") {
		t.Error("reconcileInterval should be skipped")
	}

	// Verify markers preserved
	if !strings.Contains(result, "<!-- helm-scribe:start -->") {
		t.Error("missing start marker")
	}
	if !strings.Contains(result, "<!-- helm-scribe:end -->") {
		t.Error("missing end marker")
	}

	// Verify manual content preserved
	if !strings.Contains(result, "## Other manual content") {
		t.Error("manual content lost")
	}
}
