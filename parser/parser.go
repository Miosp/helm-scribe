package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/miosp/helm-scribe/model"
	"gopkg.in/yaml.v3"
)

type sectionMarker struct {
	line int
	name string
}

func extractSectionMarkers(data []byte) []sectionMarker {
	var markers []sectionMarker
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@section ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "@section "))
			markers = append(markers, sectionMarker{line: lineNum, name: name})
		}
	}
	return markers
}

func findSection(markers []sectionMarker, afterLine, atOrBeforeLine int) string {
	result := ""
	for _, m := range markers {
		if m.line > afterLine && m.line <= atOrBeforeLine {
			result = m.name
		}
	}
	return result
}

// Parse parses raw YAML bytes into a flat list of ValueNodes.
func Parse(data []byte) ([]*model.ValueNode, []string, error) {
	markers := extractSectionMarkers(data)

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("parsing yaml: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil, fmt.Errorf("expected document node")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("expected top-level mapping")
	}

	var warnings []string
	nodes := walkMapping(root, "", markers, &warnings)
	return nodes, warnings, nil
}

func walkMapping(node *yaml.Node, prefix string, markers []sectionMarker, warnings *[]string) []*model.ValueNode {
	var nodes []*model.ValueNode
	currentSection := ""
	prevLine := 0

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		if sec := findSection(markers, prevLine, keyNode.Line); sec != "" {
			currentSection = sec
		}
		prevLine = keyNode.Line

		ann := ParseAnnotations(keyNode.HeadComment)

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
			n.Children = walkMapping(valNode, path, markers, warnings)
			propagateSection(n.Children, currentSection)
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

		if ann.Type != "" {
			n.Type = ann.Type
			n.Nullable = ann.Nullable
		}
		if len(ann.Items) > 0 {
			n.Items = ann.Items
			if ann.Type == "" {
				n.Type = "object[]"
			}
		}
		if n.Type == "null" && ann.Type == "" {
			*warnings = append(*warnings, fmt.Sprintf("key %q is null with no @type; schema will accept any value", n.Path))
		}
		if n.Type == "array" && ann.Type == "" && len(ann.Items) == 0 {
			*warnings = append(*warnings, fmt.Sprintf("key %q is an array with no @type; schema will not validate items", n.Path))
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
		switch item.Kind {
		case yaml.MappingNode:
			// Non-scalar items — store raw string representation
			var v interface{}
			_ = item.Decode(&v)
			items = append(items, v)
		default:
			items = append(items, decodeScalar(item))
		}
	}
	return items
}

func propagateSection(nodes []*model.ValueNode, section string) {
	for _, n := range nodes {
		if n.Section == "" {
			n.Section = section
		}
		if len(n.Children) > 0 {
			propagateSection(n.Children, n.Section)
		}
	}
}
