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

func TestGenerate_ArrayOfNullableItems(t *testing.T) {
	// string?[] -> array of (string | null)
	nodes := []*model.ValueNode{
		{Key: "names", Path: "names", Type: "string[]", ItemNullable: true, Default: []interface{}{}},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify schema structure
	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "names")
	if p["type"] != "array" {
		t.Fatalf("type: got %v, want array", p["type"])
	}
	items := p["items"].(map[string]interface{})
	typeArr, ok := items["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "string" || typeArr[1] != "null" {
		t.Errorf("items type: expected [string null], got %v", items["type"])
	}

	// Validate with draft-07
	sch := compileSchema(t, data)

	// Array with strings — valid
	valid := unmarshalDoc(t, `{"names": ["alice", "bob"]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("string items rejected: %v", err)
	}

	// Array with null items — valid (items are nullable)
	withNulls := unmarshalDoc(t, `{"names": ["alice", null, "bob"]}`)
	if err := sch.Validate(withNulls); err != nil {
		t.Errorf("null items rejected: %v", err)
	}

	// Array itself as null — invalid (array is not nullable)
	nullArray := unmarshalDoc(t, `{"names": null}`)
	if err := sch.Validate(nullArray); err == nil {
		t.Error("null array should be rejected (array not nullable)")
	}

	// Wrong item type — invalid
	wrongItem := unmarshalDoc(t, `{"names": [123]}`)
	if err := sch.Validate(wrongItem); err == nil {
		t.Error("integer item should be rejected")
	}
}

func TestGenerate_NullableItemsAllTypes(t *testing.T) {
	tests := []struct {
		name        string
		baseType    string
		validItem   string
		invalidItem string
	}{
		{"integer", "integer", "1", `"text"`},
		{"number", "number", "1.5", `"text"`},
		{"boolean", "boolean", "true", `"text"`},
		{"string", "string", `"hello"`, "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"?[] array of nullable", func(t *testing.T) {
			nodes := []*model.ValueNode{
				{Key: "val", Path: "val", Type: tt.baseType + "[]", ItemNullable: true, Default: []interface{}{}},
			}
			data, err := Generate(nodes)
			if err != nil {
				t.Fatal(err)
			}

			// Structure: items type should be [<baseType>, null]
			schema := mustUnmarshal(t, data)
			p := prop(t, schema, "val")
			if p["type"] != "array" {
				t.Fatalf("outer type: got %v, want array", p["type"])
			}
			items := p["items"].(map[string]interface{})
			typeArr, ok := items["type"].([]interface{})
			if !ok || len(typeArr) != 2 || typeArr[0] != tt.baseType || typeArr[1] != "null" {
				t.Fatalf("items type: expected [%s null], got %v", tt.baseType, items["type"])
			}

			sch := compileSchema(t, data)

			valid := unmarshalDoc(t, `{"val": [`+tt.validItem+`]}`)
			if err := sch.Validate(valid); err != nil {
				t.Errorf("valid item rejected: %v", err)
			}

			withNull := unmarshalDoc(t, `{"val": [`+tt.validItem+`, null]}`)
			if err := sch.Validate(withNull); err != nil {
				t.Errorf("null item rejected: %v", err)
			}

			invalid := unmarshalDoc(t, `{"val": [`+tt.invalidItem+`]}`)
			if err := sch.Validate(invalid); err == nil {
				t.Errorf("invalid item type should be rejected")
			}
		})

		t.Run(tt.name+"?[]? nullable array of nullable", func(t *testing.T) {
			nodes := []*model.ValueNode{
				{Key: "val", Path: "val", Type: tt.baseType + "[]", Nullable: true, ItemNullable: true, Default: []interface{}{}},
			}
			data, err := Generate(nodes)
			if err != nil {
				t.Fatal(err)
			}

			schema := mustUnmarshal(t, data)
			p := prop(t, schema, "val")
			outerType, ok := p["type"].([]interface{})
			if !ok || len(outerType) != 2 || outerType[0] != "array" || outerType[1] != "null" {
				t.Fatalf("outer type: expected [array null], got %v", p["type"])
			}
			items := p["items"].(map[string]interface{})
			itemType, ok := items["type"].([]interface{})
			if !ok || len(itemType) != 2 || itemType[0] != tt.baseType || itemType[1] != "null" {
				t.Fatalf("items type: expected [%s null], got %v", tt.baseType, items["type"])
			}

			sch := compileSchema(t, data)

			valid := unmarshalDoc(t, `{"val": [`+tt.validItem+`]}`)
			if err := sch.Validate(valid); err != nil {
				t.Errorf("valid item rejected: %v", err)
			}

			withNull := unmarshalDoc(t, `{"val": [`+tt.validItem+`, null]}`)
			if err := sch.Validate(withNull); err != nil {
				t.Errorf("null item rejected: %v", err)
			}

			nullArray := unmarshalDoc(t, `{"val": null}`)
			if err := sch.Validate(nullArray); err != nil {
				t.Errorf("null array rejected: %v", err)
			}

			invalid := unmarshalDoc(t, `{"val": [`+tt.invalidItem+`]}`)
			if err := sch.Validate(invalid); err == nil {
				t.Errorf("invalid item type should be rejected")
			}
		})
	}
}

func TestGenerate_NullableArrayOfNullableItems(t *testing.T) {
	// string?[]? -> (array | null) of (string | null)
	nodes := []*model.ValueNode{
		{Key: "names", Path: "names", Type: "string[]", Nullable: true, ItemNullable: true, Default: []interface{}{}},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify schema structure
	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "names")

	// Outer type should be [array, null]
	outerType, ok := p["type"].([]interface{})
	if !ok || len(outerType) != 2 || outerType[0] != "array" || outerType[1] != "null" {
		t.Errorf("outer type: expected [array null], got %v", p["type"])
	}

	// Items type should be [string, null]
	items := p["items"].(map[string]interface{})
	itemType, ok := items["type"].([]interface{})
	if !ok || len(itemType) != 2 || itemType[0] != "string" || itemType[1] != "null" {
		t.Errorf("items type: expected [string null], got %v", items["type"])
	}

	// Validate with draft-07
	sch := compileSchema(t, data)

	// Array with strings — valid
	valid := unmarshalDoc(t, `{"names": ["alice", "bob"]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("string items rejected: %v", err)
	}

	// Array with null items — valid
	withNulls := unmarshalDoc(t, `{"names": ["alice", null]}`)
	if err := sch.Validate(withNulls); err != nil {
		t.Errorf("null items rejected: %v", err)
	}

	// Null array — valid (array itself is nullable)
	nullArray := unmarshalDoc(t, `{"names": null}`)
	if err := sch.Validate(nullArray); err != nil {
		t.Errorf("null array rejected: %v", err)
	}

	// Wrong item type — invalid
	wrongItem := unmarshalDoc(t, `{"names": [123]}`)
	if err := sch.Validate(wrongItem); err == nil {
		t.Error("integer item should be rejected")
	}
}

func TestGenerate_ObjectWithAllNullChildrenNotRequired(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "app"},
		{
			Key: "config", Path: "config", Type: "object",
			Children: []*model.ValueNode{
				{Key: "desc", Path: "config.desc", Type: "null", Default: nil},
				{Key: "label", Path: "config.label", Type: "null", Default: nil},
			},
		},
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
	// config should NOT be required — all its children are null/non-required
	if len(req) != 1 || req[0] != "name" {
		t.Errorf("required: got %v, want [name]", req)
	}
}

