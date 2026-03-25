package parser

import (
	"fmt"
	"strconv"

	"github.com/miosp/helm-scribe/model"
	"gopkg.in/yaml.v3"
)

// Parse parses raw YAML bytes into a flat list of ValueNodes.
func Parse(data []byte) ([]*model.ValueNode, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("expected document node")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected top-level mapping")
	}

	return walkMapping(root, ""), nil
}

func walkMapping(node *yaml.Node, prefix string) []*model.ValueNode {
	var nodes []*model.ValueNode
	currentSection := ""

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		ann := ParseAnnotations(keyNode.HeadComment)

		if ann.Section != "" {
			currentSection = ann.Section
		}

		if ann.Skip {
			continue
		}

		path := keyNode.Value
		if prefix != "" {
			path = prefix + "." + keyNode.Value
		}

		n := &model.ValueNode{
			Key:         keyNode.Value,
			Path:        path,
			Description: ann.Description,
			Section:     currentSection,
		}

		switch valNode.Kind {
		case yaml.MappingNode:
			n.Type = "object"
			n.Default = nil
			n.Children = walkMapping(valNode, path)
			for _, child := range n.Children {
				if child.Section == "" {
					child.Section = currentSection
				}
			}
		case yaml.SequenceNode:
			n.Type = "array"
			n.Default = decodeSequence(valNode)
		case yaml.ScalarNode:
			n.Type = inferScalarType(valNode)
			n.Default = decodeScalar(valNode)
		default:
			n.Type = "string"
			n.Default = valNode.Value
		}

		nodes = append(nodes, n)
	}

	return nodes
}

func inferScalarType(node *yaml.Node) string {
	if node.Tag == "!!null" || node.Value == "null" || node.Value == "~" {
		return "null"
	}
	if node.Tag == "!!bool" {
		return "boolean"
	}
	if node.Tag == "!!int" {
		return "integer"
	}
	if node.Tag == "!!float" {
		return "number"
	}
	return "string"
}

func decodeScalar(node *yaml.Node) interface{} {
	if node.Tag == "!!null" || node.Value == "null" || node.Value == "~" {
		return nil
	}
	if node.Tag == "!!bool" {
		v, _ := strconv.ParseBool(node.Value)
		return v
	}
	if node.Tag == "!!int" {
		v, _ := strconv.ParseInt(node.Value, 10, 64)
		return int(v)
	}
	if node.Tag == "!!float" {
		v, _ := strconv.ParseFloat(node.Value, 64)
		return v
	}
	return node.Value
}

func decodeSequence(node *yaml.Node) interface{} {
	if len(node.Content) == 0 {
		return []interface{}{}
	}
	var items []interface{}
	for _, item := range node.Content {
		items = append(items, decodeScalar(item))
	}
	return items
}
