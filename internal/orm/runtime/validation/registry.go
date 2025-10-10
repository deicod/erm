package validation

import (
	"context"
	"sync"
)

// Operation identifies which ORM mutation triggered validation.
type Operation string

const (
	// OpCreate runs before inserting a new entity.
	OpCreate Operation = "create"
	// OpUpdate runs before updating an existing entity.
	OpUpdate Operation = "update"
)

// Record captures the projected field values for a mutation.
type Record map[string]any

// Subject exposes contextual information to validation rules.
type Subject struct {
	Entity    string
	Operation Operation
	Record    Record
	Input     any
}

// Rule represents a validation constraint applied to an entity mutation.
type Rule interface {
	Validate(context.Context, Subject) error
}

// RuleFunc adapts a plain function into a validation Rule.
type RuleFunc func(context.Context, Subject) error

// Validate invokes the underlying function.
func (fn RuleFunc) Validate(ctx context.Context, subject Subject) error {
	return fn(ctx, subject)
}

// Registry stores the registered validation rules per entity.
type Registry struct {
	mu       sync.RWMutex
	entities map[string]*entityRules
}

// NewRegistry constructs an empty validation registry.
func NewRegistry() *Registry {
	return &Registry{entities: make(map[string]*entityRules)}
}

// Entity returns the ruleset for the provided entity name, creating it on demand.
func (r *Registry) Entity(name string) *EntityRules {
	if name == "" {
		return &EntityRules{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entities == nil {
		r.entities = make(map[string]*entityRules)
	}
	rules, ok := r.entities[name]
	if !ok {
		rules = &entityRules{}
		r.entities[name] = rules
	}
	return &EntityRules{inner: rules}
}

// Validate executes all rules registered for the entity/operation pair.
// Multiple rule violations are aggregated into a single error value.
func (r *Registry) Validate(ctx context.Context, entity string, op Operation, record Record, input any) error {
	if entity == "" {
		return nil
	}
	r.mu.RLock()
	rules := r.entities[entity]
	r.mu.RUnlock()
	if rules == nil {
		return nil
	}
	subject := Subject{Entity: entity, Operation: op, Record: record, Input: input}
	entries := rules.snapshot(op)
	if len(entries) == 0 {
		return nil
	}
	var errs Errors
	for _, rule := range entries {
		if err := rule.Validate(ctx, subject); err != nil {
			errs = appendErrors(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// EntityRules exposes fluent helpers for registering rules on a single entity.
type EntityRules struct {
	inner *entityRules
}

// OnCreate appends rules executed before inserts.
func (r *EntityRules) OnCreate(rules ...Rule) *EntityRules {
	if r == nil || r.inner == nil {
		return r
	}
	r.inner.add(OpCreate, rules...)
	return r
}

// OnUpdate appends rules executed before updates.
func (r *EntityRules) OnUpdate(rules ...Rule) *EntityRules {
	if r == nil || r.inner == nil {
		return r
	}
	r.inner.add(OpUpdate, rules...)
	return r
}

type entityRules struct {
	mu     sync.RWMutex
	create []Rule
	update []Rule
}

func (r *entityRules) add(op Operation, rules ...Rule) {
	if len(rules) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch op {
	case OpCreate:
		r.create = append(r.create, rules...)
	case OpUpdate:
		r.update = append(r.update, rules...)
	}
}

func (r *entityRules) snapshot(op Operation) []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var src []Rule
	switch op {
	case OpCreate:
		src = r.create
	case OpUpdate:
		src = r.update
	}
	if len(src) == 0 {
		return nil
	}
	out := make([]Rule, len(src))
	copy(out, src)
	return out
}
