package game

import (
	"encoding/json"
	"math"
	"testing"
)

func TestIntegerKeepsLargeIntegerPrecisionAndRejectsOverflow(t *testing.T) {
	for _, input := range []any{int64(9007199254740993), json.Number("9007199254740993"), uint64(9007199254740993), "9007199254740993"} {
		if got := integer(input, 0); got != 9007199254740993 {
			t.Fatalf("整数经浮点数转换失真：got=%d", got)
		}
	}
	if got := integer(int64(math.MaxInt64), 0); got != math.MaxInt64 {
		t.Fatal("int64 最大值失真")
	}
	for _, input := range []any{uint64(math.MaxUint64), json.Number("9223372036854775808"), math.Inf(1), math.NaN()} {
		if got := integer(input, 7); got != 7 {
			t.Fatalf("超范围值未按原函数回退：got=%d", got)
		}
	}
}
