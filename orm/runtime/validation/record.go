package validation

import "time"

// Get retrieves the raw value for the provided field.
func (r Record) Get(field string) (any, bool) {
	if r == nil {
		return nil, false
	}
	v, ok := r[field]
	return v, ok
}

// String extracts a string value, unwrapping pointer types when possible.
func (r Record) String(field string) (string, bool) {
	v, ok := r.Get(field)
	if !ok {
		return "", false
	}
	switch val := v.(type) {
	case string:
		return val, true
	case *string:
		if val == nil {
			return "", false
		}
		return *val, true
	default:
		return "", false
	}
}

// Time extracts a time.Time value, unwrapping pointer types when available.
func (r Record) Time(field string) (time.Time, bool) {
	v, ok := r.Get(field)
	if !ok {
		return time.Time{}, false
	}
	switch val := v.(type) {
	case time.Time:
		return val, true
	case *time.Time:
		if val == nil {
			return time.Time{}, false
		}
		return *val, true
	default:
		return time.Time{}, false
	}
}

// Has reports whether the field exists in the record map.
func (r Record) Has(field string) bool {
	_, ok := r.Get(field)
	return ok
}
