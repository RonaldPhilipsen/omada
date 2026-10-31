package models

// MapStr returns the string value at key, or "" when absent/not a string.
func MapStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// MapInt returns the numeric value at key as int, or 0 when absent.
// JSON numbers decode as float64; whole floats are converted safely.
func MapInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// MapBool returns the bool value at key, or false when absent.
func MapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// MapMap returns the nested object at key, ok=false when absent.
func MapMap(m map[string]any, key string) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}
	if v, ok := m[key].(map[string]any); ok {
		return v, true
	}
	return nil, false
}

// MapPtrBool returns a pointer to the bool at key, or nil when absent.
func MapPtrBool(m map[string]any, key string) *bool {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

// MapPtrInt returns a pointer to the numeric value at key, or nil when absent.
func MapPtrInt(m map[string]any, key string) *int {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	case int64:
		n := int(v)
		return &n
	}
	return nil
}

// MapPtrStr returns a pointer to the string at key, or nil when absent.
func MapPtrStr(m map[string]any, key string) *string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(string); ok {
		return &v
	}
	return nil
}

// MapPtrFloat returns a pointer to the numeric value at key, or nil when absent.
func MapPtrFloat(m map[string]any, key string) *float64 {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	}
	return nil
}

// MapStrList returns the list of strings at key, or nil when absent.
func MapStrList(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	list, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// MapList returns the list at key as []any, or nil when absent.
func MapList(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}
