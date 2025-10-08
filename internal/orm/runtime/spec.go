package runtime

import "github.com/deicod/erm/internal/orm/dsl"

type FieldSpec struct {
	Name        string
	Column      string
	GoType      string
	Type        dsl.FieldType
	Primary     bool
	Nullable    bool
	Unique      bool
	DefaultNow  bool
	UpdateNow   bool
	DefaultExpr string
	Annotations map[string]any
}

type EdgeSpec struct {
	Name        string
	Column      string
	RefName     string
	Through     string
	Target      string
	Kind        dsl.EdgeKind
	Nullable    bool
	Unique      bool
	Annotations map[string]any
}

type IndexSpec struct {
	Name             string
	Columns          []string
	Unique           bool
	Where            string
	Method           string
	NullsNotDistinct bool
	Annotations      map[string]any
}

type EntitySpec struct {
	Name    string
	Table   string
	Fields  []FieldSpec
	Edges   []EdgeSpec
	Indexes []IndexSpec
}

type Registry struct {
	Entities map[string]EntitySpec
}

func (r Registry) Entity(name string) (EntitySpec, bool) {
	if r.Entities == nil {
		return EntitySpec{}, false
	}
	spec, ok := r.Entities[name]
	return spec, ok
}
