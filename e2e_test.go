package main

import (
	"encoding/json"
	"errors"
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

	var s map[string]any
	if err := json.Unmarshal(schemaBytes, &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if s["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Error("wrong $schema")
	}

	props := s["properties"].(map[string]any)

	// Inferred scalar types
	assertPropType(t, props, "replicaCount", "integer")
	assertPropType(t, props, "fullnameOverride", "string")
	assertPropType(t, props, "port", "integer")
	assertPropType(t, props, "debug", "boolean")
	assertPropType(t, props, "cpuLimit", "number")

	// @type override
	assertPropType(t, props, "service_type", "string")

	// Nullable
	od := props["optionalDescription"].(map[string]any)
	typeArr, ok := od["type"].([]any)
	if !ok || len(typeArr) != 2 || typeArr[0] != "string" || typeArr[1] != "null" {
		t.Errorf("optionalDescription type: expected [string null], got %v", od["type"])
	}

	// string[]
	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" {
		t.Errorf("tags type: got %v", tags["type"])
	}
	tagItems := tags["items"].(map[string]any)
	if tagItems["type"] != "string" {
		t.Errorf("tags items type: got %v", tagItems["type"])
	}

	// @item object array with nested paths
	hosts := props["hosts"].(map[string]any)
	if hosts["type"] != "array" {
		t.Errorf("hosts type: got %v", hosts["type"])
	}
	hostItems := hosts["items"].(map[string]any)
	if hostItems["type"] != "object" {
		t.Errorf("hosts items type: got %v", hostItems["type"])
	}
	hostProps := hostItems["properties"].(map[string]any)
	assertPropType(t, hostProps, "host", "string")
	paths := hostProps["paths"].(map[string]any)
	if paths["type"] != "array" {
		t.Errorf("paths type: got %v", paths["type"])
	}

	// Nested object
	img := props["image"].(map[string]any)
	if img["type"] != "object" {
		t.Errorf("image type: got %v", img["type"])
	}
	imgProps := img["properties"].(map[string]any)
	assertPropType(t, imgProps, "repository", "string")
	assertPropType(t, imgProps, "tag", "string")

	// Untyped null — empty schema (no type field)
	unknown := props["unknownField"].(map[string]any)
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
		"unknownField": "anything-goes",
		"pullPolicy": "Always", "validatedPort": 8080, "appName": "my-app",
		"oldSetting": true, "displayName": "hello", "extraConfig": {}
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
		{"string where integer expected", `{"replicaCount": "three", "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"integer where string expected", `{"replicaCount": 1, "fullnameOverride": 123, "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"string where boolean expected", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": "yes", "cpuLimit": 0.5, "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"integer where number expected", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": "high", "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"wrong array item type", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [123], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"null on non-nullable string", `{"replicaCount": 1, "fullnameOverride": null, "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
		{"wrong type in @item host", `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"}, "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [{"host": 999}], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {}}`},
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

	base := `{"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "nginx", "tag": "latest"}, "service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5, "tags": [], "hosts": [], "pullPolicy": "Always", "validatedPort": 80, "appName": "app", "oldSetting": true, "displayName": "", "extraConfig": {},`

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
	if err := os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/README.md", readmeData, 0644); err != nil {
		t.Fatal(err)
	}

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
	var s map[string]any
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
	if err := os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/README.md", readmeData, 0644); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/README.md", readmeData, 0644); err != nil {
		t.Fatal(err)
	}

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

func TestStrictWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	valuesData, _ := os.ReadFile("testdata/e2e/values.yaml")
	readmeData, _ := os.ReadFile("testdata/e2e/README.md")
	if err := os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/README.md", readmeData, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Strict = true

	err := run(cfg, tmpDir+"/values.yaml", tmpDir+"/README.md", tmpDir+"/values.schema.json")

	var we *WarningsError
	if !errors.As(err, &we) {
		t.Fatalf("expected WarningsError, got: %v", err)
	}
	if we.Count < 1 {
		t.Errorf("expected at least 1 warning, got %d", we.Count)
	}

	// Files should still be written despite strict warnings
	if _, err := os.ReadFile(tmpDir + "/values.schema.json"); err != nil {
		t.Error("schema file should still be written in strict mode")
	}
}

func TestStrictWithoutWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/values.yaml", []byte("# Name\nname: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Strict = true
	cfg.SchemaOnly = true

	err := run(cfg, tmpDir+"/values.yaml", tmpDir+"/README.md", tmpDir+"/values.schema.json")
	if err != nil {
		t.Fatalf("expected no error with strict + no warnings, got: %v", err)
	}
}

func TestNonStrictWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	valuesData, _ := os.ReadFile("testdata/e2e/values.yaml")
	readmeData, _ := os.ReadFile("testdata/e2e/README.md")
	if err := os.WriteFile(tmpDir+"/values.yaml", valuesData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/README.md", readmeData, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	// cfg.Strict is false by default

	err := run(cfg, tmpDir+"/values.yaml", tmpDir+"/README.md", tmpDir+"/values.schema.json")
	if err != nil {
		t.Fatalf("expected nil error without strict mode, got: %v", err)
	}
}

func TestEndToEnd_Phase2Schema(t *testing.T) {
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

	var s map[string]any
	if err := json.Unmarshal(schemaBytes, &s); err != nil {
		t.Fatal(err)
	}

	props := s["properties"].(map[string]any)

	// @enum
	pp := props["pullPolicy"].(map[string]any)
	enumVal, ok := pp["enum"].([]any)
	if !ok || len(enumVal) != 3 {
		t.Errorf("pullPolicy enum: got %v", pp["enum"])
	}

	// @min/@max
	vp := props["validatedPort"].(map[string]any)
	if vp["minimum"] != float64(1) {
		t.Errorf("validatedPort minimum: got %v", vp["minimum"])
	}
	if vp["maximum"] != float64(65535) {
		t.Errorf("validatedPort maximum: got %v", vp["maximum"])
	}

	// @pattern
	an := props["appName"].(map[string]any)
	if an["pattern"] != "^[a-z][a-z0-9-]*$" {
		t.Errorf("appName pattern: got %v", an["pattern"])
	}

	// @deprecated
	os_ := props["oldSetting"].(map[string]any)
	if os_["deprecated"] != true {
		t.Errorf("oldSetting deprecated: got %v", os_["deprecated"])
	}

	// @example
	dn := props["displayName"].(map[string]any)
	examples, ok := dn["examples"].([]any)
	if !ok || len(examples) != 1 || examples[0] != "my-custom-app" {
		t.Errorf("displayName examples: got %v", dn["examples"])
	}

	compileE2ESchema(t, schemaBytes)
}

func TestEndToEnd_Phase2SchemaValidation(t *testing.T) {
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

	valid := unmarshalE2E(t, `{
		"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"},
		"service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5,
		"optionalDescription": null, "tags": [], "hosts": [], "unknownField": "x",
		"pullPolicy": "Always", "validatedPort": 8080, "appName": "my-app",
		"oldSetting": true, "displayName": "hello", "extraConfig": {}
	}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid doc rejected: %v", err)
	}

	badEnum := unmarshalE2E(t, `{
		"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"},
		"service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5,
		"tags": [], "hosts": [],
		"pullPolicy": "InvalidPolicy", "validatedPort": 80, "appName": "app"
	}`)
	if err := sch.Validate(badEnum); err == nil {
		t.Error("invalid enum value should be rejected")
	}

	badPort := unmarshalE2E(t, `{
		"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"},
		"service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5,
		"tags": [], "hosts": [],
		"pullPolicy": "Always", "validatedPort": 0, "appName": "app"
	}`)
	if err := sch.Validate(badPort); err == nil {
		t.Error("port below minimum should be rejected")
	}

	badName := unmarshalE2E(t, `{
		"replicaCount": 1, "fullnameOverride": "", "image": {"repository": "x", "tag": "x"},
		"service_type": "ClusterIP", "port": 80, "debug": false, "cpuLimit": 0.5,
		"tags": [], "hosts": [],
		"pullPolicy": "Always", "validatedPort": 80, "appName": "INVALID_NAME!"
	}`)
	if err := sch.Validate(badName); err == nil {
		t.Error("name violating pattern should be rejected")
	}
}

func TestEndToEnd_Phase2Readme(t *testing.T) {
	data, err := os.ReadFile("testdata/e2e/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	nodes, _, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	opts := readme.Options{TruncateLength: 80}
	table := readme.Generate(nodes, opts)

	if !strings.Contains(table, "## Validation parameters") {
		t.Error("missing Validation parameters section")
	}

	if !strings.Contains(table, "(DEPRECATED)") {
		t.Error("missing DEPRECATED prefix for oldSetting")
	}

	if !strings.Contains(table, "`See values.yaml`") {
		t.Error("missing default override for extraConfig")
	}
}

// --- helpers ---

func assertPropType(t *testing.T, props map[string]any, key, expectedType string) {
	t.Helper()
	p, ok := props[key].(map[string]any)
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
	var doc any
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

func unmarshalE2E(t *testing.T, jsonStr string) any {
	t.Helper()
	v, err := jsonschema.UnmarshalJSON(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("invalid JSON doc: %v", err)
	}
	return v
}
