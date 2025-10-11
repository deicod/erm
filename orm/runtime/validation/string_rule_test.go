package validation

import (
	"context"
	"regexp"
	"testing"
)

func TestStringRuleBuilder(t *testing.T) {
	rule := String("Email").Required().MinLen(3).MaxLen(6).Matches(regexp.MustCompile(`^[a-z]+$`)).Rule()
	subj := Subject{Record: Record{"Email": "ab"}}
	if err := rule.Validate(context.Background(), subj); err == nil {
		t.Fatalf("expected error for short value")
	}
	subj.Record["Email"] = "toolong"
	if err := rule.Validate(context.Background(), subj); err == nil {
		t.Fatalf("expected error for long value")
	}
	subj.Record["Email"] = "abc1"
	if err := rule.Validate(context.Background(), subj); err == nil {
		t.Fatalf("expected regex failure")
	}
	subj.Record["Email"] = "abc"
	if err := rule.Validate(context.Background(), subj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStringRuleOptional(t *testing.T) {
	rule := String("Name").Optional().MinLen(2).Rule()
	subj := Subject{Record: Record{"Name": ""}}
	if err := rule.Validate(context.Background(), subj); err != nil {
		t.Fatalf("expected empty optional to pass: %v", err)
	}
	subj.Record["Name"] = "a"
	if err := rule.Validate(context.Background(), subj); err == nil {
		t.Fatalf("expected min length error")
	}
}
