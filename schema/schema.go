package schema

import (
	"encoding/json"
	"strings"

	"github.com/miosp/helm-scribe/model"
)

func Generate(nodes []*model.ValueNode) ([]byte, error) {
	schema := map[string]interface{}{
		"$schema":    "https://json-schema.org/draft-07/schema#",
		"type":       "object",
		"properties": buildProperties(nodes),
	}
	if req := requiredKeys(nodes); len(req) > 0 {
		schema["required"] = req
	}
	return json.MarshalIndent(schema, "", "  ")
}

func buildProperties(nodes []*model.ValueNode) map[string]interface{} {
	props := make(map[string]interface{})
	for _, n := range nodes {
		props[n.Key] = nodeSchema(n)
	}
	return props
}

func nodeSchema(n *model.ValueNode) map[string]interface{} {
	s := make(map[string]interface{})

	if n.Description != "" {
		s["description"] = n.Description
	}

	baseType := n.Type
	isArray := strings.HasSuffix(baseType, "[]")
	if isArray {
		baseType = strings.TrimSuffix(baseType, "[]")
	}

	switch {
	case n.Type == "null" && !n.Nullable:
		if n.Default == nil {
			s["default"] = nil
		}
		return s

	case len(n.Children) > 0:
		setType(s, "object", n.Nullable)
		s["properties"] = buildProperties(n.Children)
		if req := requiredKeys(n.Children); len(req) > 0 {
			s["required"] = req
		}

	case isArray && len(n.Items) > 0:
		setType(s, "array", n.Nullable)
		s["items"] = buildItemSchema(n.Items)

	case isArray:
		setType(s, "array", n.Nullable)
		if baseType != "object" {
			s["items"] = map[string]interface{}{"type": baseType}
		}

	default:
		setType(s, baseType, n.Nullable)
	}

	if n.Default != nil {
		s["default"] = n.Default
	}

	return s
}

func setType(s map[string]interface{}, typ string, nullable bool) {
	if nullable {
		s["type"] = []string{typ, "null"}
	} else {
		s["type"] = typ
	}
}

func requiredKeys(nodes []*model.ValueNode) []string {
	var req []string
	for _, n := range nodes {
		if n.Default != nil && !n.Nullable {
			req = append(req, n.Key)
		}
	}
	return req
}

func buildItemSchema(items []*model.ItemDef) map[string]interface{} {
	result := map[string]interface{}{
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

	properties := make(map[string]interface{})
	for _, name := range order {
		info := props[name]
		if len(info.children) > 0 {
			properties[name] = map[string]interface{}{
				"type":  "array",
				"items": buildItemSchema(info.children),
			}
		} else {
			baseType := info.typ
			isArr := strings.HasSuffix(baseType, "[]")
			if isArr {
				baseType = strings.TrimSuffix(baseType, "[]")
				properties[name] = map[string]interface{}{"type": "array"}
			} else {
				properties[name] = map[string]interface{}{"type": baseType}
			}
		}
	}

	result["properties"] = properties
	return result
}

func splitItemPath(path string) (top, rest string) {
	if idx := strings.Index(path, "[]."); idx != -1 {
		return path[:idx], path[idx+3:]
	}
	if idx := strings.Index(path, "."); idx != -1 {
		return path[:idx], path[idx+1:]
	}
	return path, ""
}
