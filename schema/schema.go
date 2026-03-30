package schema

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/miosp/helm-scribe/model"
)

func Generate(nodes []*model.ValueNode) ([]byte, error) {
	props, req := buildPropertiesWithRequired(nodes)
	schema := map[string]any{
		"$schema":    "https://json-schema.org/draft-07/schema#",
		"type":       "object",
		"properties": props,
	}
	if len(req) > 0 {
		schema["required"] = req
	}
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func buildPropertiesWithRequired(nodes []*model.ValueNode) (map[string]any, []string) {
	props := make(map[string]any)
	var req []string
	for _, n := range nodes {
		schema, hasRequired := nodeSchema(n)
		props[n.Key] = schema
		if hasRequired {
			req = append(req, n.Key)
		}
	}
	return props, req
}

// nodeSchema returns the JSON Schema for a single node and whether the node
// has required content (non-null default or required descendants).
func nodeSchema(n *model.ValueNode) (map[string]any, bool) {
	s := make(map[string]any)

	if n.Description != "" {
		s["description"] = n.Description
	}

	baseType := n.Type
	isArray := strings.HasSuffix(baseType, "[]")
	if isArray {
		baseType = strings.TrimSuffix(baseType, "[]")
	}

	hasRequired := n.Default != nil

	switch {
	// Untyped null value (inferred from nil YAML value, not from ? suffix)
	case n.Type == "null" && !n.Nullable:
		if n.Default == nil {
			s["default"] = nil
		}
		return s, false

	case len(n.Children) > 0:
		setType(s, "object", n.Nullable)
		props, req := buildPropertiesWithRequired(n.Children)
		s["properties"] = props
		if len(req) > 0 {
			s["required"] = req
			hasRequired = true
		}

	case isArray && len(n.Items) > 0:
		setType(s, "array", n.Nullable)
		s["items"] = buildItemSchema(n.Items)

	case isArray:
		setType(s, "array", n.Nullable)
		if baseType != "object" {
			itemSchema := make(map[string]any)
			setType(itemSchema, baseType, n.ItemNullable)
			s["items"] = itemSchema
		}

	default:
		setType(s, baseType, n.Nullable)
	}

	if len(n.Enum) > 0 {
		s["enum"] = convertEnum(n.Enum, baseType)
	}
	if n.Min != nil {
		s["minimum"] = *n.Min
	}
	if n.Max != nil {
		s["maximum"] = *n.Max
	}
	if n.Pattern != "" {
		s["pattern"] = n.Pattern
	}
	if n.Deprecated != "" {
		s["deprecated"] = true
	}
	if n.Example != "" {
		s["examples"] = []any{convertValue(n.Example, baseType)}
	}

	if n.Default != nil {
		s["default"] = n.Default
	}

	return s, hasRequired && !n.Nullable
}

func setType(s map[string]any, typ string, nullable bool) {
	if nullable {
		s["type"] = []string{typ, "null"}
	} else {
		s["type"] = typ
	}
}

func convertEnum(values []string, baseType string) []any {
	result := make([]any, len(values))
	for i, v := range values {
		result[i] = convertValue(v, baseType)
	}
	return result
}

func convertValue(val, baseType string) any {
	switch baseType {
	case "integer":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	case "number":
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	case "boolean":
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return val
}

func buildItemSchema(items []*model.ItemDef) map[string]any {
	result := map[string]any{
		"type": "object",
	}

	type propInfo struct {
		typ      string
		children []*model.ItemDef
	}

	props := make(map[string]*propInfo)
	var order []string

	for _, item := range items {
		top, rest := splitItemPath(item.Path)

		if rest == "" {
			if _, exists := props[top]; !exists {
				order = append(order, top)
				props[top] = &propInfo{typ: item.Type}
			} else {
				props[top].typ = item.Type
			}
		} else {
			if _, exists := props[top]; !exists {
				order = append(order, top)
				props[top] = &propInfo{typ: "object[]"}
			}
			props[top].children = append(props[top].children, &model.ItemDef{
				Path: rest,
				Type: item.Type,
			})
		}
	}

	properties := make(map[string]any)
	for _, name := range order {
		info := props[name]
		if len(info.children) > 0 {
			properties[name] = map[string]any{
				"type":  "array",
				"items": buildItemSchema(info.children),
			}
		} else {
			baseType, nullable, isArray, itemNullable := parseItemType(info.typ)
			p := make(map[string]any)
			if isArray {
				setType(p, "array", nullable)
				if baseType != "object" {
					itemSchema := make(map[string]any)
					setType(itemSchema, baseType, itemNullable)
					p["items"] = itemSchema
				}
			} else {
				setType(p, baseType, nullable)
			}
			properties[name] = p
		}
	}

	result["properties"] = properties
	return result
}

func parseItemType(expr string) (baseType string, nullable bool, isArray bool, itemNullable bool) {
	// Outer nullable: trailing ? (after any [])
	if strings.HasSuffix(expr, "?") {
		nullable = true
		expr = strings.TrimSuffix(expr, "?")
	}
	// Array: trailing []
	isArray = strings.HasSuffix(expr, "[]")
	if isArray {
		expr = strings.TrimSuffix(expr, "[]")
	}
	// Item nullable: ? before [] (e.g. string?[] -> items are nullable)
	if isArray && strings.HasSuffix(expr, "?") {
		itemNullable = true
		expr = strings.TrimSuffix(expr, "?")
	}
	return expr, nullable, isArray, itemNullable
}

// splitItemPath splits an @item path into top-level key and remainder.
// Note: keys containing literal dots are not supported.
func splitItemPath(path string) (top, rest string) {
	if idx := strings.Index(path, "[]."); idx != -1 {
		return path[:idx], path[idx+3:]
	}
	if idx := strings.Index(path, "."); idx != -1 {
		return path[:idx], path[idx+1:]
	}
	return path, ""
}
