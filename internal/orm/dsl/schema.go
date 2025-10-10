package dsl

type Schema struct{}

type ComparisonOperator string

const (
	OpEqual       ComparisonOperator = "eq"
	OpNotEqual    ComparisonOperator = "neq"
	OpGreaterThan ComparisonOperator = "gt"
	OpLessThan    ComparisonOperator = "lt"
	OpGTE         ComparisonOperator = "gte"
	OpLTE         ComparisonOperator = "lte"
	OpILike       ComparisonOperator = "ilike"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type AggregateFunc string

const (
	AggCount AggregateFunc = "count"
	AggSum   AggregateFunc = "sum"
	AggAvg   AggregateFunc = "avg"
	AggMin   AggregateFunc = "min"
	AggMax   AggregateFunc = "max"
)

type FieldType string

const (
	TypeUUID            FieldType = "uuid"
	TypeText            FieldType = "text"
	TypeVarChar         FieldType = "varchar"
	TypeChar            FieldType = "char"
	TypeBoolean         FieldType = "boolean"
	TypeSmallInt        FieldType = "smallint"
	TypeInteger         FieldType = "integer"
	TypeBigInt          FieldType = "bigint"
	TypeSmallSerial     FieldType = "smallserial"
	TypeSerial          FieldType = "serial"
	TypeBigSerial       FieldType = "bigserial"
	TypeDecimal         FieldType = "decimal"
	TypeNumeric         FieldType = "numeric"
	TypeReal            FieldType = "real"
	TypeDoublePrecision FieldType = "double precision"
	TypeMoney           FieldType = "money"
	TypeBytea           FieldType = "bytea"
	TypeDate            FieldType = "date"
	TypeTime            FieldType = "time"
	TypeTimeTZ          FieldType = "timetz"
	TypeTimestamp       FieldType = "timestamp"
	TypeTimestampTZ     FieldType = "timestamptz"
	TypeInterval        FieldType = "interval"
	TypeJSON            FieldType = "json"
	TypeJSONB           FieldType = "jsonb"
	TypeXML             FieldType = "xml"
	TypeInet            FieldType = "inet"
	TypeCIDR            FieldType = "cidr"
	TypeMACAddr         FieldType = "macaddr"
	TypeMACAddr8        FieldType = "macaddr8"
	TypeBit             FieldType = "bit"
	TypeVarBit          FieldType = "varbit"
	TypeTSVector        FieldType = "tsvector"
	TypeTSQuery         FieldType = "tsquery"
	TypePoint           FieldType = "point"
	TypeLine            FieldType = "line"
	TypeLseg            FieldType = "lseg"
	TypeBox             FieldType = "box"
	TypePath            FieldType = "path"
	TypePolygon         FieldType = "polygon"
	TypeCircle          FieldType = "circle"
	TypeInt4Range       FieldType = "int4range"
	TypeInt8Range       FieldType = "int8range"
	TypeNumRange        FieldType = "numrange"
	TypeTSRange         FieldType = "tsrange"
	TypeTSTZRange       FieldType = "tstzrange"
	TypeDateRange       FieldType = "daterange"
	TypeArray           FieldType = "array"
	TypeGeometry        FieldType = "geometry"
	TypeGeography       FieldType = "geography"
	TypeVector          FieldType = "vector"

	// Backwards compatibility aliases.
	TypeString FieldType = TypeText
	TypeInt    FieldType = TypeInteger
	TypeFloat  FieldType = TypeDoublePrecision
	TypeBool   FieldType = TypeBoolean
	TypeBytes  FieldType = TypeBytea
)

type ExpressionSpec struct {
	SQL          string
	Dependencies []string
}

func Expression(sql string, deps ...string) ExpressionSpec {
	return ExpressionSpec{SQL: sql, Dependencies: append([]string(nil), deps...)}
}

type IdentityMode string

const (
	IdentityByDefault IdentityMode = "by_default"
	IdentityAlways    IdentityMode = "always"
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
	ComputedSpec  *ComputedColumn
	ReadOnly      bool
	Annotations   map[string]any
}

type ComputedColumn struct {
	Expression ExpressionSpec
	Stored     bool
	ReadOnly   bool
}

func Computed(expr ExpressionSpec) ComputedColumn {
	return ComputedColumn{Expression: expr, Stored: true, ReadOnly: true}
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
func (f Field) Identity(mode IdentityMode) Field {
	if mode == "" {
		mode = IdentityByDefault
	}
	f = f.annotate("identity", true)
	f = f.annotate("identity_mode", string(mode))
	return f
}
func (f Field) Length(size int) Field {
	if size <= 0 {
		return f
	}
	return f.annotate("length", size)
}
func (f Field) Precision(p int) Field {
	if p <= 0 {
		return f
	}
	return f.annotate("precision", p)
}
func (f Field) Scale(s int) Field {
	if s < 0 {
		return f
	}
	return f.annotate("scale", s)
}
func (f Field) ArrayElement(elem FieldType) Field {
	return f.annotate("array_element", elem)
}

func (f Field) Computed(spec ComputedColumn) Field {
	copy := spec
	f.ComputedSpec = &copy
	if copy.ReadOnly {
		f.ReadOnly = true
	}
	return f
}
func (f Field) annotate(key string, val any) Field {
	if f.Annotations == nil {
		f.Annotations = map[string]any{}
	}
	f.Annotations[key] = val
	return f
}

func Text(name string) Field { return Field{Name: name, Type: TypeText, GoType: "string"} }
func VarChar(name string, size int) Field {
	field := Field{Name: name, Type: TypeVarChar, GoType: "string"}
	if size > 0 {
		field = field.Length(size)
	}
	return field
}
func Char(name string, size int) Field {
	field := Field{Name: name, Type: TypeChar, GoType: "string"}
	if size > 0 {
		field = field.Length(size)
	}
	return field
}
func Boolean(name string) Field { return Field{Name: name, Type: TypeBoolean, GoType: "bool"} }
func SmallInt(name string) Field {
	return Field{Name: name, Type: TypeSmallInt, GoType: "int16"}
}
func Integer(name string) Field {
	return Field{Name: name, Type: TypeInteger, GoType: "int32"}
}
func BigInt(name string) Field {
	return Field{Name: name, Type: TypeBigInt, GoType: "int64"}
}
func SmallSerial(name string) Field {
	return Field{Name: name, Type: TypeSmallSerial, GoType: "int16"}
}
func Serial(name string) Field {
	return Field{Name: name, Type: TypeSerial, GoType: "int32"}
}
func BigSerial(name string) Field {
	return Field{Name: name, Type: TypeBigSerial, GoType: "int64"}
}
func SmallIntIdentity(name string, mode IdentityMode) Field {
	return SmallInt(name).Identity(mode)
}
func IntegerIdentity(name string, mode IdentityMode) Field {
	return Integer(name).Identity(mode)
}
func BigIntIdentity(name string, mode IdentityMode) Field {
	return BigInt(name).Identity(mode)
}
func Decimal(name string, precision, scale int) Field {
	field := Field{Name: name, Type: TypeDecimal, GoType: "string"}
	if precision > 0 {
		field = field.Precision(precision)
	}
	if scale >= 0 {
		field = field.Scale(scale)
	}
	return field
}
func Numeric(name string, precision, scale int) Field {
	field := Field{Name: name, Type: TypeNumeric, GoType: "string"}
	if precision > 0 {
		field = field.Precision(precision)
	}
	if scale >= 0 {
		field = field.Scale(scale)
	}
	return field
}
func Real(name string) Field { return Field{Name: name, Type: TypeReal, GoType: "float32"} }
func DoublePrecision(name string) Field {
	return Field{Name: name, Type: TypeDoublePrecision, GoType: "float64"}
}
func Money(name string) Field     { return Field{Name: name, Type: TypeMoney, GoType: "string"} }
func Bytea(name string) Field     { return Field{Name: name, Type: TypeBytea, GoType: "[]byte"} }
func Date(name string) Field      { return Field{Name: name, Type: TypeDate, GoType: "time.Time"} }
func Time(name string) Field      { return Field{Name: name, Type: TypeTime, GoType: "time.Time"} }
func TimeTZ(name string) Field    { return Field{Name: name, Type: TypeTimeTZ, GoType: "time.Time"} }
func Timestamp(name string) Field { return Field{Name: name, Type: TypeTimestamp, GoType: "time.Time"} }
func TimestampTZ(name string) Field {
	return Field{Name: name, Type: TypeTimestampTZ, GoType: "time.Time"}
}
func Interval(name string) Field { return Field{Name: name, Type: TypeInterval, GoType: "string"} }
func JSON(name string) Field {
	return Field{Name: name, Type: TypeJSON, GoType: "json.RawMessage"}.annotate("format", "json")
}
func JSONB(name string) Field {
	return Field{Name: name, Type: TypeJSONB, GoType: "json.RawMessage"}.annotate("format", "jsonb")
}
func XML(name string) Field  { return Field{Name: name, Type: TypeXML, GoType: "string"} }
func UUID(name string) Field { return Field{Name: name, Type: TypeUUID, GoType: "string"} }
func UUIDv7(name string) Field {
	return Field{Name: name, Type: TypeUUID, GoType: "string"}
}
func Inet(name string) Field     { return Field{Name: name, Type: TypeInet, GoType: "string"} }
func CIDR(name string) Field     { return Field{Name: name, Type: TypeCIDR, GoType: "string"} }
func MACAddr(name string) Field  { return Field{Name: name, Type: TypeMACAddr, GoType: "string"} }
func MACAddr8(name string) Field { return Field{Name: name, Type: TypeMACAddr8, GoType: "string"} }
func Bit(name string, length int) Field {
	field := Field{Name: name, Type: TypeBit, GoType: "string"}
	if length > 0 {
		field = field.Length(length)
	}
	return field
}
func VarBit(name string, length int) Field {
	field := Field{Name: name, Type: TypeVarBit, GoType: "string"}
	if length > 0 {
		field = field.Length(length)
	}
	return field
}
func TSVector(name string) Field { return Field{Name: name, Type: TypeTSVector, GoType: "string"} }
func TSQuery(name string) Field  { return Field{Name: name, Type: TypeTSQuery, GoType: "string"} }
func Point(name string) Field    { return Field{Name: name, Type: TypePoint, GoType: "string"} }
func Line(name string) Field     { return Field{Name: name, Type: TypeLine, GoType: "string"} }
func Lseg(name string) Field     { return Field{Name: name, Type: TypeLseg, GoType: "string"} }
func Box(name string) Field      { return Field{Name: name, Type: TypeBox, GoType: "string"} }
func Path(name string) Field     { return Field{Name: name, Type: TypePath, GoType: "string"} }
func Polygon(name string) Field  { return Field{Name: name, Type: TypePolygon, GoType: "string"} }
func Circle(name string) Field   { return Field{Name: name, Type: TypeCircle, GoType: "string"} }
func Int4Range(name string) Field {
	return Field{Name: name, Type: TypeInt4Range, GoType: "string"}
}
func Int8Range(name string) Field {
	return Field{Name: name, Type: TypeInt8Range, GoType: "string"}
}
func NumRange(name string) Field {
	return Field{Name: name, Type: TypeNumRange, GoType: "string"}
}
func TSRange(name string) Field {
	return Field{Name: name, Type: TypeTSRange, GoType: "string"}
}
func TSTZRange(name string) Field {
	return Field{Name: name, Type: TypeTSTZRange, GoType: "string"}
}
func DateRange(name string) Field {
	return Field{Name: name, Type: TypeDateRange, GoType: "string"}
}
func Array(name string, element FieldType) Field {
	return Field{Name: name, Type: TypeArray}.ArrayElement(element)
}
func Geometry(name string) Field  { return Field{Name: name, Type: TypeGeometry, GoType: "[]byte"} }
func Geography(name string) Field { return Field{Name: name, Type: TypeGeography, GoType: "[]byte"} }
func Vector(name string, dim int) Field {
	if dim <= 0 {
		panic("vector dimensions must be positive")
	}
	return Field{Name: name, Type: TypeVector, GoType: "[]float32"}.annotate("vector_dim", dim)
}

func String(name string) Field { return Text(name) }
func Int(name string) Field    { return Integer(name) }
func Float(name string) Field  { return DoublePrecision(name) }
func Bool(name string) Field   { return Boolean(name) }
func Bytes(name string) Field  { return Bytea(name) }

type EdgeKind string

const (
	EdgeToOne      EdgeKind = "o2o"
	EdgeToMany     EdgeKind = "o2m"
	EdgeManyToMany EdgeKind = "m2m"
)

type CascadeAction string

const (
	CascadeUnset      CascadeAction = ""
	CascadeNoAction   CascadeAction = "NO ACTION"
	CascadeRestrict   CascadeAction = "RESTRICT"
	CascadeCascade    CascadeAction = "CASCADE"
	CascadeSetNull    CascadeAction = "SET NULL"
	CascadeSetDefault CascadeAction = "SET DEFAULT"
)

type EdgeCascade struct {
	OnDelete CascadeAction
	OnUpdate CascadeAction
}

type EdgeTarget struct {
	Entity    string
	Condition string
}

func PolymorphicTarget(entity, condition string) EdgeTarget {
	return EdgeTarget{Entity: entity, Condition: condition}
}

type Edge struct {
	Name               string
	Column             string
	RefName            string
	Through            string
	Target             string
	Kind               EdgeKind
	Nullable           bool
	Unique             bool
	Annotations        map[string]any
	InverseName        string
	PolymorphicTargets []EdgeTarget
	Cascade            EdgeCascade
}

func (e Edge) Field(name string) Edge        { e.Column = name; return e }
func (e Edge) Ref(name string) Edge          { e.RefName = name; return e }
func (e Edge) ThroughTable(name string) Edge { e.Through = name; return e }
func (e Edge) Optional() Edge                { e.Nullable = true; return e }
func (e Edge) UniqueEdge() Edge              { e.Unique = true; return e }
func (e Edge) Inverse(name string) Edge      { e.InverseName = name; return e }
func (e Edge) annotate(key string, val any) Edge {
	if e.Annotations == nil {
		e.Annotations = map[string]any{}
	}
	e.Annotations[key] = val
	return e
}

func (e Edge) Polymorphic(targets ...EdgeTarget) Edge {
	if len(targets) == 0 {
		return e
	}
	existing := append([]EdgeTarget(nil), e.PolymorphicTargets...)
	e.PolymorphicTargets = append(existing, targets...)
	return e
}

func (e Edge) OnDelete(action CascadeAction) Edge {
	e.Cascade.OnDelete = action
	return e
}

func (e Edge) OnUpdate(action CascadeAction) Edge {
	e.Cascade.OnUpdate = action
	return e
}

func (e Edge) OnDeleteCascade() Edge  { return e.OnDelete(CascadeCascade) }
func (e Edge) OnDeleteSetNull() Edge  { return e.OnDelete(CascadeSetNull) }
func (e Edge) OnDeleteRestrict() Edge { return e.OnDelete(CascadeRestrict) }
func (e Edge) OnDeleteNoAction() Edge { return e.OnDelete(CascadeNoAction) }
func (e Edge) OnUpdateCascade() Edge  { return e.OnUpdate(CascadeCascade) }
func (e Edge) OnUpdateSetNull() Edge  { return e.OnUpdate(CascadeSetNull) }
func (e Edge) OnUpdateRestrict() Edge { return e.OnUpdate(CascadeRestrict) }
func (e Edge) OnUpdateNoAction() Edge { return e.OnUpdate(CascadeNoAction) }

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

func ToOne(name, target string) Edge  { return Edge{Name: name, Target: target, Kind: EdgeToOne} }
func ToMany(name, target string) Edge { return Edge{Name: name, Target: target, Kind: EdgeToMany} }
func ManyToMany(name, target string) Edge {
	return Edge{Name: name, Target: target, Kind: EdgeManyToMany}
}

type Predicate struct {
	Name     string
	Field    string
	Operator ComparisonOperator
}

func NewPredicate(field string, op ComparisonOperator) Predicate {
	return Predicate{Field: field, Operator: op}
}

func (p Predicate) Named(name string) Predicate {
	p.Name = name
	return p
}

type Order struct {
	Name      string
	Field     string
	Direction SortDirection
}

func OrderBy(field string, dir SortDirection) Order {
	return Order{Field: field, Direction: dir}
}

func (o Order) Named(name string) Order {
	o.Name = name
	return o
}

type Aggregate struct {
	Name   string
	Func   AggregateFunc
	Field  string
	GoType string
}

func NewAggregate(name string, fn AggregateFunc) Aggregate {
	return Aggregate{Name: name, Func: fn}
}

func (a Aggregate) On(field string) Aggregate {
	a.Field = field
	return a
}

func (a Aggregate) WithGoType(goType string) Aggregate {
	a.GoType = goType
	return a
}

func CountAggregate(name string) Aggregate {
	return Aggregate{Name: name, Func: AggCount, GoType: "int", Field: "*"}
}

type QuerySpec struct {
	Predicates   []Predicate
	Orders       []Order
	Aggregates   []Aggregate
	DefaultLimit int
	MaxLimit     int
}

func Query() QuerySpec { return QuerySpec{} }

func (q QuerySpec) WithPredicates(predicates ...Predicate) QuerySpec {
	q.Predicates = append(q.Predicates, predicates...)
	return q
}

func (q QuerySpec) WithOrders(orders ...Order) QuerySpec {
	q.Orders = append(q.Orders, orders...)
	return q
}

func (q QuerySpec) WithAggregates(aggregates ...Aggregate) QuerySpec {
	q.Aggregates = append(q.Aggregates, aggregates...)
	return q
}

func (q QuerySpec) WithDefaultLimit(limit int) QuerySpec {
	if limit > 0 {
		q.DefaultLimit = limit
	}
	return q
}

func (q QuerySpec) WithMaxLimit(limit int) QuerySpec {
	if limit > 0 {
		q.MaxLimit = limit
	}
	return q
}
