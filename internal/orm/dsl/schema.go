package dsl

type Schema struct{}

type Field struct {
	Name string
	Type string
	Primary bool
	Nullable bool
	Annotations map[string]any
}

func (f Field) Primary() Field { f.Primary = true; return f }
func (f Field) Optional() Field { f.Nullable = true; return f }
func (f Field) Unique() Field { if f.Annotations==nil { f.Annotations=map[string]any{} }; f.Annotations["unique"]=true; return f }
func (f Field) NotEmpty() Field { if f.Annotations==nil { f.Annotations=map[string]any{} }; f.Annotations["notEmpty"]=true; return f }
func (f Field) DefaultNow() Field { if f.Annotations==nil { f.Annotations=map[string]any{} }; f.Annotations["defaultNow"]=true; return f }
func (f Field) UpdateNow() Field { if f.Annotations==nil { f.Annotations=map[string]any{} }; f.Annotations["updateNow"]=true; return f }

type Edge struct {
	Name string
	Type string
	Inverse string
}

type Index struct {
	Name string
	Columns []string
	IsUnique bool
}

func Idx(name string) Index { return Index{Name:name} }
func (i Index) On(cols ...string) Index { i.Columns = cols; return i }
func (i Index) Unique() Index { i.IsUnique = true; return i }

func UUIDv7(name string) Field { return Field{Name: name, Type: "uuidv7"} }
func String(name string) Field { return Field{Name: name, Type: "string"} }
func Time(name string) Field { return Field{Name: name, Type: "time"} }

func ToMany(name, typ string) Edge { return Edge{Name:name, Type:typ} }
