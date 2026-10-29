package controller

// mapStr returns the string value at key, or "" when absent/not a string.
func mapStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// mapInt returns the numeric value at key as int, or 0 when absent.
// JSON numbers decode as float64; whole floats are converted safely.
func mapInt(m map[string]any, key string) int {
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

// mapBool returns the bool value at key, or false when absent.
func mapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// mapMap returns the nested object at key, ok=false when absent.
func mapMap(m map[string]any, key string) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}
	if v, ok := m[key].(map[string]any); ok {
		return v, true
	}
	return nil, false
}

// mapPtrBool returns a pointer to the bool at key, or nil when absent.
func mapPtrBool(m map[string]any, key string) *bool {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

// mapPtrInt returns a pointer to the numeric value at key, or nil when absent.
func mapPtrInt(m map[string]any, key string) *int {
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

// mapPtrStr returns a pointer to the string at key, or nil when absent.
func mapPtrStr(m map[string]any, key string) *string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(string); ok {
		return &v
	}
	return nil
}

// mapPtrFloat returns a pointer to the numeric value at key, or nil when absent.
func mapPtrFloat(m map[string]any, key string) *float64 {
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

// mapStrList returns the list of strings at key, or nil when absent.
func mapStrList(m map[string]any, key string) []string {
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

// mapList returns the list at key as []any, or nil when absent.
func mapList(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}
