package parser

import (
	"strings"

	"github.com/miosp/helm-scribe/model"
)

// Annotations holds the parsed result of a comment block.
type Annotations struct {
	Description  string
	Section      string
	Skip         bool
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
			ann.Section = strings.TrimPrefix(line, "@section ")
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

		if line == "" {
			descParts = append(descParts, "\n")
		} else {
			descParts = append(descParts, line)
		}
	}

	ann.Description = buildDescription(descParts)
	return ann
}

func parseTypeExpr(expr string) (typ string, nullable bool, itemNullable bool) {
	expr = strings.TrimSpace(expr)

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
