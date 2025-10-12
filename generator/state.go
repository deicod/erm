package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/deicod/erm/orm/dsl"
)

type ComponentName string

const (
	ComponentORM        ComponentName = "orm"
	ComponentGraphQL    ComponentName = "graphql"
	ComponentMigrations ComponentName = "migrations"
)

type generatorState struct {
	Components map[ComponentName]componentState `json:"components"`
}

type componentState struct {
	InputHash string `json:"input_hash"`
}

func loadGeneratorState(root string) (generatorState, error) {
	path := cachePath(root)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return generatorState{Components: make(map[ComponentName]componentState)}, nil
		}
		return generatorState{}, err
	}
	var state generatorState
	if err := json.Unmarshal(raw, &state); err != nil {
		return generatorState{}, err
	}
	if state.Components == nil {
		state.Components = make(map[ComponentName]componentState)
	}
	return state, nil
}

func saveGeneratorState(root string, state generatorState) error {
	path := cachePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func cachePath(root string) string {
	return filepath.Join(root, ".erm", "cache", "generator_state.json")
}

type schemaSignature struct {
	Entities []entitySignature `json:"entities"`
}

type entitySignature struct {
	Name    string           `json:"name"`
	Fields  []fieldSignature `json:"fields"`
	Edges   []edgeSignature  `json:"edges"`
	Indexes []indexSignature `json:"indexes"`
	Query   querySignature   `json:"query"`
}

type fieldSignature struct {
	Name          string             `json:"name"`
	Column        string             `json:"column"`
	GoType        string             `json:"go_type"`
	Type          string             `json:"type"`
	IsPrimary     bool               `json:"is_primary"`
	Nullable      bool               `json:"nullable"`
	IsUnique      bool               `json:"is_unique"`
	HasDefaultNow bool               `json:"has_default_now"`
	HasUpdateNow  bool               `json:"has_update_now"`
	DefaultExpr   string             `json:"default_expr"`
	Annotations   []annotationRecord `json:"annotations"`
	EnumValues    []string           `json:"enum_values"`
	EnumName      string             `json:"enum_name"`
}

type edgeSignature struct {
	Name               string             `json:"name"`
	Column             string             `json:"column"`
	RefName            string             `json:"ref_name"`
	Through            string             `json:"through"`
	Target             string             `json:"target"`
	Kind               string             `json:"kind"`
	Nullable           bool               `json:"nullable"`
	Unique             bool               `json:"unique"`
	Annotations        []annotationRecord `json:"annotations"`
	InverseName        string             `json:"inverse_name"`
	PolymorphicTargets []edgeTargetRecord `json:"polymorphic_targets"`
	Cascade            cascadeRecord      `json:"cascade"`
}

type indexSignature struct {
	Name             string             `json:"name"`
	Columns          []string           `json:"columns"`
	IsUnique         bool               `json:"is_unique"`
	Where            string             `json:"where"`
	Method           string             `json:"method"`
	NullsNotDistinct bool               `json:"nulls_not_distinct"`
	Annotations      []annotationRecord `json:"annotations"`
}

type querySignature struct {
	Predicates []predicateRecord `json:"predicates"`
	Orders     []orderRecord     `json:"orders"`
	Aggregates []aggregateRecord `json:"aggregates"`
	Default    int               `json:"default_limit"`
	Max        int               `json:"max_limit"`
}

type predicateRecord struct {
	Name  string `json:"name"`
	Field string `json:"field"`
	Op    string `json:"operator"`
}

type orderRecord struct {
	Name  string `json:"name"`
	Field string `json:"field"`
	Dir   string `json:"direction"`
}

type aggregateRecord struct {
	Name   string `json:"name"`
	Func   string `json:"func"`
	Field  string `json:"field"`
	GoType string `json:"go_type"`
}

type annotationRecord struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type edgeTargetRecord struct {
	Entity    string `json:"entity"`
	Condition string `json:"condition"`
}

type cascadeRecord struct {
	OnDelete string `json:"on_delete"`
	OnUpdate string `json:"on_update"`
}

func schemaInputHash(entities []Entity) (string, error) {
	sig := buildSchemaSignature(entities)
	payload, err := json.Marshal(sig)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func buildSchemaSignature(entities []Entity) schemaSignature {
	sig := schemaSignature{}
	if len(entities) == 0 {
		sig.Entities = []entitySignature{}
		return sig
	}
	sorted := append([]Entity(nil), entities...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	sig.Entities = make([]entitySignature, 0, len(sorted))
	for _, ent := range sorted {
		sig.Entities = append(sig.Entities, entitySignature{
			Name:    ent.Name,
			Fields:  encodeFields(ent.Fields),
			Edges:   encodeEdges(ent.Edges),
			Indexes: encodeIndexes(ent.Indexes),
			Query:   encodeQuery(ent.Query),
		})
	}
	return sig
}

func encodeFields(fields []dsl.Field) []fieldSignature {
	if len(fields) == 0 {
		return []fieldSignature{}
	}
	out := make([]fieldSignature, len(fields))
	sorted := append([]dsl.Field(nil), fields...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for i, field := range sorted {
		out[i] = fieldSignature{
			Name:          field.Name,
			Column:        field.Column,
			GoType:        field.GoType,
			Type:          string(field.Type),
			IsPrimary:     field.IsPrimary,
			Nullable:      field.Nullable,
			IsUnique:      field.IsUnique,
			HasDefaultNow: field.HasDefaultNow,
			HasUpdateNow:  field.HasUpdateNow,
			DefaultExpr:   field.DefaultExpr,
			Annotations:   encodeAnnotations(field.Annotations),
			EnumValues:    append([]string(nil), field.EnumValues...),
			EnumName:      field.EnumName,
		}
	}
	return out
}

func encodeEdges(edges []dsl.Edge) []edgeSignature {
	if len(edges) == 0 {
		return []edgeSignature{}
	}
	out := make([]edgeSignature, len(edges))
	sorted := append([]dsl.Edge(nil), edges...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for i, edge := range sorted {
		out[i] = edgeSignature{
			Name:               edge.Name,
			Column:             edge.Column,
			RefName:            edge.RefName,
			Through:            edge.Through,
			Target:             edge.Target,
			Kind:               string(edge.Kind),
			Nullable:           edge.Nullable,
			Unique:             edge.Unique,
			Annotations:        encodeAnnotations(edge.Annotations),
			InverseName:        edge.InverseName,
			PolymorphicTargets: encodeEdgeTargets(edge.PolymorphicTargets),
			Cascade: cascadeRecord{
				OnDelete: string(edge.Cascade.OnDelete),
				OnUpdate: string(edge.Cascade.OnUpdate),
			},
		}
	}
	return out
}

func encodeIndexes(indexes []dsl.Index) []indexSignature {
	if len(indexes) == 0 {
		return []indexSignature{}
	}
	out := make([]indexSignature, len(indexes))
	sorted := append([]dsl.Index(nil), indexes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for i, idx := range sorted {
		cols := append([]string(nil), idx.Columns...)
		sort.Strings(cols)
		out[i] = indexSignature{
			Name:             idx.Name,
			Columns:          cols,
			IsUnique:         idx.IsUnique,
			Where:            idx.Where,
			Method:           idx.Method,
			NullsNotDistinct: idx.NullsNotDistinct,
			Annotations:      encodeAnnotations(idx.Annotations),
		}
	}
	return out
}

func encodeQuery(q dsl.QuerySpec) querySignature {
	return querySignature{
		Predicates: encodePredicates(q.Predicates),
		Orders:     encodeOrders(q.Orders),
		Aggregates: encodeAggregates(q.Aggregates),
		Default:    q.DefaultLimit,
		Max:        q.MaxLimit,
	}
}

func encodePredicates(preds []dsl.Predicate) []predicateRecord {
	if len(preds) == 0 {
		return []predicateRecord{}
	}
	out := make([]predicateRecord, len(preds))
	sorted := append([]dsl.Predicate(nil), preds...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name == sorted[j].Name {
			return sorted[i].Field < sorted[j].Field
		}
		return sorted[i].Name < sorted[j].Name
	})
	for i, p := range sorted {
		out[i] = predicateRecord{Name: p.Name, Field: p.Field, Op: string(p.Operator)}
	}
	return out
}

func encodeOrders(orders []dsl.Order) []orderRecord {
	if len(orders) == 0 {
		return []orderRecord{}
	}
	out := make([]orderRecord, len(orders))
	sorted := append([]dsl.Order(nil), orders...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name == sorted[j].Name {
			return sorted[i].Field < sorted[j].Field
		}
		return sorted[i].Name < sorted[j].Name
	})
	for i, order := range sorted {
		out[i] = orderRecord{Name: order.Name, Field: order.Field, Dir: string(order.Direction)}
	}
	return out
}

func encodeAggregates(aggregates []dsl.Aggregate) []aggregateRecord {
	if len(aggregates) == 0 {
		return []aggregateRecord{}
	}
	out := make([]aggregateRecord, len(aggregates))
	sorted := append([]dsl.Aggregate(nil), aggregates...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name == sorted[j].Name {
			return sorted[i].Field < sorted[j].Field
		}
		return sorted[i].Name < sorted[j].Name
	})
	for i, agg := range sorted {
		out[i] = aggregateRecord{Name: agg.Name, Func: string(agg.Func), Field: agg.Field, GoType: agg.GoType}
	}
	return out
}

func encodeAnnotations(annotations map[string]any) []annotationRecord {
	if len(annotations) == 0 {
		return []annotationRecord{}
	}
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]annotationRecord, 0, len(keys))
	for _, key := range keys {
		val := annotations[key]
		payload, err := json.Marshal(val)
		if err != nil {
			payload = []byte(fmt.Sprintf("%v", val))
		}
		out = append(out, annotationRecord{Key: key, Value: string(payload)})
	}
	return out
}

func encodeEdgeTargets(targets []dsl.EdgeTarget) []edgeTargetRecord {
	if len(targets) == 0 {
		return []edgeTargetRecord{}
	}
	out := make([]edgeTargetRecord, len(targets))
	sorted := append([]dsl.EdgeTarget(nil), targets...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Entity == sorted[j].Entity {
			return sorted[i].Condition < sorted[j].Condition
		}
		return sorted[i].Entity < sorted[j].Entity
	})
	for i, target := range sorted {
		out[i] = edgeTargetRecord{Entity: target.Entity, Condition: target.Condition}
	}
	return out
}

func componentInputHash(base string, component ComponentName) string {
	sum := sha256.Sum256([]byte(base + ":" + string(component)))
	return hex.EncodeToString(sum[:])
}
