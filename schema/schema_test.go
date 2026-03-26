package schema

import (
	"encoding/json"
	"testing"

	"github.com/miosp/helm-scribe/model"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func mustUnmarshal(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func prop(t *testing.T, schema map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	props := schema["properties"].(map[string]interface{})
	return props[key].(map[string]interface{})
}

func TestGenerate_Scalars(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "nginx", Description: "App name"},
		{Key: "port", Path: "port", Type: "integer", Default: 80, Description: "Port number"},
		{Key: "debug", Path: "debug", Type: "boolean", Default: false},
		{Key: "ratio", Path: "ratio", Type: "number", Default: 0.5},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	if schema["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Error("wrong $schema")
	}
	if schema["type"] != "object" {
		t.Error("wrong top-level type")
	}

	n := prop(t, schema, "name")
	if n["type"] != "string" {
		t.Errorf("name type: got %v", n["type"])
	}
	if n["description"] != "App name" {
		t.Errorf("name description: got %v", n["description"])
	}
	if n["default"] != "nginx" {
		t.Errorf("name default: got %v", n["default"])
	}

	p := prop(t, schema, "port")
	if p["default"] != float64(80) {
		t.Errorf("port default: got %v (%T)", p["default"], p["default"])
	}
}

func TestGenerate_NestedObject(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "image", Path: "image", Type: "object",
			Children: []*model.ValueNode{
				{Key: "repository", Path: "image.repository", Type: "string", Default: "nginx"},
				{Key: "tag", Path: "image.tag", Type: "string", Default: "latest"},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	img := prop(t, schema, "image")
	if img["type"] != "object" {
		t.Errorf("image type: got %v", img["type"])
	}

	imgProps := img["properties"].(map[string]interface{})
	repo := imgProps["repository"].(map[string]interface{})
	if repo["type"] != "string" {
		t.Errorf("repository type: got %v", repo["type"])
	}
}

func TestGenerate_Required(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "app"},
		{Key: "label", Path: "label", Type: "string", Default: nil, Nullable: true},
		{Key: "extra", Path: "extra", Type: "null", Default: nil},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	req, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("required missing")
	}
	if len(req) != 1 || req[0] != "name" {
		t.Errorf("required: got %v, want [name]", req)
	}
}

func TestGenerate_Nullable(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "label", Path: "label", Type: "string", Nullable: true, Default: "x"},
	}
	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}
	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "label")
	typeVal, ok := p["type"].([]interface{})
	if !ok {
		t.Fatalf("expected type array, got %T: %v", p["type"], p["type"])
	}
	if len(typeVal) != 2 || typeVal[0] != "string" || typeVal[1] != "null" {
		t.Errorf("type: got %v, want [string null]", typeVal)
	}
}

func TestGenerate_ArrayType(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "tags", Path: "tags", Type: "string[]", Default: []interface{}{}},
	}
	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}
	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "tags")
	if p["type"] != "array" {
		t.Errorf("type: got %v", p["type"])
	}
	items := p["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("items type: got %v", items["type"])
	}
}

func TestGenerate_NullableArray(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "tags", Path: "tags", Type: "string[]", Nullable: true, Default: []interface{}{}},
	}
	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}
	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "tags")
	typeVal := p["type"].([]interface{})
	if len(typeVal) != 2 || typeVal[0] != "array" || typeVal[1] != "null" {
		t.Errorf("type: got %v, want [array null]", typeVal)
	}
}

func TestGenerate_ObjectArrayWithItems(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "hosts", Path: "hosts", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "host", Type: "string"},
				{Path: "paths", Type: "object[]"},
				{Path: "paths[].path", Type: "string"},
				{Path: "paths[].pathType", Type: "string"},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	hosts := prop(t, schema, "hosts")
	if hosts["type"] != "array" {
		t.Errorf("type: got %v", hosts["type"])
	}

	items := hosts["items"].(map[string]interface{})
	if items["type"] != "object" {
		t.Errorf("items type: got %v", items["type"])
	}

	itemProps := items["properties"].(map[string]interface{})

	host := itemProps["host"].(map[string]interface{})
	if host["type"] != "string" {
		t.Errorf("host type: got %v", host["type"])
	}

	paths := itemProps["paths"].(map[string]interface{})
	if paths["type"] != "array" {
		t.Errorf("paths type: got %v", paths["type"])
	}

	pathItems := paths["items"].(map[string]interface{})
	pathProps := pathItems["properties"].(map[string]interface{})
	pathProp := pathProps["path"].(map[string]interface{})
	if pathProp["type"] != "string" {
		t.Errorf("paths[].path type: got %v", pathProp["type"])
	}
}

