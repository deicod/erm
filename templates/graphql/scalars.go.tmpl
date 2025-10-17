package graphql

import (
        "context"
        "encoding/json"
        "io"
        "time"

        gql "github.com/99designs/gqlgen/graphql"
        "github.com/vektah/gqlparser/v2/ast"
)

func cloneRawJSON(input json.RawMessage) json.RawMessage {
        if input == nil {
                return nil
        }
        return append(json.RawMessage(nil), input...)
}

func unmarshalRawJSON(v any) (json.RawMessage, error) {
        switch value := v.(type) {
        case nil:
                return nil, nil
        case json.RawMessage:
                return cloneRawJSON(value), nil
        case *json.RawMessage:
                if value == nil {
                        return nil, nil
                }
                return cloneRawJSON(*value), nil
        case []byte:
                if value == nil {
                        return nil, nil
                }
                return json.RawMessage(append([]byte(nil), value...)), nil
        }
        bytes, err := json.Marshal(v)
        if err != nil {
                return nil, err
        }
        return json.RawMessage(bytes), nil
}

func marshalRawJSON(v json.RawMessage) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.WriterFunc(func(w io.Writer) {
                _, _ = w.Write(v)
        })
}

func (ec *executionContext) unmarshalInputBoolean(ctx context.Context, v any) (bool, error) {
        return gql.UnmarshalBoolean(v)
}

func (ec *executionContext) _Boolean(ctx context.Context, sel ast.SelectionSet, v *bool) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalBoolean(*v)
}

func (ec *executionContext) unmarshalInputFloat(ctx context.Context, v any) (float64, error) {
        return gql.UnmarshalFloat(v)
}

func (ec *executionContext) _Float(ctx context.Context, sel ast.SelectionSet, v *float64) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalFloat(*v)
}

func (ec *executionContext) unmarshalInputID(ctx context.Context, v any) (string, error) {
        id, err := gql.UnmarshalID(v)
        if err != nil {
                return "", err
        }
        return string(id), nil
}

func (ec *executionContext) _ID(ctx context.Context, sel ast.SelectionSet, v *string) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalID(*v)
}

func (ec *executionContext) unmarshalInputInt(ctx context.Context, v any) (int, error) {
        return gql.UnmarshalInt(v)
}

func (ec *executionContext) _Int(ctx context.Context, sel ast.SelectionSet, v *int) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalInt(*v)
}

func (ec *executionContext) unmarshalInputString(ctx context.Context, v any) (string, error) {
        return gql.UnmarshalString(v)
}

func (ec *executionContext) _String(ctx context.Context, sel ast.SelectionSet, v *string) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalString(*v)
}

func (ec *executionContext) unmarshalInput__DirectiveLocation(ctx context.Context, v any) (string, error) {
        return gql.UnmarshalString(v)
}

func (ec *executionContext) ___DirectiveLocation(ctx context.Context, sel ast.SelectionSet, v *string) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalString(*v)
}

func (ec *executionContext) unmarshalInput__TypeKind(ctx context.Context, v any) (string, error) {
        return gql.UnmarshalString(v)
}

func (ec *executionContext) ___TypeKind(ctx context.Context, sel ast.SelectionSet, v *string) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalString(*v)
}

func (ec *executionContext) unmarshalInputTime(ctx context.Context, v any) (time.Time, error) {
        return gql.UnmarshalTime(v)
}

func (ec *executionContext) _Time(ctx context.Context, sel ast.SelectionSet, v *time.Time) gql.Marshaler {
	if v == nil {
		return gql.Null
	}
	return gql.MarshalTime(*v)
}

func (ec *executionContext) unmarshalInputTimestamptz(ctx context.Context, v any) (time.Time, error) {
        return gql.UnmarshalTime(v)
}

func (ec *executionContext) _Timestamptz(ctx context.Context, sel ast.SelectionSet, v *time.Time) gql.Marshaler {
        if v == nil {
                return gql.Null
        }
        return gql.MarshalTime(*v)
}

func (ec *executionContext) unmarshalInputJSON(ctx context.Context, v any) (json.RawMessage, error) {
        return unmarshalRawJSON(v)
}

func (ec *executionContext) _JSON(ctx context.Context, sel ast.SelectionSet, v json.RawMessage) gql.Marshaler {
        _ = sel
        return marshalRawJSON(v)
}

func (ec *executionContext) unmarshalInputJSONB(ctx context.Context, v any) (json.RawMessage, error) {
        return unmarshalRawJSON(v)
}

func (ec *executionContext) _JSONB(ctx context.Context, sel ast.SelectionSet, v json.RawMessage) gql.Marshaler {
        _ = sel
        return marshalRawJSON(v)
}
