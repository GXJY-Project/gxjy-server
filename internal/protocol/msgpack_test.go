package protocol

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestMessagePackPythonVectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Value
		hex   string
	}{
		{"nil", Nil(), "c0"},
		{"true", Bool(true), "c3"},
		{"false", Bool(false), "c2"},
		{"zero", Int(0), "00"},
		{"positive-fix-max", Int(127), "7f"},
		{"uint8-min", Int(128), "cc80"},
		{"uint8-max", Int(255), "ccff"},
		{"uint16-min", Int(256), "cd0100"},
		{"uint16-max", Int(65535), "cdffff"},
		{"uint32-min", Int(65536), "ce00010000"},
		{"uint32-max", Uint(1<<32 - 1), "ceffffffff"},
		{"uint64-min", Uint(1 << 32), "cf0000000100000000"},
		{"negative-one", Int(-1), "ff"},
		{"negative-fix-min", Int(-32), "e0"},
		{"int8-next", Int(-33), "d0df"},
		{"int8-min", Int(-128), "d080"},
		{"int16-next", Int(-129), "d1ff7f"},
		{"int16-min", Int(-32768), "d18000"},
		{"int32-next", Int(-32769), "d2ffff7fff"},
		{"int32-min", Int(math.MinInt32), "d280000000"},
		{"int64-next", Int(math.MinInt32 - 1), "d3ffffffff7fffffff"},
		{"float64", Float(1.5), "cb3ff8000000000000"},
		{"unicode", String("中文测试"), "ace4b8ade69687e6b58be8af95"},
		{"binary", Binary([]byte{0x00, 0xff}), "c40200ff"},
		{"array", Array(Int(1), String("x"), Nil()), "9301a178c0"},
		{
			"ordered-object",
			Object(Named("result", Int(0)), Named("cmd", String("MSG_TYPE_LOGIN"))),
			"82a6726573756c7400a3636d64ae4d53475f545950455f4c4f47494e",
		},
		{
			"nested",
			Object(
				Named("list", Array(Int(1), Int(2), Int(3))),
				Named("nested", Object(Named("a", String("中文测试")))),
			),
			"82a46c69737493010203a66e657374656481a161ace4b8ade69687e6b58be8af95",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, err := hex.DecodeString(test.hex)
			if err != nil {
				t.Fatal(err)
			}

			got, err := MarshalMessagePack(test.value)
			if err != nil {
				t.Fatalf("MarshalMessagePack: %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("encoded = %x, want %x", got, want)
			}

			decoded, next, err := UnmarshalMessagePack(got, 0)
			if err != nil {
				t.Fatalf("UnmarshalMessagePack: %v", err)
			}
			if next != len(got) {
				t.Fatalf("next = %d, want %d", next, len(got))
			}
			if !decoded.Equal(test.value) {
				t.Fatalf("decoded = %#v, want %#v", decoded, test.value)
			}
		})
	}
}

func TestMessagePackLengthBoundaries(t *testing.T) {
	t.Parallel()

	stringsToTest := []struct {
		size   int
		prefix string
	}{
		{31, "bf"},
		{32, "d920"},
		{255, "d9ff"},
		{256, "da0100"},
	}
	for _, test := range stringsToTest {
		encoded, err := MarshalMessagePack(String(strings.Repeat("x", test.size)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(hex.EncodeToString(encoded), test.prefix) {
			t.Fatalf("string size %d encoded with %x", test.size, encoded[:min(5, len(encoded))])
		}
	}

	array15 := make([]Value, 15)
	array16 := make([]Value, 16)
	encoded15, _ := MarshalMessagePack(Array(array15...))
	encoded16, _ := MarshalMessagePack(Array(array16...))
	if encoded15[0] != 0x9f || string(encoded16[:3]) != string([]byte{0xdc, 0x00, 0x10}) {
		t.Fatalf("array headers = %x and %x", encoded15[:1], encoded16[:3])
	}
}

func TestMessagePackDecodeFloat32(t *testing.T) {
	t.Parallel()

	value, next, err := UnmarshalMessagePack([]byte{0xca, 0x3f, 0xc0, 0x00, 0x00}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if next != 5 || value.Kind != KindFloat || value.Float64 != 1.5 {
		t.Fatalf("decoded = %#v, next = %d", value, next)
	}
}

func TestMessagePackDuplicateMapKeyMatchesPython(t *testing.T) {
	t.Parallel()

	// Python 字典会替换旧值，但不会移动键的位置。
	input := []byte{0x82, 0xa1, 'a', 0x01, 0xa1, 'a', 0x02}
	value, next, err := UnmarshalMessagePack(input, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := Object(Named("a", Int(2)))
	if next != len(input) || !value.Equal(want) {
		t.Fatalf("decoded = %#v, next = %d", value, next)
	}
}

func TestMessagePackErrors(t *testing.T) {
	t.Parallel()

	if _, err := MarshalMessagePack(Value{Kind: KindInteger, Negative: true}); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("negative zero error = %v", err)
	}
	if _, err := MarshalMessagePack(String(string([]byte{0xff}))); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
	if _, _, err := UnmarshalMessagePack([]byte{0xd9, 0x03, 'x'}, 0); !errors.Is(err, ErrTruncated) {
		t.Fatalf("truncated string error = %v", err)
	}
	if _, _, err := UnmarshalMessagePack([]byte{0xc1}, 0); !errors.Is(err, ErrUnsupportedValue) {
		t.Fatalf("unsupported marker error = %v", err)
	}
	if _, _, err := UnmarshalMessagePack(nil, 0); !errors.Is(err, ErrTruncated) {
		t.Fatalf("empty input error = %v", err)
	}
}
