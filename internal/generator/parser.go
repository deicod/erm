package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deicod/erm/internal/orm/dsl"
)

type Entity struct {
	Name        string
	Fields      []dsl.Field
	Edges       []dsl.Edge
	Indexes     []dsl.Index
	Query       dsl.QuerySpec
	Annotations []dsl.Annotation
}

func loadEntities(root string) ([]Entity, error) {
	schemaDir := filepath.Join(root, "schema")
	matches, err := filepath.Glob(filepath.Join(schemaDir, "*.schema.go"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	fset := token.NewFileSet()
	typeDecls := map[string]*ast.StructType{}
	files := map[string]*ast.File{}

	for _, file := range matches {
		astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		files[file] = astFile
		for _, decl := range astFile.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if embedsDSLSchema(st) {
					typeDecls[ts.Name.Name] = st
				}
			}
		}
	}

	evaluator := newExprEvaluator()
	entities := make([]Entity, 0, len(typeDecls))
	for _, astFile := range files {
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}
			recv := receiverName(fn)
			if _, exists := typeDecls[recv]; !exists {
				continue
			}

			ent := ensureEntity(&entities, recv)
			switch fn.Name.Name {
			case "Fields":
				fields, err := evaluator.evalFieldSlice(fn)
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", recv, fn.Name.Name, err)
				}
				ent.Fields = fields
			case "Edges":
				edges, err := evaluator.evalEdgeSlice(fn)
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", recv, fn.Name.Name, err)
				}
				ent.Edges = edges
			case "Indexes":
				indexes, err := evaluator.evalIndexSlice(fn)
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", recv, fn.Name.Name, err)
				}
				ent.Indexes = indexes
			case "Query":
				spec, err := evaluator.evalQuerySpec(fn)
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", recv, fn.Name.Name, err)
				}
				ent.Query = spec
			case "Annotations":
				annotations, err := evaluator.evalAnnotationSlice(fn)
				if err != nil {
					return nil, fmt.Errorf("%s.%s: %w", recv, fn.Name.Name, err)
				}
				ent.Annotations = annotations
			}
		}
	}

	// ensure deterministic ordering and presence for all types
	out := make([]Entity, 0, len(typeDecls))
	for name := range typeDecls {
		ent := findEntity(entities, name)
		out = append(out, ent)
	}
	for i := range out {
		ensureDefaultField(&out[i])
		ensureDefaultQuery(&out[i])
	}
	synthesizeInverseEdges(out)
	return out, nil
}

func embedsDSLSchema(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}
	for _, field := range st.Fields.List {
		if len(field.Names) > 0 {
			continue
		}
		switch expr := field.Type.(type) {
		case *ast.SelectorExpr:
			if ident, ok := expr.X.(*ast.Ident); ok && ident.Name == "dsl" && expr.Sel.Name == "Schema" {
				return true
			}
		}
	}
	return false
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	field := fn.Recv.List[0]
	switch expr := field.Type.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func ensureEntity(entities *[]Entity, name string) *Entity {
	for i := range *entities {
		if (*entities)[i].Name == name {
			return &(*entities)[i]
		}
	}
	*entities = append(*entities, Entity{Name: name})
	return &(*entities)[len(*entities)-1]
}

func findEntity(entities []Entity, name string) Entity {
	for _, ent := range entities {
		if ent.Name == name {
			if ent.Fields == nil {
				ent.Fields = []dsl.Field{}
			}
			if ent.Edges == nil {
				ent.Edges = []dsl.Edge{}
			}
			if ent.Indexes == nil {
				ent.Indexes = []dsl.Index{}
			}
			if ent.Annotations == nil {
				ent.Annotations = []dsl.Annotation{}
			}
			return ent
		}
	}
	return Entity{Name: name, Fields: []dsl.Field{}, Edges: []dsl.Edge{}, Indexes: []dsl.Index{}, Annotations: []dsl.Annotation{}}
}

func ensureDefaultField(ent *Entity) {
	for _, field := range ent.Fields {
		if strings.EqualFold(field.Name, "id") {
			return
		}
	}
	ent.Fields = append([]dsl.Field{dsl.UUIDv7("id").Primary()}, ent.Fields...)
}

func ensureDefaultQuery(ent *Entity) {
	if ent.Query.DefaultLimit == 0 {
		ent.Query.DefaultLimit = 20
	}
	if len(ent.Query.Predicates) == 0 {
		ent.Query.Predicates = []dsl.Predicate{
			dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
		}
	}
	if len(ent.Query.Orders) == 0 {
		ent.Query.Orders = []dsl.Order{
			dsl.OrderBy("id", dsl.SortAsc).Named("IDAsc"),
		}
	}
	if len(ent.Query.Aggregates) == 0 {
		ent.Query.Aggregates = []dsl.Aggregate{
			dsl.CountAggregate("Count"),
		}
	}
}

type exprEvaluator struct{}