func TestGenerate_DeeplyNestedAllNullNotRequired(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "top", Path: "top", Type: "object",
			Children: []*model.ValueNode{
				{
					Key: "mid", Path: "top.mid", Type: "object",
					Children: []*model.ValueNode{
						{Key: "bottom", Path: "top.mid.bottom", Type: "null", Default: nil},
					},
				},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	// No property at any level should be required
	if _, ok := schema["required"]; ok {
		t.Errorf("top-level required should be absent, got %v", schema["required"])
	}

	top := prop(t, schema, "top")
	if _, ok := top["required"]; ok {
		t.Errorf("top.required should be absent, got %v", top["required"])
	}

	topProps := top["properties"].(map[string]interface{})
	mid := topProps["mid"].(map[string]interface{})
	if _, ok := mid["required"]; ok {
		t.Errorf("mid.required should be absent, got %v", mid["required"])
	}
}

func TestGenerate_ObjectWithMixedChildrenRequired(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "config", Path: "config", Type: "object",
			Children: []*model.ValueNode{
				{Key: "name", Path: "config.name", Type: "string", Default: "app"},
				{Key: "label", Path: "config.label", Type: "null", Default: nil},
			},
		},
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
	if len(req) != 1 || req[0] != "config" {
		t.Errorf("required: got %v, want [config]", req)
	}
}

