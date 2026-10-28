package protocol

import (
	"bytes"
	"math"
)

// Kind 表示 Python 服务端所用的 MessagePack 值子集。
type Kind uint8

const (
	KindNil Kind = iota
	KindBool
	KindInteger
	KindFloat
	KindString
	KindBinary
	KindArray
	KindObject
)

// Field 保留 MessagePack 映射的插入顺序。
// Python 字典保留顺序，Go 映射不保证顺序。
type Field struct {
	Key   Value
	Value Value
}

// Value 是明确的协议值。
// 整数使用符号和绝对值，保持 Python 单一整数类型。
type Value struct {
	Kind      Kind
	Boolean   bool
	Negative  bool
	Magnitude uint64
	Float64   float64
	Text      string
	Bytes     []byte
	Elements  []Value
	Fields    []Field
}

func Nil() Value {
	return Value{Kind: KindNil}
}

func Bool(value bool) Value {
	return Value{Kind: KindBool, Boolean: value}
}

func Int(value int64) Value {
	if value >= 0 {
		return Uint(uint64(value))
	}

	// 这种写法也能正确处理 math.MinInt64。
	magnitude := uint64(-(value + 1)) + 1
	return Value{Kind: KindInteger, Negative: true, Magnitude: magnitude}
}

func Uint(value uint64) Value {
	return Value{Kind: KindInteger, Magnitude: value}
}

func Float(value float64) Value {
	return Value{Kind: KindFloat, Float64: value}
}

func String(value string) Value {
	return Value{Kind: KindString, Text: value}
}

func Binary(value []byte) Value {
	return Value{Kind: KindBinary, Bytes: bytes.Clone(value)}
}

func Array(values ...Value) Value {
	return Value{Kind: KindArray, Elements: append([]Value(nil), values...)}
}

func Object(fields ...Field) Value {
	return Value{Kind: KindObject, Fields: append([]Field(nil), fields...)}
}

func Named(name string, value Value) Field {
	return Field{Key: String(name), Value: value}
}

// Equal 比较协议值，并检查对象字段顺序。
func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}

	switch v.Kind {
	case KindNil:
		return true
	case KindBool:
		return v.Boolean == other.Boolean
	case KindInteger:
		return v.Negative == other.Negative && v.Magnitude == other.Magnitude
	case KindFloat:
		return math.Float64bits(v.Float64) == math.Float64bits(other.Float64)
	case KindString:
		return v.Text == other.Text
	case KindBinary:
		return bytes.Equal(v.Bytes, other.Bytes)
	case KindArray:
		if len(v.Elements) != len(other.Elements) {
			return false
		}
		for i := range v.Elements {
			if !v.Elements[i].Equal(other.Elements[i]) {
				return false
			}
		}
		return true
	case KindObject:
		if len(v.Fields) != len(other.Fields) {
			return false
		}
		for i := range v.Fields {
			if !v.Fields[i].Key.Equal(other.Fields[i].Key) ||
				!v.Fields[i].Value.Equal(other.Fields[i].Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
