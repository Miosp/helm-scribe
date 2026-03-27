package model

// ItemDef describes a single property within an object array item.
type ItemDef struct {
	Path string
	Type string
}

// ValueNode is the intermediate representation of a single values.yaml entry.
type ValueNode struct {
	Key         string
	Path        string // dot-notation: "image.repository"
	Description string
	Type        string // inferred from value
	Default     interface{}
	Nullable    bool
	// ItemNullable indicates array items are nullable (e.g. string?[]). Only meaningful when Type ends in "[]".
	ItemNullable bool
	Section      string
	Skip         bool
	Children     []*ValueNode
	Items           []*ItemDef
	Enum            []string
	Min             *float64
	Max             *float64
	Deprecated      string
	DefaultOverride *string
	Example         string
	Pattern         string
}
