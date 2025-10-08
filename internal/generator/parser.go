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
	Name    string
	Fields  []dsl.Field
	Edges   []dsl.Edge
	Indexes []dsl.Index
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
	}
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
			return ent
		}
	}
	return Entity{Name: name, Fields: []dsl.Field{}, Edges: []dsl.Edge{}, Indexes: []dsl.Index{}}
}

func ensureDefaultField(ent *Entity) {
	for _, field := range ent.Fields {
		if strings.EqualFold(field.Name, "id") {
			return
		}
	}
	ent.Fields = append([]dsl.Field{dsl.UUIDv7("id").Primary()}, ent.Fields...)
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
	default:
		return nil, fmt.Errorf("unsupported call base %T", base)
	}
}

func executeDSLFunc(name string, args []any) (any, error) {
	switch name {
	case "UUIDv7":
		return dsl.UUIDv7(argString(args, 0)), nil
	case "String":
		return dsl.String(argString(args, 0)), nil
	case "Int":
		return dsl.Int(argString(args, 0)), nil
	case "Float":
		return dsl.Float(argString(args, 0)), nil
	case "Bool":
		return dsl.Bool(argString(args, 0)), nil
	case "Bytes":
		return dsl.Bytes(argString(args, 0)), nil
	case "Time":
		return dsl.Time(argString(args, 0)), nil
	case "JSON":
		return dsl.JSON(argString(args, 0)), nil
	case "ToOne":
		return dsl.ToOne(argString(args, 0), argString(args, 1)), nil
	case "ToMany":
		return dsl.ToMany(argString(args, 0), argString(args, 1)), nil
	case "ManyToMany":
		return dsl.ManyToMany(argString(args, 0), argString(args, 1)), nil
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
	default:
		return nil, fmt.Errorf("unsupported edge method %s", name)
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

func argString(args []any, idx int) string {
	if idx >= len(args) {
		return ""
	}
	if s, ok := args[idx].(string); ok {
		return s
	}
	return fmt.Sprint(args[idx])
}