func newExprEvaluator() *exprEvaluator { return &exprEvaluator{} }

func (e *exprEvaluator) evalFieldSlice(fn *ast.FuncDecl) ([]dsl.Field, error) {
	if fn.Body == nil {
		return nil, errors.New("missing body")
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		val, err := e.evalExpr(ret.Results[0])
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		fields, ok := val.([]dsl.Field)
		if !ok {
			return nil, fmt.Errorf("expected []dsl.Field, got %T", val)
		}
		return fields, nil
	}
	return nil, errors.New("no return statement found")
}

func (e *exprEvaluator) evalEdgeSlice(fn *ast.FuncDecl) ([]dsl.Edge, error) {
	if fn.Body == nil {
		return nil, errors.New("missing body")
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		val, err := e.evalExpr(ret.Results[0])
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		edges, ok := val.([]dsl.Edge)
		if !ok {
			return nil, fmt.Errorf("expected []dsl.Edge, got %T", val)
		}
		return edges, nil
	}
	return nil, errors.New("no return statement found")
}

func (e *exprEvaluator) evalIndexSlice(fn *ast.FuncDecl) ([]dsl.Index, error) {
	if fn.Body == nil {
		return nil, errors.New("missing body")
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		val, err := e.evalExpr(ret.Results[0])
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		indexes, ok := val.([]dsl.Index)
		if !ok {
			return nil, fmt.Errorf("expected []dsl.Index, got %T", val)
		}
		return indexes, nil
	}
	return nil, errors.New("no return statement found")
}

func (e *exprEvaluator) evalQuerySpec(fn *ast.FuncDecl) (dsl.QuerySpec, error) {
	if fn.Body == nil {
		return dsl.QuerySpec{}, errors.New("missing body")
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		val, err := e.evalExpr(ret.Results[0])
		if err != nil {
			return dsl.QuerySpec{}, err
		}
		spec, ok := val.(dsl.QuerySpec)
		if !ok {
			return dsl.QuerySpec{}, fmt.Errorf("expected dsl.QuerySpec, got %T", val)
		}
		return spec, nil
	}
	return dsl.QuerySpec{}, errors.New("no return statement found")
}

func (e *exprEvaluator) evalAnnotationSlice(fn *ast.FuncDecl) ([]dsl.Annotation, error) {
	if fn.Body == nil {
		return nil, errors.New("missing body")
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}
		val, err := e.evalExpr(ret.Results[0])
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		annotations, ok := val.([]dsl.Annotation)
		if !ok {
			return nil, fmt.Errorf("expected []dsl.Annotation, got %T", val)
		}
		return annotations, nil
	}
	return nil, errors.New("no return statement found")
}

func (e *exprEvaluator) evalExpr(expr ast.Expr) (any, error) {
	switch exp := expr.(type) {
	case *ast.BasicLit:
		switch exp.Kind {
		case token.STRING:
			return strconv.Unquote(exp.Value)
		case token.INT:
			return strconv.Atoi(exp.Value)
		case token.FLOAT:
			return strconv.ParseFloat(exp.Value, 64)
		}
	case *ast.Ident:
		switch exp.Name {
		case "nil":
			return nil, nil
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return exp.Name, nil
		}
	case *ast.CompositeLit:
		return e.evalCompositeLit(exp)
	case *ast.CallExpr:
		return e.evalCallExpr(exp)
	case *ast.SelectorExpr:
		// This can happen for constants like time.Second; return fully-qualified name
		prefix, err := e.evalExpr(exp.X)
		if err != nil {
			return nil, err
		}
		switch p := prefix.(type) {
		case string:
			if p == "dsl" {
				// treat as simple identifier for fallback
				return exp.Sel.Name, nil
			}
			return p + "." + exp.Sel.Name, nil
		default:
			return nil, fmt.Errorf("unsupported selector base %T", prefix)
		}
	}
	return nil, fmt.Errorf("unsupported expression %T", expr)
}

