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
	Nullable     bool
	ItemNullable bool
	Section     string
	Skip        bool
	Children    []*ValueNode
	Items       []*ItemDef
}
