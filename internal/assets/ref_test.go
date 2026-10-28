package assets

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeHeaderlessREFPythonVector(t *testing.T) {
	t.Parallel()

	container := decodeHex(t,
		"5245463a00000002425a1e3f97a453ec90a426f589d0377f5811aa5b696d8e35"+
			"1bc19a8140e80a9d")
	want := decodeHex(t, "1f8b08000000000002ffab564a54b232ac0500afac1b5607000000")
	got, err := DecodeREF(container, []byte("test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("无头 REF = %x，期望 %x", got, want)
	}
}

func TestDecodeHeaderedREFPythonVector(t *testing.T) {
	t.Parallel()

	container := decodeHex(t,
		"5245463a00000002123400000020425a1e3f97a453ec90a426f589d0377f5811aa5b"+
			"696d8e351bc19a8140e80a9d5441494c")
	want := decodeHex(t, "1f8b08000000000002ffab564a54b232ac0500afac1b56070000005441494c")
	got, err := DecodeREF(container, []byte("test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("有头 REF = %x，期望 %x", got, want)
	}
}

func TestDecodeHeaderlessJSONREF(t *testing.T) {
	t.Parallel()

	container := append([]byte{'R', 'E', 'F', ':', 0, 0, 0, 9}, decodeHex(t, "1be88660968d0cce1284e88b")...)
	got, err := DecodeREF(container, []byte("test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("JSON REF = %q", got)
	}
}

func TestDecodeREFFile(t *testing.T) {
	t.Parallel()

	container := append(refSignature[:], decodeHex(t, "1be88660968d0cce1284e88b")...)
	path := filepath.Join(t.TempDir(), "sample.ref")
	if err := os.WriteFile(path, container, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeREFFile(path, []byte("test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("文件解码 = %q", got)
	}
}

func TestDecodeREFErrors(t *testing.T) {
	t.Parallel()

	if _, err := DecodeREF([]byte("plain"), nil); !errors.Is(err, ErrREFSignature) {
		t.Fatalf("签名错误：%v", err)
	}
	if _, err := DecodeREF(refSignature[:], nil); !errors.Is(err, ErrREFUnknownLayout) {
		t.Fatalf("空正文错误：%v", err)
	}

	tooLong := append([]byte(nil), refSignature[:]...)
	tooLong = append(tooLong, 0, 0, 0, 0, 0, 8, 1, 2, 3, 4)
	if _, err := DecodeREF(tooLong, nil); !errors.Is(err, ErrREFUnknownLayout) {
		t.Fatalf("超长声明错误：%v", err)
	}

	unaligned := append([]byte(nil), refSignature[:]...)
	unaligned = append(unaligned, 0, 0, 0, 0, 0, 3, 1, 2, 3)
	if _, err := DecodeREF(unaligned, nil); !errors.Is(err, ErrREFUnknownLayout) {
		t.Fatalf("未对齐声明错误：%v", err)
	}
}
