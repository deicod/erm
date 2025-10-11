package runtime

import "reflect"

// IsZeroValue reports whether the provided value is the zero value for its type.
func IsZeroValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	return rv.IsZero()
}
