package assets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func TestXXTEAPythonVectors(t *testing.T) {
	t.Parallel()

	key := []byte("test-key")
	tests := []struct {
		name      string
		plaintext []byte
		cipherHex string
	}{
		{"空消息", nil, "885774a9"},
		{"短消息", []byte("hello"), "8907b60356a3f8ef0bd99a13"},
		{"对齐消息", []byte("12345678"), "09e1a1a674e921da55094cdd"},
		{"JSON", []byte(`{"a":1}`), "1be88660968d0cce1284e88b"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ciphertext, err := XXTEAEncrypt(test.plaintext, key)
			if err != nil {
				t.Fatal(err)
			}
			want := decodeHex(t, test.cipherHex)
			if !bytes.Equal(ciphertext, want) {
				t.Fatalf("密文 = %x，期望 %x", ciphertext, want)
			}
			plaintext, err := XXTEADecrypt(ciphertext, key)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(plaintext, test.plaintext) {
				t.Fatalf("明文 = %x，期望 %x", plaintext, test.plaintext)
			}
		})
	}
}

func TestXXTEAInputRules(t *testing.T) {
	t.Parallel()

	if _, err := XXTEADecrypt(nil, nil); !errors.Is(err, ErrXXTEADataLength) {
		t.Fatalf("空密文错误：%v", err)
	}
	if _, err := XXTEADecrypt([]byte{1, 2, 3}, nil); !errors.Is(err, ErrXXTEADataLength) {
		t.Fatalf("未对齐密文错误：%v", err)
	}

	ciphertext, err := XXTEAEncrypt([]byte("hello"), []byte("1234567890123456-extra"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := XXTEAEncrypt([]byte("hello"), []byte("1234567890123456"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ciphertext, want) {
		t.Fatal("过长密钥没有只使用前十六字节")
	}
}

func TestXXTEADoesNotModifyInputs(t *testing.T) {
	t.Parallel()

	plaintext := []byte("hello")
	key := []byte("test-key")
	plaintextCopy := bytes.Clone(plaintext)
	keyCopy := bytes.Clone(key)
	ciphertext, err := XXTEAEncrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, plaintextCopy) || !bytes.Equal(key, keyCopy) {
		t.Fatal("XXTEAEncrypt 修改了输入")
	}
	ciphertextCopy := bytes.Clone(ciphertext)
	if _, err := XXTEADecrypt(ciphertext, key); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ciphertext, ciphertextCopy) || !bytes.Equal(key, keyCopy) {
		t.Fatal("XXTEADecrypt 修改了输入")
	}
}

func TestXXTEAMatchesPythonBadPlainLength(t *testing.T) {
	t.Parallel()

	// Python 会把无效长度尾字转换为空字节。
	got, err := XXTEADecrypt([]byte{0, 0, 0, 0}, []byte("test-key"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("损坏密文结果 = %x", got)
	}
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}