func TestGenerate_NullableItemType(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "entries", Path: "entries", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "name", Type: "string"},
				{Path: "label", Type: "string?"},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	entries := prop(t, schema, "entries")
	items := entries["items"].(map[string]interface{})
	itemProps := items["properties"].(map[string]interface{})

	name := itemProps["name"].(map[string]interface{})
	if name["type"] != "string" {
		t.Errorf("name type: got %v", name["type"])
	}

	label := itemProps["label"].(map[string]interface{})
	typeArr, ok := label["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "string" || typeArr[1] != "null" {
		t.Errorf("label type: expected [string null], got %v", label["type"])
	}
}

func TestGenerate_NullWithoutType(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "unknown", Path: "unknown", Type: "null", Default: nil},
	}
	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "unknown")

	// Should only contain "default": null — no "type" field
	if _, hasType := p["type"]; hasType {
		t.Errorf("null without @type should not have type field, got %v", p["type"])
	}
	if len(p) != 1 {
		t.Errorf("expected exactly 1 field (default), got %d: %v", len(p), p)
	}
}

// compileSchema compiles generated schema bytes into a jsonschema.Schema
// that can validate documents. Fails the test if the schema is not valid draft-07.
func compileSchema(t *testing.T, schemaBytes []byte) *jsonschema.Schema {
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

func unmarshalDoc(t *testing.T, jsonStr string) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		t.Fatalf("invalid JSON doc: %v", err)
	}
	return v
}

func TestGenerate_ValidDraft07(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "nginx", Description: "App name"},
		{Key: "port", Path: "port", Type: "integer", Default: 80},
		{Key: "debug", Path: "debug", Type: "boolean", Default: false},
		{Key: "label", Path: "label", Type: "string", Nullable: true, Default: nil},
		{Key: "tags", Path: "tags", Type: "string[]", Default: []interface{}{}},
		{
			Key: "image", Path: "image", Type: "object",
			Children: []*model.ValueNode{
				{Key: "repository", Path: "image.repository", Type: "string", Default: "nginx"},
				{Key: "tag", Path: "image.tag", Type: "string", Default: "latest"},
			},
		},
		{
			Key: "hosts", Path: "hosts", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "host", Type: "string"},
				{Path: "port", Type: "integer"},
			},
		},
		{Key: "unknown", Path: "unknown", Type: "null", Default: nil},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	// This will fail if the schema is not valid draft-07
	compileSchema(t, data)
}

func TestGenerate_SchemaValidatesDocuments(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "nginx"},
		{Key: "port", Path: "port", Type: "integer", Default: 80},
		{Key: "debug", Path: "debug", Type: "boolean", Default: false},
		{Key: "label", Path: "label", Type: "string", Nullable: true, Default: nil},
		{Key: "tags", Path: "tags", Type: "string[]", Default: []interface{}{}},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	sch := compileSchema(t, data)

	// Valid document
	valid := unmarshalDoc(t, `{"name": "app", "port": 8080, "debug": true, "label": null, "tags": ["v1"]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid document rejected: %v", err)
	}

	// Wrong type for port (string instead of integer)
	wrongType := unmarshalDoc(t, `{"name": "app", "port": "not-a-number", "debug": false, "tags": []}`)
	if err := sch.Validate(wrongType); err == nil {
		t.Error("expected validation error for wrong port type, got none")
	}

	// Wrong type in array items (integer instead of string)
	wrongItems := unmarshalDoc(t, `{"name": "app", "port": 80, "debug": false, "tags": [123]}`)
	if err := sch.Validate(wrongItems); err == nil {
		t.Error("expected validation error for wrong array item type, got none")
	}

	// Nullable field accepts null
	withNull := unmarshalDoc(t, `{"name": "app", "port": 80, "debug": false, "label": null, "tags": []}`)
	if err := sch.Validate(withNull); err != nil {
		t.Errorf("nullable field should accept null: %v", err)
	}

	// Non-nullable field rejects null
	nullName := unmarshalDoc(t, `{"name": null, "port": 80, "debug": false, "tags": []}`)
	if err := sch.Validate(nullName); err == nil {
		t.Error("expected validation error for null on non-nullable field, got none")
	}
}

func TestGenerate_ObjectArraySchemaValidates(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "hosts", Path: "hosts", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "host", Type: "string"},
				{Path: "port", Type: "integer"},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"hosts": [{"host": "example.com", "port": 443}]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid hosts rejected: %v", err)
	}

	wrongItemType := unmarshalDoc(t, `{"hosts": [{"host": 123, "port": 443}]}`)
	if err := sch.Validate(wrongItemType); err == nil {
		t.Error("expected validation error for wrong host type")
	}
}