func (e *exprEvaluator) evalCompositeLit(lit *ast.CompositeLit) (any, error) {
	switch typ := lit.Type.(type) {
	case *ast.ArrayType:
		switch elem := typ.Elt.(type) {
		case *ast.SelectorExpr:
			ident, ok := elem.X.(*ast.Ident)
			if !ok || ident.Name != "dsl" {
				return nil, fmt.Errorf("unsupported array element type %T", elem)
			}
			switch elem.Sel.Name {
			case "Field":
				var fields []dsl.Field
				for _, elt := range lit.Elts {
					val, err := e.evalExpr(elt)
					if err != nil {
						return nil, err
					}
					field, ok := val.(dsl.Field)
					if !ok {
						return nil, fmt.Errorf("expected dsl.Field, got %T", val)
					}
					fields = append(fields, field)
				}
				return fields, nil
			case "Edge":
				var edges []dsl.Edge
				for _, elt := range lit.Elts {
					val, err := e.evalExpr(elt)
					if err != nil {
						return nil, err
					}
					edge, ok := val.(dsl.Edge)
					if !ok {
						return nil, fmt.Errorf("expected dsl.Edge, got %T", val)
					}
					edges = append(edges, edge)
				}
				return edges, nil
			case "Index":
				var indexes []dsl.Index
				for _, elt := range lit.Elts {
					val, err := e.evalExpr(elt)
					if err != nil {
						return nil, err
					}
					index, ok := val.(dsl.Index)
					if !ok {
						return nil, fmt.Errorf("expected dsl.Index, got %T", val)
					}
					indexes = append(indexes, index)
				}
				return indexes, nil
			case "Annotation":
				var annotations []dsl.Annotation
				for _, elt := range lit.Elts {
					val, err := e.evalExpr(elt)
					if err != nil {
						return nil, err
					}
					annotation, ok := val.(dsl.Annotation)
					if !ok {
						return nil, fmt.Errorf("expected dsl.Annotation, got %T", val)
					}
					annotations = append(annotations, annotation)
				}
				return annotations, nil
			}
		}
	}
	return nil, fmt.Errorf("unsupported composite literal %T", lit.Type)
}

func (e *exprEvaluator) evalCallExpr(call *ast.CallExpr) (any, error) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, fmt.Errorf("unsupported call expression %T", call.Fun)
	}

	// Evaluate base
	base, err := e.evalExpr(selector.X)
	if err != nil {
		return nil, err
	}

	args := make([]any, len(call.Args))
	for i, arg := range call.Args {
		val, err := e.evalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	switch b := base.(type) {
	case string:
		if b != "dsl" {
			return nil, fmt.Errorf("unsupported selector base %s", b)
		}
		return executeDSLFunc(selector.Sel.Name, args)
	case dsl.Field:
		return executeFieldMethod(b, selector.Sel.Name, args)
	case dsl.Edge:
		return executeEdgeMethod(b, selector.Sel.Name, args)
	case dsl.Index:
		return executeIndexMethod(b, selector.Sel.Name, args)
	case dsl.QuerySpec:
		return executeQuerySpecMethod(b, selector.Sel.Name, args)
	case dsl.Predicate:
		return executePredicateMethod(b, selector.Sel.Name, args)
	case dsl.Order:
		return executeOrderMethod(b, selector.Sel.Name, args)
	case dsl.Aggregate:
		return executeAggregateMethod(b, selector.Sel.Name, args)
	default:
		return nil, fmt.Errorf("unsupported call base %T", base)
	}
}

