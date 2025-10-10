package validation

import (
	"errors"
	"strings"
)

// FieldError represents a validation failure scoped to a specific field.
type FieldError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e FieldError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Field
	}
	return e.Field + ": " + e.Message
}

// Errors aggregates multiple field errors.
type Errors []FieldError

// Error implements the error interface.
func (errs Errors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, len(errs))
	for i, err := range errs {
		parts[i] = err.Error()
	}
	return strings.Join(parts, "; ")
}

func appendErrors(dst Errors, err error) Errors {
	if err == nil {
		return dst
	}
	switch v := err.(type) {
	case Errors:
		return append(dst, v...)
	case *Errors:
		return append(dst, (*v)...)
	case FieldError:
		return append(dst, v)
	case *FieldError:
		return append(dst, *v)
	}
	var ferr FieldError
	if errors.As(err, &ferr) {
		return append(dst, ferr)
	}
	var multi Errors
	if errors.As(err, &multi) {
		return append(dst, multi...)
	}
	return append(dst, FieldError{Message: err.Error()})
}