func TestGenerate_NullableObject(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "service", Path: "service", Type: "object", Nullable: true,
			Children: []*model.ValueNode{
				{Key: "type", Path: "service.type", Type: "string", Default: "ClusterIP"},
				{Key: "port", Path: "service.port", Type: "integer", Default: 80},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "service")
	typeArr, ok := p["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "object" || typeArr[1] != "null" {
		t.Fatalf("type: expected [object null], got %v", p["type"])
	}
	if _, ok := p["properties"]; !ok {
		t.Fatal("missing properties on nullable object")
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"service": {"type": "NodePort", "port": 443}}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid object rejected: %v", err)
	}

	nullObj := unmarshalDoc(t, `{"service": null}`)
	if err := sch.Validate(nullObj); err != nil {
		t.Errorf("null object rejected: %v", err)
	}

	wrongProp := unmarshalDoc(t, `{"service": {"type": "ClusterIP", "port": "not-a-number"}}`)
	if err := sch.Validate(wrongProp); err == nil {
		t.Error("wrong property type should be rejected")
	}

	wrongObjType := unmarshalDoc(t, `{"service": "not-an-object"}`)
	if err := sch.Validate(wrongObjType); err == nil {
		t.Error("string instead of object should be rejected")
	}
}

func TestGenerate_ObjectWithNullableProperties(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "config", Path: "config", Type: "object",
			Children: []*model.ValueNode{
				{Key: "name", Path: "config.name", Type: "string", Default: "app"},
				{Key: "description", Path: "config.description", Type: "string", Nullable: true, Default: nil},
				{Key: "replicas", Path: "config.replicas", Type: "integer", Nullable: true, Default: nil},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	cfg := prop(t, schema, "config")
	if cfg["type"] != "object" {
		t.Fatalf("config type: got %v", cfg["type"])
	}
	cfgProps := cfg["properties"].(map[string]interface{})

	name := cfgProps["name"].(map[string]interface{})
	if name["type"] != "string" {
		t.Errorf("name type: got %v", name["type"])
	}

	desc := cfgProps["description"].(map[string]interface{})
	descType, ok := desc["type"].([]interface{})
	if !ok || len(descType) != 2 || descType[0] != "string" || descType[1] != "null" {
		t.Errorf("description type: expected [string null], got %v", desc["type"])
	}

	rep := cfgProps["replicas"].(map[string]interface{})
	repType, ok := rep["type"].([]interface{})
	if !ok || len(repType) != 2 || repType[0] != "integer" || repType[1] != "null" {
		t.Errorf("replicas type: expected [integer null], got %v", rep["type"])
	}

	req, ok := cfg["required"].([]interface{})
	if !ok || len(req) != 1 || req[0] != "name" {
		t.Errorf("required: expected [name], got %v", req)
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"config": {"name": "myapp", "description": "A service", "replicas": 3}}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}

	withNulls := unmarshalDoc(t, `{"config": {"name": "myapp", "description": null, "replicas": null}}`)
	if err := sch.Validate(withNulls); err != nil {
		t.Errorf("null nullable properties rejected: %v", err)
	}

	nullName := unmarshalDoc(t, `{"config": {"name": null}}`)
	if err := sch.Validate(nullName); err == nil {
		t.Error("null non-nullable name should be rejected")
	}

	wrongDescType := unmarshalDoc(t, `{"config": {"name": "myapp", "description": 123}}`)
	if err := sch.Validate(wrongDescType); err == nil {
		t.Error("integer for nullable string should be rejected")
	}
}

