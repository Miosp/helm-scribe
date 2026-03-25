package readme

import (
	"errors"
	"fmt"
	"strings"

	"github.com/miosp/helm-scribe/model"
)

const defaultTruncateLength = 80

type Options struct {
	TruncateLength int
}

func DefaultOptions() Options {
	return Options{TruncateLength: defaultTruncateLength}
}

func Generate(nodes []*model.ValueNode, opts Options) string {
	flat := flatten(nodes)
	sections := groupBySection(flat)

	var b strings.Builder
	for _, sec := range sections {
		if sec.name != "" {
			fmt.Fprintf(&b, "## %s\n\n", sec.name)
		}
		b.WriteString("| Key | Description | Default |\n")
		b.WriteString("|-----|-------------|--------|\n")
		for _, n := range sec.nodes {
			def := formatDefault(n.Default, opts.TruncateLength)
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", n.Path, n.Description, def)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func flatten(nodes []*model.ValueNode) []*model.ValueNode {
	var result []*model.ValueNode
	for _, n := range nodes {
		if len(n.Children) > 0 {
			result = append(result, flatten(n.Children)...)
		} else {
			result = append(result, n)
		}
	}
	return result
}

type section struct {
	name  string
	nodes []*model.ValueNode
}

func groupBySection(nodes []*model.ValueNode) []section {
	var sections []section
	seen := map[string]int{}

	for _, n := range nodes {
		name := n.Section
		if idx, ok := seen[name]; ok {
			sections[idx].nodes = append(sections[idx].nodes, n)
		} else {
			seen[name] = len(sections)
			sections = append(sections, section{name: name, nodes: []*model.ValueNode{n}})
		}
	}
	return sections
}

func formatDefault(val interface{}, truncateLen int) string {
	if val == nil {
		return "`null`"
	}
	s := fmt.Sprintf("%v", val)

	switch v := val.(type) {
	case string:
		s = fmt.Sprintf(`"%s"`, v)
	case []interface{}:
		if len(v) == 0 {
			s = "[]"
		} else {
			return "See values.yaml"
		}
	}

	if truncateLen > 0 && len(s) > truncateLen {
		return "See values.yaml"
	}
	return fmt.Sprintf("`%s`", s)
}

const (
	markerStart = "<!-- helm-scribe:start -->"
	markerEnd   = "<!-- helm-scribe:end -->"
)

func InsertIntoReadme(existing, content string) (string, error) {
	startIdx := strings.Index(existing, markerStart)
	endIdx := strings.Index(existing, markerEnd)

	if startIdx == -1 || endIdx == -1 {
		return "", errors.New("helm-scribe markers not found in README; add <!-- helm-scribe:start --> and <!-- helm-scribe:end --> markers")
	}

	var b strings.Builder
	b.WriteString(existing[:startIdx])
	b.WriteString(markerStart)
	b.WriteByte('\n')
	b.WriteString(content)
	b.WriteString(markerEnd)
	b.WriteString(existing[endIdx+len(markerEnd):])

	return b.String(), nil
}
