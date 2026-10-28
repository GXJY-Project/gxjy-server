package protocol

import (
	"encoding/hex"
	"errors"
	"testing"
)

func TestWriteVarintPythonVectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   int64
		hex  string
	}{
		{"negative", -1, "00"},
		{"zero", 0, "00"},
		{"one", 1, "01"},
		{"sixty-three", 63, "3f"},
		{"sixty-four", 64, "40"},
		{"fixed-max", 127, "7f"},
		{"two-byte-min", 128, "8001"},
		{"three-hundred", 300, "ac02"},
		{"two-byte-max", 16383, "ff7f"},
		{"three-byte-min", 16384, "808001"},
		{"one-mebibit", 1 << 20, "808040"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, err := hex.DecodeString(test.hex)
			if err != nil {
				t.Fatal(err)
			}
			got := WriteVarint(test.in)
			if string(got) != string(want) {
				t.Fatalf("WriteVarint(%d) = %x, want %x", test.in, got, want)
			}
		})
	}
}

func TestReadVarintPythonVectors(t *testing.T) {
	t.Parallel()

	for _, value := range []int64{0, 1, 63, 64, 127, 128, 300, 16383, 16384, 1 << 20} {
		encoded := append([]byte{0xaa}, WriteVarint(value)...)
		got, next, err := ReadVarint(encoded, 1)
		if err != nil {
			t.Fatalf("ReadVarint(%d): %v", value, err)
		}
		if got != uint64(value) || next != len(encoded) {
			t.Fatalf("ReadVarint(%d) = (%d, %d), want (%d, %d)", value, got, next, value, len(encoded))
		}
	}
}

func TestReadVarintErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := ReadVarint(nil, -1); !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("negative offset error = %v", err)
	}
	if _, _, err := ReadVarint([]byte{0x80}, 0); !errors.Is(err, ErrTruncated) {
		t.Fatalf("truncated error = %v", err)
	}
	overflow := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02}
	if _, _, err := ReadVarint(overflow, 0); !errors.Is(err, ErrVarintOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
}