func TestGenerate_NullableNestedObjects(t *testing.T) {
	nodes := []*model.ValueNode{
		{
			Key: "outer", Path: "outer", Type: "object",
			Children: []*model.ValueNode{
				{
					Key: "middle", Path: "outer.middle", Type: "object", Nullable: true,
					Children: []*model.ValueNode{
						{Key: "leaf", Path: "outer.middle.leaf", Type: "string", Default: "val"},
						{Key: "optLeaf", Path: "outer.middle.optLeaf", Type: "integer", Nullable: true, Default: nil},
					},
				},
				{Key: "sibling", Path: "outer.sibling", Type: "string", Default: "hi"},
			},
		},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	outer := prop(t, schema, "outer")
	outerProps := outer["properties"].(map[string]interface{})
	middle := outerProps["middle"].(map[string]interface{})
	midType, ok := middle["type"].([]interface{})
	if !ok || len(midType) != 2 || midType[0] != "object" || midType[1] != "null" {
		t.Fatalf("middle type: expected [object null], got %v", middle["type"])
	}
	midProps := middle["properties"].(map[string]interface{})
	optLeaf := midProps["optLeaf"].(map[string]interface{})
	optLeafType, ok := optLeaf["type"].([]interface{})
	if !ok || len(optLeafType) != 2 || optLeafType[0] != "integer" || optLeafType[1] != "null" {
		t.Errorf("optLeaf type: expected [integer null], got %v", optLeaf["type"])
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"outer": {"middle": {"leaf": "x", "optLeaf": 5}, "sibling": "hi"}}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("fully populated rejected: %v", err)
	}

	nullMiddle := unmarshalDoc(t, `{"outer": {"middle": null, "sibling": "hi"}}`)
	if err := sch.Validate(nullMiddle); err != nil {
		t.Errorf("null middle rejected: %v", err)
	}

	nullOptLeaf := unmarshalDoc(t, `{"outer": {"middle": {"leaf": "x", "optLeaf": null}, "sibling": "hi"}}`)
	if err := sch.Validate(nullOptLeaf); err != nil {
		t.Errorf("null optLeaf rejected: %v", err)
	}

	nullOuter := unmarshalDoc(t, `{"outer": null}`)
	if err := sch.Validate(nullOuter); err == nil {
		t.Error("null non-nullable outer should be rejected")
	}

	nullRequiredLeaf := unmarshalDoc(t, `{"outer": {"middle": {"leaf": null, "optLeaf": 1}, "sibling": "hi"}}`)
	if err := sch.Validate(nullRequiredLeaf); err == nil {
		t.Error("null non-nullable leaf should be rejected")
	}

	wrongDeep := unmarshalDoc(t, `{"outer": {"middle": {"leaf": "x", "optLeaf": "not-int"}, "sibling": "hi"}}`)
	if err := sch.Validate(wrongDeep); err == nil {
		t.Error("wrong type for optLeaf should be rejected")
	}
}

func TestGenerate_ItemWithNullableArrayType(t *testing.T) {
	// @item values: string?[] should produce array of nullable strings
	nodes := []*model.ValueNode{
		{
			Key: "entries", Path: "entries", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "name", Type: "string"},
				{Path: "values", Type: "string?[]"},
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

	vals := itemProps["values"].(map[string]interface{})
	if vals["type"] != "array" {
		t.Fatalf("values type: got %v, want array", vals["type"])
	}
	valItems, ok := vals["items"].(map[string]interface{})
	if !ok {
		t.Fatal("values should have items constraint")
	}
	typeArr, ok := valItems["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "string" || typeArr[1] != "null" {
		t.Errorf("values items type: expected [string null], got %v", valItems["type"])
	}

	// Also validate with draft-07
	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"entries": [{"name": "a", "values": ["x", null]}]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid doc rejected: %v", err)
	}

	invalid := unmarshalDoc(t, `{"entries": [{"name": "a", "values": [123]}]}`)
	if err := sch.Validate(invalid); err == nil {
		t.Error("integer item in string?[] should be rejected")
	}
}

