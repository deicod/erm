package dsl

type Schema struct{}

type FieldType string

const (
	TypeUUID      FieldType = "uuid"
	TypeString    FieldType = "string"
	TypeInt       FieldType = "int"
	TypeFloat     FieldType = "float"
	TypeBool      FieldType = "bool"
	TypeBytes     FieldType = "bytes"
	TypeTime      FieldType = "time"
	TypeJSON      FieldType = "json"
	TypeGeometry  FieldType = "geometry"
	TypeGeography FieldType = "geography"
	TypeVector    FieldType = "vector"
)

type Field struct {
	Name          string
	Column        string
	GoType        string
	Type          FieldType
	IsPrimary     bool
	Nullable      bool
	IsUnique      bool
	HasDefaultNow bool
	HasUpdateNow  bool
	DefaultExpr   string
	Annotations   map[string]any
}

func (f Field) Primary() Field                { f.IsPrimary = true; return f }
func (f Field) Optional() Field               { f.Nullable = true; return f }
func (f Field) ColumnName(name string) Field  { f.Column = name; return f }
func (f Field) UniqueConstraint() Field       { f.IsUnique = true; return f }
func (f Field) Unique() Field                 { return f.UniqueConstraint() }
func (f Field) NotEmpty() Field               { return f.annotate("notEmpty", true) }
func (f Field) DefaultNow() Field             { f.HasDefaultNow = true; return f }
func (f Field) UpdateNow() Field              { f.HasUpdateNow = true; return f }
func (f Field) WithDefault(expr string) Field { f.DefaultExpr = expr; return f }
func (f Field) SRID(srid int) Field           { return f.annotate("srid", srid) }
func (f Field) TimeSeries() Field             { return f.annotate("timeseries", true) }
func (f Field) annotate(key string, val any) Field {
	if f.Annotations == nil {
		f.Annotations = map[string]any{}
	}
	f.Annotations[key] = val
	return f
}

type EdgeKind string

const (
	EdgeToOne      EdgeKind = "o2o"
	EdgeToMany     EdgeKind = "o2m"
	EdgeManyToMany EdgeKind = "m2m"
)

type Edge struct {
	Name        string
	Column      string
	RefName     string
	Through     string
	Target      string
	Kind        EdgeKind
	Nullable    bool
	Unique      bool
	Annotations map[string]any
}

func (e Edge) Field(name string) Edge        { e.Column = name; return e }
func (e Edge) Ref(name string) Edge          { e.RefName = name; return e }
func (e Edge) ThroughTable(name string) Edge { e.Through = name; return e }
func (e Edge) Optional() Edge                { e.Nullable = true; return e }
func (e Edge) UniqueEdge() Edge              { e.Unique = true; return e }
func (e Edge) annotate(key string, val any) Edge {
	if e.Annotations == nil {
		e.Annotations = map[string]any{}
	}
	e.Annotations[key] = val
	return e
}

type Index struct {
	Name             string
	Columns          []string
	IsUnique         bool
	Where            string
	Method           string
	NullsNotDistinct bool
	Annotations      map[string]any
}

func Idx(name string) Index                       { return Index{Name: name} }
func (i Index) On(cols ...string) Index           { i.Columns = cols; return i }
func (i Index) UniqueConstraint() Index           { i.IsUnique = true; return i }
func (i Index) Unique() Index                     { return i.UniqueConstraint() }
func (i Index) WhereClause(clause string) Index   { i.Where = clause; return i }
func (i Index) MethodUsing(method string) Index   { i.Method = method; return i }
func (i Index) NullsNotDistinctConstraint() Index { i.NullsNotDistinct = true; return i }
func (i Index) annotate(key string, val any) Index {
	if i.Annotations == nil {
		i.Annotations = map[string]any{}
	}
	i.Annotations[key] = val
	return i
}

func UUIDv7(name string) Field { return Field{Name: name, Type: TypeUUID, GoType: "string"} }
func String(name string) Field { return Field{Name: name, Type: TypeString, GoType: "string"} }
func Int(name string) Field    { return Field{Name: name, Type: TypeInt, GoType: "int"} }
func Float(name string) Field  { return Field{Name: name, Type: TypeFloat, GoType: "float64"} }
func Bool(name string) Field   { return Field{Name: name, Type: TypeBool, GoType: "bool"} }
func Bytes(name string) Field  { return Field{Name: name, Type: TypeBytes, GoType: "[]byte"} }
func Time(name string) Field   { return Field{Name: name, Type: TypeTime, GoType: "time.Time"} }
func JSON(name string) Field {
	return Field{Name: name, Type: TypeJSON, GoType: "json.RawMessage"}.annotate("format", "json")
}
func Geometry(name string) Field  { return Field{Name: name, Type: TypeGeometry, GoType: "[]byte"} }
func Geography(name string) Field { return Field{Name: name, Type: TypeGeography, GoType: "[]byte"} }
func Vector(name string, dim int) Field {
	return Field{Name: name, Type: TypeVector, GoType: "[]float32"}.annotate("vector_dim", dim)
}

func ToOne(name, target string) Edge  { return Edge{Name: name, Target: target, Kind: EdgeToOne} }
func ToMany(name, target string) Edge { return Edge{Name: name, Target: target, Kind: EdgeToMany} }
func ManyToMany(name, target string) Edge {
	return Edge{Name: name, Target: target, Kind: EdgeManyToMany}
}
