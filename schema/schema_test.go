package schema

import (
	"encoding/json"
	"testing"

	"github.com/miosp/helm-scribe/model"
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
	if _, hasType := p["type"]; hasType {
		t.Errorf("null without @type should not have type field, got %v", p["type"])
	}
}