func TestGenerate_ItemWithNullableArray(t *testing.T) {
	// @item values: string[]? should produce nullable array of strings
	nodes := []*model.ValueNode{
		{
			Key: "entries", Path: "entries", Type: "object[]",
			Default: []interface{}{},
			Items: []*model.ItemDef{
				{Path: "name", Type: "string"},
				{Path: "tags", Type: "string[]?"},
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

	tags := itemProps["tags"].(map[string]interface{})
	typeArr, ok := tags["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "array" || typeArr[1] != "null" {
		t.Fatalf("tags type: expected [array null], got %v", tags["type"])
	}
	tagItems, ok := tags["items"].(map[string]interface{})
	if !ok {
		t.Fatal("tags should have items constraint")
	}
	if tagItems["type"] != "string" {
		t.Errorf("tags items type: expected string, got %v", tagItems["type"])
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"entries": [{"name": "a", "tags": null}]}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("null array rejected: %v", err)
	}
}

func TestGenerate_Enum(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "policy", Path: "policy", Type: "string", Default: "Always",
			Enum: []string{"Always", "IfNotPresent", "Never"}},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "policy")
	enumVal, ok := p["enum"].([]interface{})
	if !ok {
		t.Fatalf("enum missing, got %v", p["enum"])
	}
	if len(enumVal) != 3 || enumVal[0] != "Always" {
		t.Errorf("enum: got %v", enumVal)
	}

	sch := compileSchema(t, data)
	valid := unmarshalDoc(t, `{"policy": "IfNotPresent"}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid enum value rejected: %v", err)
	}

	invalid := unmarshalDoc(t, `{"policy": "Unknown"}`)
	if err := sch.Validate(invalid); err == nil {
		t.Error("invalid enum value should be rejected")
	}
}

func TestGenerate_EnumInteger(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "level", Path: "level", Type: "integer", Default: 1,
			Enum: []string{"1", "2", "3"}},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "level")
	enumVal := p["enum"].([]interface{})
	if enumVal[0] != float64(1) {
		t.Errorf("enum[0]: got %v (%T), want numeric 1", enumVal[0], enumVal[0])
	}

	sch := compileSchema(t, data)
	valid := unmarshalDoc(t, `{"level": 2}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid enum rejected: %v", err)
	}
	invalid := unmarshalDoc(t, `{"level": 5}`)
	if err := sch.Validate(invalid); err == nil {
		t.Error("out-of-enum value should be rejected")
	}
}

func TestGenerate_MinMax(t *testing.T) {
	min, max := float64(1), float64(65535)
	nodes := []*model.ValueNode{
		{Key: "port", Path: "port", Type: "integer", Default: 80,
			Min: &min, Max: &max},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "port")
	if p["minimum"] != float64(1) {
		t.Errorf("minimum: got %v", p["minimum"])
	}
	if p["maximum"] != float64(65535) {
		t.Errorf("maximum: got %v", p["maximum"])
	}

	sch := compileSchema(t, data)
	valid := unmarshalDoc(t, `{"port": 8080}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid port rejected: %v", err)
	}
	tooLow := unmarshalDoc(t, `{"port": 0}`)
	if err := sch.Validate(tooLow); err == nil {
		t.Error("port below minimum should be rejected")
	}
	tooHigh := unmarshalDoc(t, `{"port": 70000}`)
	if err := sch.Validate(tooHigh); err == nil {
		t.Error("port above maximum should be rejected")
	}
}

func TestGenerate_Pattern(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "my-app",
			Pattern: "^[a-z][a-z0-9-]*$"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "name")
	if p["pattern"] != "^[a-z][a-z0-9-]*$" {
		t.Errorf("pattern: got %v", p["pattern"])
	}

	sch := compileSchema(t, data)
	valid := unmarshalDoc(t, `{"name": "my-app"}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid name rejected: %v", err)
	}
	invalid := unmarshalDoc(t, `{"name": "My_App!"}`)
	if err := sch.Validate(invalid); err == nil {
		t.Error("invalid name should be rejected")
	}
}

func TestGenerate_Deprecated(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "old", Path: "old", Type: "boolean", Default: true,
			Deprecated: "Use newSetting instead"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "old")
	if p["deprecated"] != true {
		t.Errorf("deprecated: got %v", p["deprecated"])
	}

	compileSchema(t, data)
}

func TestGenerate_Example(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "",
			Example: "my-custom-app"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "name")
	examples, ok := p["examples"].([]interface{})
	if !ok || len(examples) != 1 || examples[0] != "my-custom-app" {
		t.Errorf("examples: got %v", p["examples"])
	}

	compileSchema(t, data)
}

func TestGenerate_ExampleInteger(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "port", Path: "port", Type: "integer", Default: 80,
			Example: "8080"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "port")
	examples := p["examples"].([]interface{})
	if examples[0] != float64(8080) {
		t.Errorf("example: got %v (%T), want numeric 8080", examples[0], examples[0])
	}
}

func TestGenerate_DeprecatedBare(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "old", Path: "old", Type: "boolean", Default: true,
			Deprecated: "deprecated"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "old")
	if p["deprecated"] != true {
		t.Errorf("deprecated: got %v, want true", p["deprecated"])
	}

	compileSchema(t, data)
}

