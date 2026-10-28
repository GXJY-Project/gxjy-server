package protocol

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidOffset  = errors.New("invalid offset")
	ErrTruncated      = errors.New("truncated input")
	ErrVarintOverflow = errors.New("varint overflows uint64")
)

// WriteVarint 保持 Python write_varint 的编码行为。
// 非正数统一编码为一个零字节。
func WriteVarint(value int64) []byte {
	if value <= 0 {
		return []byte{0}
	}

	remaining := uint64(value)
	out := make([]byte, 0, 10)
	for remaining > 127 {
		out = append(out, byte(remaining&0x7f)|0x80)
		remaining >>= 7
	}
	return append(out, byte(remaining))
}

// ReadVarint 从指定偏移读取七位小端整数。
// Python 遇到截断会抛异常；Go 返回可分类错误。
func ReadVarint(data []byte, offset int) (value uint64, next int, err error) {
	if offset < 0 || offset > len(data) {
		return 0, offset, ErrInvalidOffset
	}

	next = offset
	for shift := uint(0); shift < 64; shift += 7 {
		if next >= len(data) {
			return 0, next, fmt.Errorf("%w at offset %d", ErrTruncated, next)
		}

		current := data[next]
		next++
		payload := uint64(current & 0x7f)
		if shift == 63 && payload > 1 {
			return 0, next, ErrVarintOverflow
		}
		value |= payload << shift
		if current&0x80 == 0 {
			return value, next, nil
		}
	}

	return 0, next, ErrVarintOverflow
}
