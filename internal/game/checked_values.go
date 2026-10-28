package game

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
)

// checkedInteger 在任务进度与奖励路径拒绝损坏或超范围的数值，不将关键字段变成零值
func checkedInteger(value any) (int64, error) {
	if !truth(value) {
		return 0, nil
	}
	switch v := value.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case uint64:
		if v <= math.MaxInt64 {
			return int64(v), nil
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return n, nil
		}
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n, nil
		}
		if strings.ContainsAny(string(v), ".eE") {
			if n, err := v.Float64(); err == nil {
				return checkedFloatInteger(n)
			}
		}
	case float64:
		return checkedFloatInteger(v)
	}
	return 0, errors.New("数值类型无效或超出 int64 范围")
}

func checkedFloatInteger(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value >= 9223372036854775808.0 || value < -9223372036854775808.0 {
		return 0, errors.New("数值超出 int64 范围")
	}
	return int64(value), nil
}

func checkedAdd(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		return 0, errors.New("数值相加超出 int64 范围")
	}
	return a + b, nil
}

func checkedMultiply(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a == math.MinInt64 && b == -1 || b == math.MinInt64 && a == -1 {
		return 0, errors.New("数值相乘超出 int64 范围")
	}
	result := a * b
	if result/b != a {
		return 0, errors.New("数值相乘超出 int64 范围")
	}
	return result, nil
}
