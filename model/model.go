package model

// ValueNode is the intermediate representation of a single values.yaml entry.
type ValueNode struct {
	Key         string
	Path        string // dot-notation: "image.repository"
	Description string
	Type        string // inferred from value
	Default     interface{}
	Nullable    bool
	Section     string
	Skip        bool
	Children    []*ValueNode
}
