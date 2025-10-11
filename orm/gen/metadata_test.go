package gen

import (
	"testing"

	"github.com/deicod/erm/orm/dsl"
	"github.com/deicod/erm/orm/runtime"
)

func TestRegistryStoresCascadeAndPolymorphicMetadata(t *testing.T) {
	reg := Registry
	if reg.Entities == nil {
		reg.Entities = map[string]runtime.EntitySpec{}
	}
	reg.Entities["Fixture"] = runtime.EntitySpec{
		Name:  "Fixture",
		Table: "fixtures",
		Edges: []runtime.EdgeSpec{
			{
				Name:   "subjects",
				Target: "Subject",
				Kind:   dsl.EdgeToMany,
				Cascade: runtime.CascadeSpec{
					OnDelete: runtime.CascadeCascade,
					OnUpdate: runtime.CascadeRestrict,
				},
				PolymorphicTargets: []runtime.EdgeTargetSpec{
					{Entity: "Subject", Condition: "kind = 'primary'"},
				},
			},
		},
	}

	spec, ok := reg.Entity("Fixture")
	if !ok {
		t.Fatalf("expected fixture entity to be registered")
	}
	if len(spec.Edges) != 1 {
		t.Fatalf("expected one edge, got %d", len(spec.Edges))
	}
	edge := spec.Edges[0]
	if edge.Cascade.OnDelete != runtime.CascadeCascade {
		t.Fatalf("unexpected cascade metadata: %+v", edge.Cascade)
	}
	if len(edge.PolymorphicTargets) == 0 {
		t.Fatalf("expected polymorphic target metadata to be preserved")
	}
}
