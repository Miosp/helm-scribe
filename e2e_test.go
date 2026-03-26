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
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func TestEndToEnd_Readme(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	readmeData, err := os.ReadFile("testdata/e2e/README.md")
	if err != nil {
		t.Fatal(err)
	}

	nodes, _, err := parser.Parse(data)
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

	// All sections present
	for _, sec := range []string{"Common parameters", "Image parameters", "Network parameters"} {
		if !strings.Contains(result, "## "+sec) {
			t.Errorf("missing section %q", sec)
		}
	}

	// All expected keys present
	expectedKeys := []string{
		"`replicaCount`", "`fullnameOverride`",
		"`image.repository`", "`image.tag`",
		"`service_type`", "`port`", "`debug`", "`cpuLimit`",
		"`optionalDescription`", "`tags`", "`hosts`",
		"`unknownField`",
	}
	for _, key := range expectedKeys {
		if !strings.Contains(result, key) {
			t.Errorf("missing key %s in readme output", key)
		}
	}

	// Skipped key absent
	if strings.Contains(result, "reconcileInterval") {
		t.Error("reconcileInterval should be skipped")
	}

	// Markers and manual content preserved
	if !strings.Contains(result, "<!-- helm-scribe:start -->") {
		t.Error("missing start marker")
	}
	if !strings.Contains(result, "<!-- helm-scribe:end -->") {
		t.Error("missing end marker")
	}
	if !strings.Contains(result, "## Other manual content") {
		t.Error("manual content lost")
	}
}

func TestEndToEnd_SchemaStructure(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, _, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	schemaBytes, err := schema.Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	var s map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if s["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Error("wrong $schema")
	}

	props := s["properties"].(map[string]interface{})

	// Inferred scalar types
	assertPropType(t, props, "replicaCount", "integer")
	assertPropType(t, props, "fullnameOverride", "string")
	assertPropType(t, props, "port", "integer")
	assertPropType(t, props, "debug", "boolean")
	assertPropType(t, props, "cpuLimit", "number")

	// @type override
	assertPropType(t, props, "service_type", "string")

	// Nullable
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

	// @item object array with nested paths
	hosts := props["hosts"].(map[string]interface{})
	if hosts["type"] != "array" {
		t.Errorf("hosts type: got %v", hosts["type"])
	}
	hostItems := hosts["items"].(map[string]interface{})
	if hostItems["type"] != "object" {
		t.Errorf("hosts items type: got %v", hostItems["type"])
	}
	hostProps := hostItems["properties"].(map[string]interface{})
	assertPropType(t, hostProps, "host", "string")
	paths := hostProps["paths"].(map[string]interface{})
	if paths["type"] != "array" {
		t.Errorf("paths type: got %v", paths["type"])
	}

	// Nested object
	img := props["image"].(map[string]interface{})
	if img["type"] != "object" {
		t.Errorf("image type: got %v", img["type"])
	}
	imgProps := img["properties"].(map[string]interface{})
	assertPropType(t, imgProps, "repository", "string")
	assertPropType(t, imgProps, "tag", "string")

	// Untyped null — empty schema (no type field)
	unknown := props["unknownField"].(map[string]interface{})
	if _, hasType := unknown["type"]; hasType {
		t.Errorf("unknownField should have no type, got %v", unknown["type"])
	}

	// Skipped key absent
	if _, ok := props["reconcileInterval"]; ok {
		t.Error("reconcileInterval should be skipped from schema")
	}
}

func TestEndToEnd_SchemaIsValidDraft07(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, _, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	schemaBytes, err := schema.Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	// Compile through a real draft-07 JSON Schema compiler
	sch := compileE2ESchema(t, schemaBytes)

	// Valid Helm values document
	valid := unmarshalE2E(t, `{
		"replicaCount": 3,
		"fullnameOverride": "my-release",
		"image": {"repository": "myapp", "tag": "v1.0"},
		"service_type": "NodePort",
		"port": 8080,
		"debug": true,
		"cpuLimit": 1.5,
		"optionalDescription": null,
		"tags": ["production", "v2"],
		"hosts": [{"host": "example.com", "paths": [{"path": "/api"}]}],
		"unknownField": "anything-goes"
	}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid document rejected: %v", err)
	}
}

func TestEndToEnd_SchemaRejectsInvalidTypes(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, _, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	schemaBytes, err := schema.Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	sch := compileE2ESchema(t, schemaBytes)

	tests := []struct {
		name string
		doc  string
	}{
		{"string where integer expected", `{"replicaCount": "three", "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": []}`},
		{"integer where string expected", `{"replicaCount": 1, "fullnameOverride": 123, "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": []}`},
		{"string where boolean expected", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": "yes", "cpuLimit": 0.5, "tags": [], "hosts": []}`},
		{"integer where number expected", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": "high", "tags": [], "hosts": []}`},
		{"wrong array item type", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [123], "hosts": []}`},
		{"null on non-nullable string", `{"replicaCount": 1, "fullnameOverride": null, "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": []}`},
		{"wrong type in @item host", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [{"host": 999}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := unmarshalE2E(t, tt.doc)
			if err := sch.Validate(doc); err == nil {
				t.Error("expected validation error, got none")
			}
		})
	}
}

func TestEndToEnd_SchemaNullableAcceptsNull(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, _, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	schemaBytes, err := schema.Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	sch := compileE2ESchema(t, schemaBytes)

	base := `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "nginx", "tag": "latest"}, "service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [],`

	// optionalDescription is string? — should accept both string and null
	withString := unmarshalE2E(t, base+`"optionalDescription": "hello"}`)
	if err := sch.Validate(withString); err != nil {
		t.Errorf("string value for nullable field rejected: %v", err)
	}

	withNull := unmarshalE2E(t, base+`"optionalDescription": null}`)
	if err := sch.Validate(withNull); err != nil {
		t.Errorf("null value for nullable field rejected: %v", err)
	}
}