func TestGenerate_MinMaxOnStringType(t *testing.T) {
	min, max := float64(1), float64(100)
	nodes := []*model.ValueNode{
		{Key: "name", Path: "name", Type: "string", Default: "app",
			Min: &min, Max: &max},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "name")
	if p["type"] != "string" {
		t.Errorf("type: got %v", p["type"])
	}
	// minimum/maximum are emitted even on string — semantically odd but valid schema
	if p["minimum"] != float64(1) {
		t.Errorf("minimum: got %v", p["minimum"])
	}
	if p["maximum"] != float64(100) {
		t.Errorf("maximum: got %v", p["maximum"])
	}

	compileSchema(t, data)
}

func TestGenerate_PatternOnNonStringType(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "port", Path: "port", Type: "integer", Default: 80,
			Pattern: "^[0-9]+$"},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "port")
	if p["type"] != "integer" {
		t.Errorf("type: got %v", p["type"])
	}
	// pattern is emitted even on integer — semantically odd but valid schema
	if p["pattern"] != "^[0-9]+$" {
		t.Errorf("pattern: got %v", p["pattern"])
	}

	compileSchema(t, data)
}

func TestGenerate_EnumTypeConversionFailure(t *testing.T) {
	t.Run("non-numeric strings on integer type", func(t *testing.T) {
		nodes := []*model.ValueNode{
			{Key: "level", Path: "level", Type: "integer", Default: 1,
				Enum: []string{"low", "medium", "high"}},
		}

		data, err := Generate(nodes)
		if err != nil {
			t.Fatal(err)
		}

		schema := mustUnmarshal(t, data)
		p := prop(t, schema, "level")
		enumVal := p["enum"].([]interface{})
		// Conversion fails, so raw strings are emitted
		if enumVal[0] != "low" {
			t.Errorf("enum[0]: got %v (%T), want string 'low'", enumVal[0], enumVal[0])
		}

		compileSchema(t, data)
	})

	t.Run("non-numeric strings on number type", func(t *testing.T) {
		nodes := []*model.ValueNode{
			{Key: "ratio", Path: "ratio", Type: "number", Default: 0.5,
				Enum: []string{"small", "large"}},
		}

		data, err := Generate(nodes)
		if err != nil {
			t.Fatal(err)
		}

		schema := mustUnmarshal(t, data)
		p := prop(t, schema, "ratio")
		enumVal := p["enum"].([]interface{})
		if enumVal[0] != "small" {
			t.Errorf("enum[0]: got %v (%T), want string 'small'", enumVal[0], enumVal[0])
		}

		compileSchema(t, data)
	})

	t.Run("non-boolean strings on boolean type", func(t *testing.T) {
		nodes := []*model.ValueNode{
			{Key: "flag", Path: "flag", Type: "boolean", Default: true,
				Enum: []string{"yes", "no"}},
		}

		data, err := Generate(nodes)
		if err != nil {
			t.Fatal(err)
		}

		schema := mustUnmarshal(t, data)
		p := prop(t, schema, "flag")
		enumVal := p["enum"].([]interface{})
		// "yes"/"no" don't parse as booleans, so raw strings are emitted
		if enumVal[0] != "yes" {
			t.Errorf("enum[0]: got %v (%T), want string 'yes'", enumVal[0], enumVal[0])
		}

		compileSchema(t, data)
	})
}

func TestGenerate_NullableObjectNoChildren(t *testing.T) {
	nodes := []*model.ValueNode{
		{Key: "extra", Path: "extra", Type: "object", Nullable: true, Default: nil},
	}

	data, err := Generate(nodes)
	if err != nil {
		t.Fatal(err)
	}

	schema := mustUnmarshal(t, data)
	p := prop(t, schema, "extra")
	typeArr, ok := p["type"].([]interface{})
	if !ok || len(typeArr) != 2 || typeArr[0] != "object" || typeArr[1] != "null" {
		t.Fatalf("type: expected [object null], got %v", p["type"])
	}

	sch := compileSchema(t, data)

	valid := unmarshalDoc(t, `{"extra": {"any": "thing"}}`)
	if err := sch.Validate(valid); err != nil {
		t.Errorf("valid object rejected: %v", err)
	}

	nullVal := unmarshalDoc(t, `{"extra": null}`)
	if err := sch.Validate(nullVal); err != nil {
		t.Errorf("null rejected: %v", err)
	}

	wrongObjType := unmarshalDoc(t, `{"extra": "not-object"}`)
	if err := sch.Validate(wrongObjType); err == nil {
		t.Error("string for nullable object should be rejected")
	}
}
