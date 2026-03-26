package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/miosp/helm-scribe/config"
	"github.com/miosp/helm-scribe/parser"
	"github.com/miosp/helm-scribe/readme"
	"github.com/miosp/helm-scribe/schema"
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

	nodes, _, err := parser.Parse(valuesData)
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

	// Verify keys present (padded in pretty-print mode)
	for _, key := range []string{"`replicaCount`", "`image.repository`", "`image.tag`"} {
		if !strings.Contains(result, key) {
			t.Errorf("missing %s", key)
		}
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

func TestEndToEnd_Schema(t *testing.T) {
	valuesData, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	nodes, _, err := parser.Parse(valuesData)
	if err != nil {
		t.Fatal(err)
	}

	data, err := schema.Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	var s map[string]interface{}
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if s["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Error("wrong $schema")
	}

	props := s["properties"].(map[string]interface{})

	// @type override
	st := props["service_type"].(map[string]interface{})
	if st["type"] != "string" {
		t.Errorf("service_type type: got %v", st["type"])
	}

	// nullable
	od := props["optionalDescription"].(map[string]interface{})
	typeArr, ok := od["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "string" || typeArr[1] != "null" {
		t.Errorf("optionalDescription type: expected [string null], got %v", od["type"])
	}

	// string[]
	tags := props["tags"].(map[string]interface{})
	if tags["type"] != "array" {
		t.Errorf("tags type: got %v", tags["type"])
	}
	tagItems := tags["items"].(map[string]interface{})
	if tagItems["type"] != "string" {
		t.Errorf("tags items type: got %v", tagItems["type"])
	}

	// @item object array
	hosts := props["hosts"].(map[string]interface{})
	if hosts["type"] != "array" {
		t.Errorf("hosts type: got %v", hosts["type"])
	}
	hostItems := hosts["items"].(map[string]interface{})
	if hostItems["type"] != "object" {
		t.Errorf("hosts items type: got %v", hostItems["type"])
	}
	hostProps := hostItems["properties"].(map[string]interface{})
	host := hostProps["host"].(map[string]interface{})
	if host["type"] != "string" {
		t.Errorf("host type: got %v", host["type"])
	}

	// nested object should have children in schema
	img := props["image"].(map[string]interface{})
	if img["type"] != "object" {
		t.Errorf("image type: got %v", img["type"])
	}
	imgProps := img["properties"].(map[string]interface{})
	if _, ok := imgProps["repository"]; !ok {
		t.Error("missing image.repository in schema")
	}

	// skipped key should be absent
	if _, ok := props["reconcileInterval"]; ok {
		t.Error("reconcileInterval should be skipped from schema")
	}
}