func TestEndToEnd_Warnings(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, warnings, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	hasNullWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "unknownField") && strings.Contains(w, "null") {
			hasNullWarning = true
		}
	}
	if !hasNullWarning {
		t.Errorf("expected null warning for unknownField, got: %v", warnings)
	}
}

func TestEndToEnd_RunFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Copy fixture files
	valuesData, _ := os.ReadFile("testdata/e2e/values.yaml")
	readmeData, _ := os.ReadFile("testdata/e2e/README.md")
	os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644)
	os.WriteFile(tmpDir+"/README.md", readmeData, 0644)

	cfg := config.DefaultConfig()
	valuesPath := tmpDir + "/values.yaml"
	readmePath := tmpDir + "/README.md"
	schemaPath := tmpDir + "/values.schema.json"

	err := run(cfg, valuesPath, readmePath, schemaPath)
	if err != nil {
		t.Fatal(err)
	}

	// Schema file was written and is valid JSON
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("schema file not written: %v", err)
	}
	var s map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &s); err != nil {
		t.Fatalf("schema file is not valid JSON: %v", err)
	}
	if s["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Error("wrong $schema in generated file")
	}

	// README was updated
	updatedReadme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedReadme), "`replicaCount`") {
		t.Error("README not updated with generated content")
	}
}

func TestEndToEnd_RunSchemaOnly(t *testing.T) {
	tmpDir := t.TempDir()

	valuesData, _ := os.ReadFile("testdata/e2e/values.yaml")
	readmeData, _ := os.ReadFile("testdata/e2e/README.md")
	os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644)
	os.WriteFile(tmpDir+"/README.md", readmeData, 0644)

	cfg := config.DefaultConfig()
	cfg.SchemaOnly = true

	err := run(cfg, tmpDir+"/values.yaml", tmpDir+"/README.md", tmpDir+"/values.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	// Schema was written
	if _, err := os.ReadFile(tmpDir + "/values.schema.json"); err != nil {
		t.Error("schema file should exist with --schema-only")
	}

	// README should be unchanged (no markers filled)
	updated, _ := os.ReadFile(tmpDir + "/README.md")
	if strings.Contains(string(updated), "`replicaCount`") {
		t.Error("README should not be updated with --schema-only")
	}
}

func TestEndToEnd_RunReadmeOnly(t *testing.T) {
	tmpDir := t.TempDir()

	valuesData, _ := os.ReadFile("testdata/e2e/values.yaml")
	readmeData, _ := os.ReadFile("testdata/e2e/README.md")
	os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644)
	os.WriteFile(tmpDir+"/README.md", readmeData, 0644)

	cfg := config.DefaultConfig()
	cfg.ReadmeOnly = true

	err := run(cfg, tmpDir+"/values.yaml", tmpDir+"/README.md", tmpDir+"/values.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	// Schema should NOT be written
	if _, err := os.ReadFile(tmpDir + "/values.schema.json"); err == nil {
		t.Error("schema file should not exist with --readme-only")
	}

	// README should be updated
	updated, _ := os.ReadFile(tmpDir + "/README.md")
	if !strings.Contains(string(updated), "`replicaCount`") {
		t.Error("README should be updated with --readme-only")
	}
}

// --- helpers ---

func assertPropType(t *testing.T, props map[string]interface{}, key, expectedType string) {
	t.Helper()
	p, ok := props[key].(map[string]interface{})
	if !ok {
		t.Errorf("property %q missing or wrong type", key)
		return
	}
	if p["type"] != expectedType {
		t.Errorf("%s type: got %v, want %s", key, p["type"], expectedType)
	}
}

func compileE2ESchema(t *testing.T, schemaBytes []byte) *jsonschema.Schema {
	t.Helper()
	var doc interface{}
	if err := json.Unmarshal(schemaBytes, &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", doc); err != nil {
		t.Fatalf("failed to add schema resource: %v", err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		t.Fatalf("schema does not compile as valid JSON Schema: %v", err)
	}
	return sch
}

func unmarshalE2E(t *testing.T, jsonStr string) interface{} {
	t.Helper()
	v, err := jsonschema.UnmarshalJSON(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("invalid JSON doc: %v", err)
	}
	return v
}
