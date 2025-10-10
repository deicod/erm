package testkit

import "testing"

func TestSandboxSmoke(t *testing.T) {
	sandbox := NewPostgresSandbox(t)
	if sandbox == nil {
		t.Fatalf("expected sandbox to be initialised")
	}
	if sandbox.ORM(t) == nil {
		t.Fatalf("expected ORM client to be initialised")
	}
	sandbox.ExpectationsWereMet(t)
}
