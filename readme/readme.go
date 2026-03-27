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
	HeadingLevel   int
	NoPrettyPrint  bool
	TypeColumn     bool
}

func DefaultOptions() Options {
	return Options{TruncateLength: defaultTruncateLength, HeadingLevel: 2}
}

type tableRow struct {
	key, typ, description, def string
}

func Generate(nodes []*model.ValueNode, opts Options) string {
	flat := flatten(nodes)
	sections := groupBySection(flat)

	var b strings.Builder
	for _, sec := range sections {
		if sec.name != "" {
			level := opts.HeadingLevel
			if level < 1 || level > 6 {
				level = 2
			}
			fmt.Fprintf(&b, "%s %s\n\n", strings.Repeat("#", level), sec.name)
		}

		var rows []tableRow
		for _, n := range sec.nodes {
			desc := n.Description
			if n.Deprecated != "" {
				desc = "(DEPRECATED) " + desc
			}
			defStr := formatDefault(n.Default, opts.TruncateLength)
			if n.DefaultOverride != nil {
				defStr = fmt.Sprintf("`%s`", *n.DefaultOverride)
			}
			typStr := ""
			if opts.TypeColumn {
				typStr = fmt.Sprintf("`%s`", n.Type)
				if n.Nullable {
					typStr = fmt.Sprintf("`%s?`", n.Type)
				}
			}
			rows = append(rows, tableRow{
				key:         fmt.Sprintf("`%s`", n.Path),
				typ:         typStr,
				description: desc,
				def:         defStr,
			})
		}

		writeTable(&b, rows, opts)
		b.WriteByte('\n')
	}

	return b.String()
}

func writeTable(b *strings.Builder, rows []tableRow, opts Options) {
	hasType := opts.TypeColumn
	headers := tableRow{key: "Key", typ: "Type", description: "Description", def: "Default"}

	if opts.NoPrettyPrint {
		if hasType {
			fmt.Fprintf(b, "| %s | %s | %s | %s |\n", headers.key, headers.typ, headers.description, headers.def)
			b.WriteString("|-----|------|-------------|--------|\n")
			for _, r := range rows {
				fmt.Fprintf(b, "| %s | %s | %s | %s |\n", r.key, r.typ, r.description, r.def)
			}
		} else {
			fmt.Fprintf(b, "| %s | %s | %s |\n", headers.key, headers.description, headers.def)
			b.WriteString("|-----|-------------|--------|\n")
			for _, r := range rows {
				fmt.Fprintf(b, "| %s | %s | %s |\n", r.key, r.description, r.def)
			}
		}
		return
	}

	kw, tw, dw, fw := len(headers.key), len(headers.typ), len(headers.description), len(headers.def)
	for _, r := range rows {
		if len(r.key) > kw {
			kw = len(r.key)
		}
		if len(r.typ) > tw {
			tw = len(r.typ)
		}
		if len(r.description) > dw {
			dw = len(r.description)
		}
		if len(r.def) > fw {
			fw = len(r.def)
		}
	}

	if hasType {
		fmtStr := fmt.Sprintf("| %%-%ds | %%-%ds | %%-%ds | %%-%ds |\n", kw, tw, dw, fw)
		fmt.Fprintf(b, fmtStr, headers.key, headers.typ, headers.description, headers.def)
		fmt.Fprintf(b, "|-%s-|-%s-|-%s-|-%s-|\n", strings.Repeat("-", kw), strings.Repeat("-", tw), strings.Repeat("-", dw), strings.Repeat("-", fw))
		for _, r := range rows {
			fmt.Fprintf(b, fmtStr, r.key, r.typ, r.description, r.def)
		}
	} else {
		fmtStr := fmt.Sprintf("| %%-%ds | %%-%ds | %%-%ds |\n", kw, dw, fw)
		fmt.Fprintf(b, fmtStr, headers.key, headers.description, headers.def)
		fmt.Fprintf(b, "|-%s-|-%s-|-%s-|\n", strings.Repeat("-", kw), strings.Repeat("-", dw), strings.Repeat("-", fw))
		for _, r := range rows {
			fmt.Fprintf(b, fmtStr, r.key, r.description, r.def)
		}
	}
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
