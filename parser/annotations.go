package parser

import (
	"strconv"
	"strings"

	"github.com/miosp/helm-scribe/model"
)

// Annotations holds the parsed result of a comment block.
type Annotations struct {
	Description string
	Skip        bool
	Type         string
	Nullable     bool
	ItemNullable bool
	Items           []*model.ItemDef
	Enum            []string
	Min             *float64
	Max             *float64
	Deprecated      string
	DefaultOverride *string
	Example         string
	Pattern         string
}

// ParseAnnotations extracts description text and tags from a raw HeadComment.
func ParseAnnotations(raw string) Annotations {
	var ann Annotations
	if raw == "" {
		return ann
	}

	var descParts []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "@section ") {
			continue
		}
		if line == "@skip" {
			ann.Skip = true
			continue
		}
		if strings.HasPrefix(line, "@type ") {
			ann.Type, ann.Nullable, ann.ItemNullable = parseTypeExpr(strings.TrimPrefix(line, "@type "))
			continue
		}
		if strings.HasPrefix(line, "@item ") {
			if item, ok := parseItemDef(strings.TrimPrefix(line, "@item ")); ok {
				ann.Items = append(ann.Items, item)
			}
			continue
		}
		if strings.HasPrefix(line, "@enum ") {
			ann.Enum = parseEnum(strings.TrimPrefix(line, "@enum "))
			continue
		}
		if strings.HasPrefix(line, "@min ") {
			if v, err := strconv.ParseFloat(strings.TrimPrefix(line, "@min "), 64); err == nil {
				ann.Min = &v
			}
			continue
		}
		if strings.HasPrefix(line, "@max ") {
			if v, err := strconv.ParseFloat(strings.TrimPrefix(line, "@max "), 64); err == nil {
				ann.Max = &v
			}
			continue
		}
		if strings.HasPrefix(line, "@default ") {
			val := strings.TrimPrefix(line, "@default ")
			ann.DefaultOverride = &val
			continue
		}
		if line == "@deprecated" {
			ann.Deprecated = "deprecated"
			continue
		}
		if strings.HasPrefix(line, "@deprecated ") {
			ann.Deprecated = strings.TrimPrefix(line, "@deprecated ")
			continue
		}
		if strings.HasPrefix(line, "@example ") {
			ann.Example = strings.TrimPrefix(line, "@example ")
			continue
		}
		if strings.HasPrefix(line, "@pattern ") {
			ann.Pattern = strings.TrimPrefix(line, "@pattern ")
			continue
		}

		if line == "" {
			descParts = append(descParts, "\n")
		} else {
			descParts = append(descParts, line)
		}
	}

	ann.Description = buildDescription(descParts)
	return ann
}

func parseTypeExpr(raw string) (typ string, nullable bool, itemNullable bool) {
	expr := strings.TrimSpace(raw)

	// Outer nullable: trailing ? (after any [])
	if strings.HasSuffix(expr, "?") {
		nullable = true
		expr = strings.TrimSuffix(expr, "?")
	}

	// Array: trailing []
	isArray := strings.HasSuffix(expr, "[]")
	if isArray {
		expr = strings.TrimSuffix(expr, "[]")
	}

	// Item nullable: ? before [] (e.g. string?[] -> string, itemNullable=true)
	if isArray && strings.HasSuffix(expr, "?") {
		itemNullable = true
		expr = strings.TrimSuffix(expr, "?")
	}

	// Empty base type means the expression was nonsensical (e.g. "?", "[]", "??").
	// Return the raw input so validateType can warn about it.
	if expr == "" {
		return strings.TrimSpace(raw), false, false
	}
	if isArray {
		return expr + "[]", nullable, itemNullable
	}
	return expr, nullable, itemNullable
}

func parseItemDef(raw string) (*model.ItemDef, bool) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return nil, false
	}
	return &model.ItemDef{
		Path: strings.TrimSpace(parts[0]),
		Type: strings.TrimSpace(parts[1]),
	}, true
}

// parseEnum parses a bracketed, comma-separated list like [a, b, c].
// Limitation: values containing commas are not supported, even when quoted.
// For example, ["a,b", "c"] is incorrectly split into 3 values instead of 2.
func parseEnum(raw string) []string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil
	}
	raw = raw[1 : len(raw)-1]
	var values []string
	for _, v := range strings.Split(raw, ",") {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}

func buildDescription(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	var result strings.Builder
	prevWasBreak := false
	first := true
	for _, p := range parts {
		if p == "\n" {
			prevWasBreak = true
			continue
		}
		if !first {
			if prevWasBreak {
				result.WriteByte('\n')
			} else {
				result.WriteByte(' ')
			}
		}
		result.WriteString(p)
		prevWasBreak = false
		first = false
	}
	return result.String()
}
