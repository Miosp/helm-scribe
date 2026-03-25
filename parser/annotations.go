package parser

import "strings"

// Annotations holds the parsed result of a comment block.
type Annotations struct {
	Description string
	Section     string
	Skip        bool
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

		if line == "" {
			descParts = append(descParts, "\n")
		} else {
			descParts = append(descParts, line)
		}
	}

	ann.Description = buildDescription(descParts)
	return ann
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
