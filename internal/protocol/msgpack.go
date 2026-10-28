package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidValue     = errors.New("invalid protocol value")
	ErrUnsupportedValue = errors.New("unsupported MessagePack value")
	ErrNestingTooDeep   = errors.New("MessagePack nesting too deep")
)

const maxMessagePackDepth = 128

// MarshalMessagePack 编码 Python mp_pack 支持的值子集。
func MarshalMessagePack(value Value) ([]byte, error) {
	out := make([]byte, 0, 128)
	return appendMessagePack(out, value, 0)
}

func appendMessagePack(out []byte, value Value, depth int) ([]byte, error) {
	if depth > maxMessagePackDepth {
		return nil, ErrNestingTooDeep
	}

	switch value.Kind {
	case KindNil:
		return append(out, 0xc0), nil
	case KindBool:
		if value.Boolean {
			return append(out, 0xc3), nil
		}
		return append(out, 0xc2), nil
	case KindInteger:
		return appendInteger(out, value)
	case KindFloat:
		out = append(out, 0xcb)
		return binary.BigEndian.AppendUint64(out, math.Float64bits(value.Float64)), nil
	case KindString:
		if !utf8.ValidString(value.Text) {
			return nil, fmt.Errorf("%w: string is not UTF-8", ErrInvalidValue)
		}
		out, err := appendStringHeader(out, len(value.Text))
		if err != nil {
			return nil, err
		}
		return append(out, value.Text...), nil
	case KindBinary:
		out, err := appendBinaryHeader(out, len(value.Bytes))
		if err != nil {
			return nil, err
		}
		return append(out, value.Bytes...), nil
	case KindArray:
		out, err := appendArrayHeader(out, len(value.Elements))
		if err != nil {
			return nil, err
		}
		for _, element := range value.Elements {
			out, err = appendMessagePack(out, element, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case KindObject:
		out, err := appendObjectHeader(out, len(value.Fields))
		if err != nil {
			return nil, err
		}
		for _, field := range value.Fields {
			if !messagePackMapKey(field.Key) {
				return nil, fmt.Errorf("%w: map key kind %d", ErrInvalidValue, field.Key.Kind)
			}
			out, err = appendMessagePack(out, field.Key, depth+1)
			if err != nil {
				return nil, err
			}
			out, err = appendMessagePack(out, field.Value, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: kind %d", ErrUnsupportedValue, value.Kind)
	}
}

func appendInteger(out []byte, value Value) ([]byte, error) {
	if !value.Negative {
		n := value.Magnitude
		switch {
		case n <= 0x7f:
			return append(out, byte(n)), nil
		case n <= 0xff:
			return append(out, 0xcc, byte(n)), nil
		case n <= 0xffff:
			out = append(out, 0xcd)
			return binary.BigEndian.AppendUint16(out, uint16(n)), nil
		case n <= 0xffffffff:
			out = append(out, 0xce)
			return binary.BigEndian.AppendUint32(out, uint32(n)), nil
		default:
			out = append(out, 0xcf)
			return binary.BigEndian.AppendUint64(out, n), nil
		}
	}

	if value.Magnitude == 0 || value.Magnitude > 1<<63 {
		return nil, fmt.Errorf("%w: negative magnitude %d", ErrInvalidValue, value.Magnitude)
	}

	var signed int64
	if value.Magnitude == 1<<63 {
		signed = math.MinInt64
	} else {
		signed = -int64(value.Magnitude)
	}

	switch {
	case signed >= -32:
		return append(out, byte(int8(signed))), nil
	case signed >= math.MinInt8:
		return append(out, 0xd0, byte(int8(signed))), nil
	case signed >= math.MinInt16:
		out = append(out, 0xd1)
		return binary.BigEndian.AppendUint16(out, uint16(int16(signed))), nil
	case signed >= math.MinInt32:
		out = append(out, 0xd2)
		return binary.BigEndian.AppendUint32(out, uint32(int32(signed))), nil
	default:
		out = append(out, 0xd3)
		return binary.BigEndian.AppendUint64(out, uint64(signed)), nil
	}
}

func appendStringHeader(out []byte, size int) ([]byte, error) {
	switch {
	case size <= 31:
		return append(out, 0xa0|byte(size)), nil
	case size <= 0xff:
		return append(out, 0xd9, byte(size)), nil
	case size <= 0xffff:
		out = append(out, 0xda)
		return binary.BigEndian.AppendUint16(out, uint16(size)), nil
	case uint64(size) <= 0xffffffff:
		out = append(out, 0xdb)
		return binary.BigEndian.AppendUint32(out, uint32(size)), nil
	default:
		return nil, fmt.Errorf("%w: string length %d", ErrInvalidValue, size)
	}
}

func appendBinaryHeader(out []byte, size int) ([]byte, error) {
	switch {
	case size <= 0xff:
		return append(out, 0xc4, byte(size)), nil
	case size <= 0xffff:
		out = append(out, 0xc5)
		return binary.BigEndian.AppendUint16(out, uint16(size)), nil
	case uint64(size) <= 0xffffffff:
		out = append(out, 0xc6)
		return binary.BigEndian.AppendUint32(out, uint32(size)), nil
	default:
		return nil, fmt.Errorf("%w: binary length %d", ErrInvalidValue, size)
	}
}

func appendArrayHeader(out []byte, size int) ([]byte, error) {
	switch {
	case size <= 15:
		return append(out, 0x90|byte(size)), nil
	case size <= 0xffff:
		out = append(out, 0xdc)
		return binary.BigEndian.AppendUint16(out, uint16(size)), nil
	case uint64(size) <= 0xffffffff:
		out = append(out, 0xdd)
		return binary.BigEndian.AppendUint32(out, uint32(size)), nil
	default:
		return nil, fmt.Errorf("%w: array length %d", ErrInvalidValue, size)
	}
}

func appendObjectHeader(out []byte, size int) ([]byte, error) {
	switch {
	case size <= 15:
		return append(out, 0x80|byte(size)), nil
	case size <= 0xffff:
		out = append(out, 0xde)
		return binary.BigEndian.AppendUint16(out, uint16(size)), nil
	case uint64(size) <= 0xffffffff:
		out = append(out, 0xdf)
		return binary.BigEndian.AppendUint32(out, uint32(size)), nil
	default:
		return nil, fmt.Errorf("%w: map length %d", ErrInvalidValue, size)
	}
}

// UnmarshalMessagePack 从指定偏移解码一个值，并返回下一偏移。
func UnmarshalMessagePack(data []byte, offset int) (Value, int, error) {
	if offset < 0 || offset > len(data) {
		return Value{}, offset, ErrInvalidOffset
	}
	return decodeMessagePack(data, offset, 0)
}

func decodeMessagePack(data []byte, offset int, depth int) (Value, int, error) {
	if depth > maxMessagePackDepth {
		return Value{}, offset, ErrNestingTooDeep
	}
	if offset >= len(data) {
		return Value{}, offset, fmt.Errorf("%w at offset %d", ErrTruncated, offset)
	}

	marker := data[offset]
	offset++

	switch {
	case marker <= 0x7f:
		return Uint(uint64(marker)), offset, nil
	case marker >= 0xe0:
		return Int(int64(int8(marker))), offset, nil
	case marker >= 0x80 && marker <= 0x8f:
		return decodeObject(data, offset, int(marker&0x0f), depth)
	case marker >= 0x90 && marker <= 0x9f:
		return decodeArray(data, offset, int(marker&0x0f), depth)
	case marker >= 0xa0 && marker <= 0xbf:
		return decodeString(data, offset, uint64(marker&0x1f))
	}

	switch marker {
	case 0xc0:
		return Nil(), offset, nil
	case 0xc2:
		return Bool(false), offset, nil
	case 0xc3:
		return Bool(true), offset, nil
	case 0xc4:
		size, next, err := readUint8(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeBinary(data, next, uint64(size))
	case 0xc5:
		size, next, err := readUint16(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeBinary(data, next, uint64(size))
	case 0xc6:
		size, next, err := readUint32(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeBinary(data, next, uint64(size))
	case 0xca:
		bits, next, err := readUint32(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return Float(float64(math.Float32frombits(bits))), next, nil
	case 0xcb:
		bits, next, err := readUint64(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return Float(math.Float64frombits(bits)), next, nil
	case 0xcc:
		n, next, err := readUint8(data, offset)
		return Uint(uint64(n)), next, err
	case 0xcd:
		n, next, err := readUint16(data, offset)
		return Uint(uint64(n)), next, err
	case 0xce:
		n, next, err := readUint32(data, offset)
		return Uint(uint64(n)), next, err
	case 0xcf:
		n, next, err := readUint64(data, offset)
		return Uint(n), next, err
	case 0xd0:
		n, next, err := readUint8(data, offset)
		return Int(int64(int8(n))), next, err
	case 0xd1:
		n, next, err := readUint16(data, offset)
		return Int(int64(int16(n))), next, err
	case 0xd2:
		n, next, err := readUint32(data, offset)
		return Int(int64(int32(n))), next, err
	case 0xd3:
		n, next, err := readUint64(data, offset)
		return Int(int64(n)), next, err
	case 0xd9:
		size, next, err := readUint8(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeString(data, next, uint64(size))
	case 0xda:
		size, next, err := readUint16(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeString(data, next, uint64(size))
	case 0xdb:
		size, next, err := readUint32(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeString(data, next, uint64(size))
	case 0xdc:
		size, next, err := readUint16(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeArray(data, next, int(size), depth)
	case 0xdd:
		size, next, err := readUint32(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		if uint64(size) > uint64(len(data)-next) {
			return Value{}, next, fmt.Errorf("%w: array length %d", ErrTruncated, size)
		}
		return decodeArray(data, next, int(size), depth)
	case 0xde:
		size, next, err := readUint16(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		return decodeObject(data, next, int(size), depth)
	case 0xdf:
		size, next, err := readUint32(data, offset)
		if err != nil {
			return Value{}, next, err
		}
		if uint64(size) > uint64(len(data)-next)/2 {
			return Value{}, next, fmt.Errorf("%w: map length %d", ErrTruncated, size)
		}
		return decodeObject(data, next, int(size), depth)
	default:
		return Value{}, offset, fmt.Errorf("%w: marker 0x%02x at %d", ErrUnsupportedValue, marker, offset-1)
	}
}

func decodeArray(data []byte, offset, size, depth int) (Value, int, error) {
	if size > len(data)-offset {
		return Value{}, offset, fmt.Errorf("%w: array length %d", ErrTruncated, size)
	}

	values := make([]Value, 0, size)
	for range size {
		value, next, err := decodeMessagePack(data, offset, depth+1)
		if err != nil {
			return Value{}, next, err
		}
		values = append(values, value)
		offset = next
	}
	return Value{Kind: KindArray, Elements: values}, offset, nil
}

func decodeObject(data []byte, offset, size, depth int) (Value, int, error) {
	if size > (len(data)-offset)/2 {
		return Value{}, offset, fmt.Errorf("%w: map length %d", ErrTruncated, size)
	}

	fields := make([]Field, 0, size)
	for range size {
		key, next, err := decodeMessagePack(data, offset, depth+1)
		if err != nil {
			return Value{}, next, err
		}
		if !messagePackMapKey(key) {
			return Value{}, next, fmt.Errorf("%w: map key kind %d", ErrInvalidValue, key.Kind)
		}
		value, next, err := decodeMessagePack(data, next, depth+1)
		if err != nil {
			return Value{}, next, err
		}
		fields = setObjectField(fields, Field{Key: key, Value: value})
		offset = next
	}
	return Value{Kind: KindObject, Fields: fields}, offset, nil
}

func decodeString(data []byte, offset int, size uint64) (Value, int, error) {
	chunk, next, err := readBytes(data, offset, size)
	if err != nil {
		return Value{}, next, err
	}
	text := string(chunk)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "\uFFFD")
	}
	return String(text), next, nil
}

func decodeBinary(data []byte, offset int, size uint64) (Value, int, error) {
	chunk, next, err := readBytes(data, offset, size)
	if err != nil {
		return Value{}, next, err
	}
	return Binary(chunk), next, nil
}

func readBytes(data []byte, offset int, size uint64) ([]byte, int, error) {
	if offset < 0 || offset > len(data) || size > uint64(len(data)-offset) {
		return nil, offset, fmt.Errorf("%w at offset %d for %d bytes", ErrTruncated, offset, size)
	}
	next := offset + int(size)
	return data[offset:next], next, nil
}

func readUint8(data []byte, offset int) (uint8, int, error) {
	chunk, next, err := readBytes(data, offset, 1)
	if err != nil {
		return 0, next, err
	}
	return chunk[0], next, nil
}

func readUint16(data []byte, offset int) (uint16, int, error) {
	chunk, next, err := readBytes(data, offset, 2)
	if err != nil {
		return 0, next, err
	}
	return binary.BigEndian.Uint16(chunk), next, nil
}

func readUint32(data []byte, offset int) (uint32, int, error) {
	chunk, next, err := readBytes(data, offset, 4)
	if err != nil {
		return 0, next, err
	}
	return binary.BigEndian.Uint32(chunk), next, nil
}

func readUint64(data []byte, offset int) (uint64, int, error) {
	chunk, next, err := readBytes(data, offset, 8)
	if err != nil {
		return 0, next, err
	}
	return binary.BigEndian.Uint64(chunk), next, nil
}

func messagePackMapKey(value Value) bool {
	switch value.Kind {
	case KindNil, KindBool, KindInteger, KindFloat, KindString, KindBinary:
		return true
	default:
		return false
	}
}

func setObjectField(fields []Field, field Field) []Field {
	for i := range fields {
		if fields[i].Key.Equal(field.Key) {
			fields[i].Value = field.Value
			return fields
		}
	}
	return append(fields, field)
}