func executeDSLFunc(name string, args []any) (any, error) {
	switch name {
	case "UUIDv7":
		return dsl.UUIDv7(argString(args, 0)), nil
	case "UUID":
		return dsl.UUID(argString(args, 0)), nil
	case "Text":
		return dsl.Text(argString(args, 0)), nil
	case "String":
		return dsl.String(argString(args, 0)), nil
	case "VarChar":
		return dsl.VarChar(argString(args, 0), argIntDefault(args, 1, 0)), nil
	case "Char":
		return dsl.Char(argString(args, 0), argIntDefault(args, 1, 0)), nil
	case "Boolean":
		return dsl.Boolean(argString(args, 0)), nil
	case "Int":
		return dsl.Int(argString(args, 0)), nil
	case "Integer":
		return dsl.Integer(argString(args, 0)), nil
	case "SmallInt":
		return dsl.SmallInt(argString(args, 0)), nil
	case "BigInt":
		return dsl.BigInt(argString(args, 0)), nil
	case "SmallSerial":
		return dsl.SmallSerial(argString(args, 0)), nil
	case "Serial":
		return dsl.Serial(argString(args, 0)), nil
	case "BigSerial":
		return dsl.BigSerial(argString(args, 0)), nil
	case "SmallIntIdentity":
		return dsl.SmallIntIdentity(argString(args, 0), argIdentityMode(args, 1)), nil
	case "IntegerIdentity":
		return dsl.IntegerIdentity(argString(args, 0), argIdentityMode(args, 1)), nil
	case "BigIntIdentity":
		return dsl.BigIntIdentity(argString(args, 0), argIdentityMode(args, 1)), nil
	case "Float":
		return dsl.Float(argString(args, 0)), nil
	case "Real":
		return dsl.Real(argString(args, 0)), nil
	case "DoublePrecision":
		return dsl.DoublePrecision(argString(args, 0)), nil
	case "Decimal":
		return dsl.Decimal(argString(args, 0), argIntDefault(args, 1, 0), argIntDefault(args, 2, 0)), nil
	case "Numeric":
		return dsl.Numeric(argString(args, 0), argIntDefault(args, 1, 0), argIntDefault(args, 2, 0)), nil
	case "Bool":
		return dsl.Bool(argString(args, 0)), nil
	case "Bytes":
		return dsl.Bytes(argString(args, 0)), nil
	case "Bytea":
		return dsl.Bytea(argString(args, 0)), nil
	case "Money":
		return dsl.Money(argString(args, 0)), nil
	case "Date":
		return dsl.Date(argString(args, 0)), nil
	case "Time":
		return dsl.Time(argString(args, 0)), nil
	case "TimeTZ":
		return dsl.TimeTZ(argString(args, 0)), nil
	case "Timestamp":
		return dsl.Timestamp(argString(args, 0)), nil
	case "TimestampTZ":
		return dsl.TimestampTZ(argString(args, 0)), nil
	case "Interval":
		return dsl.Interval(argString(args, 0)), nil
	case "JSON":
		return dsl.JSON(argString(args, 0)), nil
	case "JSONB":
		return dsl.JSONB(argString(args, 0)), nil
	case "XML":
		return dsl.XML(argString(args, 0)), nil
	case "Inet":
		return dsl.Inet(argString(args, 0)), nil
	case "CIDR":
		return dsl.CIDR(argString(args, 0)), nil
	case "MACAddr":
		return dsl.MACAddr(argString(args, 0)), nil
	case "MACAddr8":
		return dsl.MACAddr8(argString(args, 0)), nil
	case "Bit":
		return dsl.Bit(argString(args, 0), argIntDefault(args, 1, 0)), nil
	case "VarBit":
		return dsl.VarBit(argString(args, 0), argIntDefault(args, 1, 0)), nil
	case "TSVector":
		return dsl.TSVector(argString(args, 0)), nil
	case "TSQuery":
		return dsl.TSQuery(argString(args, 0)), nil
	case "Point":
		return dsl.Point(argString(args, 0)), nil
	case "Line":
		return dsl.Line(argString(args, 0)), nil
	case "Lseg":
		return dsl.Lseg(argString(args, 0)), nil
	case "Box":
		return dsl.Box(argString(args, 0)), nil
	case "Path":
		return dsl.Path(argString(args, 0)), nil
	case "Polygon":
		return dsl.Polygon(argString(args, 0)), nil
	case "Circle":
		return dsl.Circle(argString(args, 0)), nil
	case "Int4Range":
		return dsl.Int4Range(argString(args, 0)), nil
	case "Int8Range":
		return dsl.Int8Range(argString(args, 0)), nil
	case "NumRange":
		return dsl.NumRange(argString(args, 0)), nil
	case "TSRange":
		return dsl.TSRange(argString(args, 0)), nil
	case "TSTZRange":
		return dsl.TSTZRange(argString(args, 0)), nil
	case "DateRange":
		return dsl.DateRange(argString(args, 0)), nil
	case "Array":
		elem, err := argFieldType(args, 1)
		if err != nil {
			return nil, err
		}
		return dsl.Array(argString(args, 0), elem), nil
	case "Query":
		return dsl.Query(), nil
	case "NewPredicate":
		op, err := argComparisonOperator(args, 1)
		if err != nil {
			return nil, err
		}
		return dsl.NewPredicate(argString(args, 0), op), nil
	case "OrderBy":
		dir, err := argSortDirection(args, 1)
		if err != nil {
			return nil, err
		}
		return dsl.OrderBy(argString(args, 0), dir), nil
	case "NewAggregate":
		fn, err := argAggregateFunc(args, 1)
		if err != nil {
			return nil, err
		}
		return dsl.NewAggregate(argString(args, 0), fn), nil
	case "CountAggregate":
		return dsl.CountAggregate(argString(args, 0)), nil
	case "GraphQL":
		opts := make([]dsl.GraphQLOption, 0, len(args)-1)
		for i := 1; i < len(args); i++ {
			opt, ok := args[i].(dsl.GraphQLOption)
			if !ok {
				return nil, fmt.Errorf("GraphQL expects dsl.GraphQLOption, got %T", args[i])
			}
			opts = append(opts, opt)
		}
		return dsl.GraphQL(argString(args, 0), opts...), nil
	case "GraphQLSubscriptions":
		events := make([]dsl.SubscriptionEvent, 0, len(args))
		for i := range args {
			event, err := argSubscriptionEvent(args, i)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		return dsl.GraphQLSubscriptions(events...), nil
	case "Geometry":
		return dsl.Geometry(argString(args, 0)), nil
	case "Geography":
		return dsl.Geography(argString(args, 0)), nil
	case "Vector":
		return dsl.Vector(argString(args, 0), argInt(args, 1)), nil
	case "Expression":
		if len(args) == 0 {
			return nil, fmt.Errorf("Expression requires SQL argument")
		}
		sql := argString(args, 0)
		deps := make([]string, 0, len(args)-1)
		for i := 1; i < len(args); i++ {
			switch v := args[i].(type) {
			case string:
				deps = append(deps, v)
			case []string:
				deps = append(deps, v...)
			case nil:
				continue
			default:
				return nil, fmt.Errorf("Expression dependencies must be string or []string, got %T", args[i])
			}
		}
		return dsl.Expression(sql, deps...), nil
	case "Computed":
		if len(args) == 0 {
			return nil, fmt.Errorf("Computed requires expression argument")
		}
		expr, ok := args[0].(dsl.ExpressionSpec)
		if !ok {
			return nil, fmt.Errorf("Computed expects dsl.ExpressionSpec, got %T", args[0])
		}
		return dsl.Computed(expr), nil
	case "ToOne":
		return dsl.ToOne(argString(args, 0), argString(args, 1)), nil
	case "ToMany":
		return dsl.ToMany(argString(args, 0), argString(args, 1)), nil
	case "ManyToMany":
		return dsl.ManyToMany(argString(args, 0), argString(args, 1)), nil
	case "PolymorphicTarget":
		return dsl.PolymorphicTarget(argString(args, 0), argString(args, 1)), nil
	case "Idx":
		return dsl.Idx(argString(args, 0)), nil
	default:
		return nil, fmt.Errorf("unsupported dsl function %s", name)
	}
}

func executeFieldMethod(f dsl.Field, name string, args []any) (any, error) {
	switch name {
	case "Primary":
		return f.Primary(), nil
	case "Optional":
		return f.Optional(), nil
	case "ColumnName":
		return f.ColumnName(argString(args, 0)), nil
	case "Unique", "UniqueConstraint":
		return f.Unique(), nil
	case "NotEmpty":
		return f.NotEmpty(), nil
	case "DefaultNow":
		return f.DefaultNow(), nil
	case "UpdateNow":
		return f.UpdateNow(), nil
	case "WithDefault":
		return f.WithDefault(argString(args, 0)), nil
	case "SRID":
		return f.SRID(argInt(args, 0)), nil
	case "TimeSeries":
		return f.TimeSeries(), nil
	case "Identity":
		return f.Identity(argIdentityMode(args, 0)), nil
	case "Length":
		return f.Length(argInt(args, 0)), nil
	case "Precision":
		return f.Precision(argInt(args, 0)), nil
	case "Scale":
		return f.Scale(argInt(args, 0)), nil
	case "ArrayElement":
		elem, err := argFieldType(args, 0)
		if err != nil {
			return nil, err
		}
		return f.ArrayElement(elem), nil
	case "Computed":
		if len(args) == 0 {
			return nil, fmt.Errorf("Computed expects descriptor argument")
		}
		switch spec := args[0].(type) {
		case dsl.ComputedColumn:
			return f.Computed(spec), nil
		case *dsl.ComputedColumn:
			if spec == nil {
				return nil, fmt.Errorf("Computed descriptor cannot be nil")
			}
			return f.Computed(*spec), nil
		default:
			return nil, fmt.Errorf("Computed expects dsl.ComputedColumn, got %T", args[0])
		}
	default:
		return nil, fmt.Errorf("unsupported field method %s", name)
	}
}

func executeEdgeMethod(edge dsl.Edge, name string, args []any) (any, error) {
	switch name {
	case "Field":
		return edge.Field(argString(args, 0)), nil
	case "Ref":
		return edge.Ref(argString(args, 0)), nil
	case "ThroughTable":
		return edge.ThroughTable(argString(args, 0)), nil
	case "Optional":
		return edge.Optional(), nil
	case "UniqueEdge":
		return edge.UniqueEdge(), nil
	case "Inverse":
		return edge.Inverse(argString(args, 0)), nil
	case "Polymorphic":
		targets := make([]dsl.EdgeTarget, len(args))
		for i, arg := range args {
			target, ok := arg.(dsl.EdgeTarget)
			if !ok {
				return nil, fmt.Errorf("Polymorphic expects dsl.EdgeTarget, got %T", arg)
			}
			targets[i] = target
		}
		return edge.Polymorphic(targets...), nil
	case "PolymorphicTargets":
		targets := make([]dsl.EdgeTarget, len(args))
		for i, arg := range args {
			target, ok := arg.(dsl.EdgeTarget)
			if !ok {
				return nil, fmt.Errorf("PolymorphicTargets expects dsl.EdgeTarget, got %T", arg)
			}
			targets[i] = target
		}
		return edge.Polymorphic(targets...), nil
	case "OnDelete":
		return edge.OnDelete(argCascade(args, 0)), nil
	case "OnUpdate":
		return edge.OnUpdate(argCascade(args, 0)), nil
	case "OnDeleteCascade":
		return edge.OnDeleteCascade(), nil
	case "OnDeleteSetNull":
		return edge.OnDeleteSetNull(), nil
	case "OnDeleteRestrict":
		return edge.OnDeleteRestrict(), nil
	case "OnDeleteNoAction":
		return edge.OnDeleteNoAction(), nil
	case "OnUpdateCascade":
		return edge.OnUpdateCascade(), nil
	case "OnUpdateSetNull":
		return edge.OnUpdateSetNull(), nil
	case "OnUpdateRestrict":
		return edge.OnUpdateRestrict(), nil
	case "OnUpdateNoAction":
		return edge.OnUpdateNoAction(), nil
	default:
		return nil, fmt.Errorf("unsupported edge method %s", name)
	}
}

func synthesizeInverseEdges(entities []Entity) {
	index := make(map[string]int, len(entities))
	for i := range entities {
		index[entities[i].Name] = i
	}
	for i := range entities {
		source := entities[i]
		for _, edge := range source.Edges {
			if edge.InverseName == "" {
				continue
			}
			targetIdx, ok := index[edge.Target]
			if !ok {
				continue
			}
			if hasEdgeNamed(entities[targetIdx].Edges, edge.InverseName) {
				continue
			}
			inverse := buildInverseEdge(source, edge)
			entities[targetIdx].Edges = append(entities[targetIdx].Edges, inverse)
		}
	}
}

func hasEdgeNamed(edges []dsl.Edge, name string) bool {
	for _, edge := range edges {
		if edge.Name == name {
			return true
		}
	}
	return false
}

func buildInverseEdge(source Entity, edge dsl.Edge) dsl.Edge {
	inverse := dsl.Edge{
		Name:        edge.InverseName,
		Target:      source.Name,
		Kind:        inverseEdgeKind(edge),
		Nullable:    edge.Nullable,
		Unique:      edge.Unique,
		Through:     edge.Through,
		InverseName: edge.Name,
		Cascade:     edge.Cascade,
	}
	if len(edge.PolymorphicTargets) > 0 {
		inverse.PolymorphicTargets = append([]dsl.EdgeTarget(nil), edge.PolymorphicTargets...)
	}
	switch edge.Kind {
	case dsl.EdgeToOne:
		// Mirror the resolved column from the forward edge (explicit or derived).
		inverse.RefName = edgeColumn(edge)
	case dsl.EdgeToMany:
		primary, _ := findPrimaryField(source)
		// Reuse the forward edge reference resolution so explicit overrides win.
		column := edgeRefColumn(source, edge, primary)
		inverse.Column = column
		if column != "" {
			inverse.RefName = column
		}
	case dsl.EdgeManyToMany:
		// Through already copied above.
	}
	return inverse
}

func inverseEdgeKind(edge dsl.Edge) dsl.EdgeKind {
	switch edge.Kind {
	case dsl.EdgeToOne:
		if edge.Unique {
			return dsl.EdgeToOne
		}
		return dsl.EdgeToMany
	case dsl.EdgeToMany:
		return dsl.EdgeToOne
	case dsl.EdgeManyToMany:
		return dsl.EdgeManyToMany
	default:
		return edge.Kind
	}
}

func executeIndexMethod(index dsl.Index, name string, args []any) (any, error) {
	switch name {
	case "On":
		cols := make([]string, len(args))
		for i := range args {
			cols[i] = argString(args, i)
		}
		return index.On(cols...), nil
	case "Unique", "UniqueConstraint":
		return index.Unique(), nil
	case "WhereClause":
		return index.WhereClause(argString(args, 0)), nil
	case "MethodUsing":
		return index.MethodUsing(argString(args, 0)), nil
	case "NullsNotDistinctConstraint":
		return index.NullsNotDistinctConstraint(), nil
	default:
		return nil, fmt.Errorf("unsupported index method %s", name)
	}
}

func executeQuerySpecMethod(spec dsl.QuerySpec, name string, args []any) (any, error) {
	switch name {
	case "WithPredicates":
		preds := make([]dsl.Predicate, len(args))
		for i, arg := range args {
			p, ok := arg.(dsl.Predicate)
			if !ok {
				return nil, fmt.Errorf("expected dsl.Predicate, got %T", arg)
			}
			preds[i] = p
		}
		return spec.WithPredicates(preds...), nil
	case "WithOrders":
		orders := make([]dsl.Order, len(args))
		for i, arg := range args {
			o, ok := arg.(dsl.Order)
			if !ok {
				return nil, fmt.Errorf("expected dsl.Order, got %T", arg)
			}
			orders[i] = o
		}
		return spec.WithOrders(orders...), nil
	case "WithAggregates":
		aggs := make([]dsl.Aggregate, len(args))
		for i, arg := range args {
			a, ok := arg.(dsl.Aggregate)
			if !ok {
				return nil, fmt.Errorf("expected dsl.Aggregate, got %T", arg)
			}
			aggs[i] = a
		}
		return spec.WithAggregates(aggs...), nil
	case "WithDefaultLimit":
		return spec.WithDefaultLimit(argInt(args, 0)), nil
	case "WithMaxLimit":
		return spec.WithMaxLimit(argInt(args, 0)), nil
	default:
		return nil, fmt.Errorf("unsupported query spec method %s", name)
	}
}

func executePredicateMethod(pred dsl.Predicate, name string, args []any) (any, error) {
	switch name {
	case "Named":
		return pred.Named(argString(args, 0)), nil
	default:
		return nil, fmt.Errorf("unsupported predicate method %s", name)
	}
}

func executeOrderMethod(order dsl.Order, name string, args []any) (any, error) {
	switch name {
	case "Named":
		return order.Named(argString(args, 0)), nil
	default:
		return nil, fmt.Errorf("unsupported order method %s", name)
	}
}

func executeAggregateMethod(agg dsl.Aggregate, name string, args []any) (any, error) {
	switch name {
	case "On":
		return agg.On(argString(args, 0)), nil
	case "WithGoType":
		return agg.WithGoType(argString(args, 0)), nil
	default:
		return nil, fmt.Errorf("unsupported aggregate method %s", name)
	}
}

func argString(args []any, idx int) string {
	if idx >= len(args) {
		return ""
	}
	if s, ok := args[idx].(string); ok {
		return s
	}
	return fmt.Sprint(args[idx])
}

func argInt(args []any, idx int) int {
	if idx >= len(args) {
		return 0
	}
	switch v := args[idx].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

func argIntDefault(args []any, idx, def int) int {
	if idx >= len(args) {
		return def
	}
	return argInt(args, idx)
}

func argIdentityMode(args []any, idx int) dsl.IdentityMode {
	if idx >= len(args) {
		return dsl.IdentityByDefault
	}
	switch v := args[idx].(type) {
	case dsl.IdentityMode:
		if v == "" {
			return dsl.IdentityByDefault
		}
		return v
	case string:
		if mode, ok := identityModeLookup[v]; ok {
			return mode
		}
	}
	return dsl.IdentityByDefault
}

func argCascade(args []any, idx int) dsl.CascadeAction {
	if idx >= len(args) {
		return dsl.CascadeUnset
	}
	switch v := args[idx].(type) {
	case dsl.CascadeAction:
		return v
	case string:
		if v == "" {
			return dsl.CascadeUnset
		}
		return dsl.CascadeAction(strings.ToUpper(v))
	default:
		return dsl.CascadeUnset
	}
}

func argFieldType(args []any, idx int) (dsl.FieldType, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("missing field type argument")
	}
	switch v := args[idx].(type) {
	case dsl.FieldType:
		if v == "" {
			return "", fmt.Errorf("invalid field type")
		}
		return v, nil
	case string:
		if ft, ok := fieldTypeLookup[v]; ok {
			return ft, nil
		}
		if ft, ok := fieldTypeLookup[strings.TrimPrefix(v, "dsl.")]; ok {
			return ft, nil
		}
	}
	return "", fmt.Errorf("unsupported field type %v", args[idx])
}

var identityModeLookup = map[string]dsl.IdentityMode{
	"IdentityByDefault": dsl.IdentityByDefault,
	"IdentityAlways":    dsl.IdentityAlways,
}

func argSubscriptionEvent(args []any, idx int) (dsl.SubscriptionEvent, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("missing subscription event")
	}
	switch v := args[idx].(type) {
	case dsl.SubscriptionEvent:
		if v == "" {
			return "", fmt.Errorf("invalid subscription event")
		}
		return v, nil
	case string:
		if evt, ok := subscriptionEventLookup[v]; ok {
			return evt, nil
		}
		if evt, ok := subscriptionEventLookup[strings.TrimPrefix(v, "dsl.")]; ok {
			return evt, nil
		}
	}
	return "", fmt.Errorf("unsupported subscription event %v", args[idx])
}

