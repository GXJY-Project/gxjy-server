package game

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type record = map[string]any

func obj(v any) record {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return record{}
}
func arr(v any) []any {
	if a, ok := v.([]any); ok && a != nil {
		return a
	}
	return []any{}
}
func truth(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case json.Number:
		return num(x, 0) != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	case uint64:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	default:
		return true
	}
}
func or(v, fallback any) any {
	if truth(v) {
		return v
	}
	return fallback
}
func str(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}
func text(v any) string { return str(or(v, "")) }
func num(v any, fallback float64) float64 {
	switch x := v.(type) {
	case bool:
		if x {
			return 1
		}
		return 0
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	case float64:
		return x
	case json.Number:
		n, e := x.Float64()
		if e == nil {
			return n
		}
	case string:
		n, e := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if e == nil {
			return n
		}
	}
	return fallback
}

// integer 对应 Python int，数字字符串不接受浮点形式
func integer(v any, fallback int64) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case uint64:
		if x <= math.MaxInt64 {
			return int64(x)
		}
		return fallback
	case float64:
		if n, err := checkedFloatInteger(x); err == nil {
			return n
		}
		return fallback
	case string:
		n, e := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if e == nil {
			return n
		}
		return fallback
	case json.Number:
		if n, e := x.Int64(); e == nil {
			return n
		}
		if strings.ContainsAny(string(x), ".eE") {
			if n, err := x.Float64(); err == nil {
				if value, err := checkedFloatInteger(n); err == nil {
					return value
				}
			}
		}
		return fallback
	default:
		return int64(num(v, float64(fallback)))
	}
}
func iv(v any) int64 { return integer(or(v, 0), 0) }
func copyValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		r := record{}
		for k, v := range x {
			r[k] = copyValue(v)
		}
		return r
	case []any:
		r := make([]any, len(x))
		for i, v := range x {
			r[i] = copyValue(v)
		}
		return r
	default:
		return v
	}
}
func ensure(m record, key string) record {
	if v, ok := m[key]; ok {
		return obj(v)
	}
	v := record{}
	m[key] = v
	return v
}
func merge(dst, src record) {
	for k, v := range src {
		dst[k] = copyValue(v)
	}
}
func missing(dst, src record) {
	for k, v := range src {
		if _, ok := dst[k]; !ok {
			dst[k] = copyValue(v)
		}
	}
}
