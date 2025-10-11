package validation

import (
	"context"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// String builds a string-specific validation rule for the provided field name.
func String(field string) *StringRuleBuilder {
	return &StringRuleBuilder{field: field}
}

// StringRuleBuilder provides fluent helpers for describing string constraints.
type StringRuleBuilder struct {
	field      string
	required   bool
	minLen     int
	hasMin     bool
	maxLen     int
	hasMax     bool
	pattern    *regexp.Regexp
	allowEmpty bool
}

// Required enforces that the field must be present and non-empty.
func (b *StringRuleBuilder) Required() *StringRuleBuilder {
	b.required = true
	return b
}

// Optional marks the field as optional, even if other constraints are present.
func (b *StringRuleBuilder) Optional() *StringRuleBuilder {
	b.required = false
	return b
}

// MinLen enforces a minimum rune length when the field is provided.
func (b *StringRuleBuilder) MinLen(n int) *StringRuleBuilder {
	if n < 0 {
		n = 0
	}
	b.hasMin = true
	b.minLen = n
	return b
}

// MaxLen enforces a maximum rune length when the field is provided.
func (b *StringRuleBuilder) MaxLen(n int) *StringRuleBuilder {
	if n < 0 {
		n = 0
	}
	b.hasMax = true
	b.maxLen = n
	return b
}

// AllowEmpty treats empty strings as valid when the field is otherwise optional.
func (b *StringRuleBuilder) AllowEmpty() *StringRuleBuilder {
	b.allowEmpty = true
	return b
}

// Matches applies the provided regular expression to non-empty values.
func (b *StringRuleBuilder) Matches(re *regexp.Regexp) *StringRuleBuilder {
	b.pattern = re
	return b
}

// Rule materializes the builder into a concrete validation Rule.
func (b *StringRuleBuilder) Rule() Rule {
	field := b.field
	required := b.required
	hasMin, minLen := b.hasMin, b.minLen
	hasMax, maxLen := b.hasMax, b.maxLen
	pattern := b.pattern
	allowEmpty := b.allowEmpty
	return RuleFunc(func(_ context.Context, subject Subject) error {
		if field == "" {
			return nil
		}
		raw, ok := subject.Record.Get(field)
		if !ok {
			if required {
				return FieldError{Field: field, Message: "is required"}
			}
			return nil
		}
		var value string
		switch v := raw.(type) {
		case string:
			value = v
		case *string:
			if v == nil {
				if required {
					return FieldError{Field: field, Message: "is required"}
				}
				return nil
			}
			value = *v
		default:
			return FieldError{Field: field, Message: "must be a string"}
		}
		if value == "" {
			if required {
				return FieldError{Field: field, Message: "cannot be empty"}
			}
			if !allowEmpty {
				return nil
			}
		}
		if value == "" && allowEmpty {
			return nil
		}
		length := utf8.RuneCountInString(value)
		var errs Errors
		if hasMin && length < minLen {
			errs = append(errs, FieldError{Field: field, Message: fmt.Sprintf("must be at least %d characters", minLen)})
		}
		if hasMax && length > maxLen {
			errs = append(errs, FieldError{Field: field, Message: fmt.Sprintf("must be at most %d characters", maxLen)})
		}
		if pattern != nil && !pattern.MatchString(value) {
			errs = append(errs, FieldError{Field: field, Message: "is invalid"})
		}
		if len(errs) > 0 {
			return errs
		}
		return nil
	})
}