var subscriptionEventLookup = map[string]dsl.SubscriptionEvent{
	"SubscriptionEventCreate": dsl.SubscriptionEventCreate,
	"SubscriptionEventUpdate": dsl.SubscriptionEventUpdate,
	"SubscriptionEventDelete": dsl.SubscriptionEventDelete,
	"create":                  dsl.SubscriptionEventCreate,
	"update":                  dsl.SubscriptionEventUpdate,
	"delete":                  dsl.SubscriptionEventDelete,
}

func argComparisonOperator(args []any, idx int) (dsl.ComparisonOperator, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("missing comparison operator")
	}
	switch v := args[idx].(type) {
	case dsl.ComparisonOperator:
		if v == "" {
			return "", fmt.Errorf("invalid comparison operator")
		}
		return v, nil
	case string:
		if op, ok := comparisonLookup[v]; ok {
			return op, nil
		}
		if op, ok := comparisonLookup[strings.TrimPrefix(v, "dsl.")]; ok {
			return op, nil
		}
	}
	return "", fmt.Errorf("unsupported comparison operator %v", args[idx])
}

func argSortDirection(args []any, idx int) (dsl.SortDirection, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("missing sort direction")
	}
	switch v := args[idx].(type) {
	case dsl.SortDirection:
		if v == "" {
			return "", fmt.Errorf("invalid sort direction")
		}
		return v, nil
	case string:
		if dir, ok := sortDirectionLookup[v]; ok {
			return dir, nil
		}
		if dir, ok := sortDirectionLookup[strings.TrimPrefix(v, "dsl.")]; ok {
			return dir, nil
		}
	}
	return "", fmt.Errorf("unsupported sort direction %v", args[idx])
}

func argAggregateFunc(args []any, idx int) (dsl.AggregateFunc, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("missing aggregate function")
	}
	switch v := args[idx].(type) {
	case dsl.AggregateFunc:
		if v == "" {
			return "", fmt.Errorf("invalid aggregate func")
		}
		return v, nil
	case string:
		if fn, ok := aggregateFuncLookup[v]; ok {
			return fn, nil
		}
		if fn, ok := aggregateFuncLookup[strings.TrimPrefix(v, "dsl.")]; ok {
			return fn, nil
		}
	}
	return "", fmt.Errorf("unsupported aggregate func %v", args[idx])
}

var fieldTypeLookup = map[string]dsl.FieldType{
	"TypeUUID":            dsl.TypeUUID,
	"TypeText":            dsl.TypeText,
	"TypeString":          dsl.TypeString,
	"TypeVarChar":         dsl.TypeVarChar,
	"TypeChar":            dsl.TypeChar,
	"TypeBoolean":         dsl.TypeBoolean,
	"TypeBool":            dsl.TypeBool,
	"TypeSmallInt":        dsl.TypeSmallInt,
	"TypeInteger":         dsl.TypeInteger,
	"TypeInt":             dsl.TypeInt,
	"TypeBigInt":          dsl.TypeBigInt,
	"TypeSmallSerial":     dsl.TypeSmallSerial,
	"TypeSerial":          dsl.TypeSerial,
	"TypeBigSerial":       dsl.TypeBigSerial,
	"TypeDecimal":         dsl.TypeDecimal,
	"TypeNumeric":         dsl.TypeNumeric,
	"TypeReal":            dsl.TypeReal,
	"TypeDoublePrecision": dsl.TypeDoublePrecision,
	"TypeFloat":           dsl.TypeFloat,
	"TypeMoney":           dsl.TypeMoney,
	"TypeBytea":           dsl.TypeBytea,
	"TypeBytes":           dsl.TypeBytes,
	"TypeDate":            dsl.TypeDate,
	"TypeTime":            dsl.TypeTime,
	"TypeTimeTZ":          dsl.TypeTimeTZ,
	"TypeTimestamp":       dsl.TypeTimestamp,
	"TypeTimestampTZ":     dsl.TypeTimestampTZ,
	"TypeInterval":        dsl.TypeInterval,
	"TypeJSON":            dsl.TypeJSON,
	"TypeJSONB":           dsl.TypeJSONB,
	"TypeXML":             dsl.TypeXML,
	"TypeInet":            dsl.TypeInet,
	"TypeCIDR":            dsl.TypeCIDR,
	"TypeMACAddr":         dsl.TypeMACAddr,
	"TypeMACAddr8":        dsl.TypeMACAddr8,
	"TypeBit":             dsl.TypeBit,
	"TypeVarBit":          dsl.TypeVarBit,
	"TypeTSVector":        dsl.TypeTSVector,
	"TypeTSQuery":         dsl.TypeTSQuery,
	"TypePoint":           dsl.TypePoint,
	"TypeLine":            dsl.TypeLine,
	"TypeLseg":            dsl.TypeLseg,
	"TypeBox":             dsl.TypeBox,
	"TypePath":            dsl.TypePath,
	"TypePolygon":         dsl.TypePolygon,
	"TypeCircle":          dsl.TypeCircle,
	"TypeInt4Range":       dsl.TypeInt4Range,
	"TypeInt8Range":       dsl.TypeInt8Range,
	"TypeNumRange":        dsl.TypeNumRange,
	"TypeTSRange":         dsl.TypeTSRange,
	"TypeTSTZRange":       dsl.TypeTSTZRange,
	"TypeDateRange":       dsl.TypeDateRange,
	"TypeArray":           dsl.TypeArray,
	"TypeGeometry":        dsl.TypeGeometry,
	"TypeGeography":       dsl.TypeGeography,
	"TypeVector":          dsl.TypeVector,
}

var comparisonLookup = map[string]dsl.ComparisonOperator{
	"OpEqual":       dsl.OpEqual,
	"OpNotEqual":    dsl.OpNotEqual,
	"OpGreaterThan": dsl.OpGreaterThan,
	"OpLessThan":    dsl.OpLessThan,
	"OpGTE":         dsl.OpGTE,
	"OpLTE":         dsl.OpLTE,
	"OpILike":       dsl.OpILike,
}

var sortDirectionLookup = map[string]dsl.SortDirection{
	"SortAsc":  dsl.SortAsc,
	"SortDesc": dsl.SortDesc,
}

var aggregateFuncLookup = map[string]dsl.AggregateFunc{
	"AggCount": dsl.AggCount,
	"AggSum":   dsl.AggSum,
	"AggAvg":   dsl.AggAvg,
	"AggMin":   dsl.AggMin,
	"AggMax":   dsl.AggMax,
}
