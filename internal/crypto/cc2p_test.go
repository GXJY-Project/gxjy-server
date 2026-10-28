package gxjycrypto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func TestCC2PRFC7539Vector(t *testing.T) {
	t.Parallel()

	key := mustHex(t, "808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9f")
	nonce := mustHex(t, "070000004041424344454647")
	aad := mustHex(t, "50515253c0c1c2c3c4c5c6c7")
	plaintext := []byte("Ladies and Gentlemen of the class of '99: If I could offer you only one tip for the future, sunscreen would be it.")
	wantCiphertext := mustHex(t,
		"d31a8d34648e60db7b86afbc53ef7ec2a4aded51296e08fea9e2b5a736ee62d6"+
			"3dbea45e8ca9671282fafb69da92728b1a71de0a9e060b2905d6a5b67ecd3b36"+
			"92ddbd7f2d778b8c9803aee328091b58fab324e4fad675945585808b4831d7bc"+
			"3ff4def08e4b7a9de576d26586cec64b6116")
	wantTag := mustHex(t, "1ae10b594f09e26a7e902ecbd0600691")

	ciphertext, tag, err := CC2PEncrypt(key, nonce, plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ciphertext, wantCiphertext) {
		t.Fatalf("密文不一致：\n得到 %x\n期望 %x", ciphertext, wantCiphertext)
	}
	if !bytes.Equal(tag, wantTag) {
		t.Fatalf("标签不一致：得到 %x，期望 %x", tag, wantTag)
	}

	decoded, err := CC2PDecrypt(key, nonce, ciphertext, tag, aad)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, plaintext) {
		t.Fatalf("解密结果不一致：得到 %x", decoded)
	}
}

func TestChaCha20RFC7539BlockVector(t *testing.T) {
	t.Parallel()

	key := mustHex(t, "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	nonce := mustHex(t, "000000090000004a00000000")
	want := mustHex(t,
		"10f1e7e4d13b5915500fdd1fa32071c4c7d1f4c733c068030422aa9ac3d46c4e"+
			"d2826446079faa0914c2d705d98b02a2b5129cd1de164eb9cbd083e8a2503c4e")

	got, err := ChaCha20(key, nonce, make([]byte, 64), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("密钥流不一致：\n得到 %x\n期望 %x", got, want)
	}
}

func TestCC2PPythonVectors(t *testing.T) {
	t.Parallel()

	key := sequence(32)
	nonce := sequence(12)
	tests := []struct {
		name          string
		plaintext     []byte
		aad           []byte
		wantCipherHex string
		wantTagHex    string
	}{
		{"空消息", nil, nil, "", "295a498b8841a1c5f55d4d606f731159"},
		{"短消息", []byte("hello"), nil, "e19e646c46", "f5939695c670586d63a69f58c41396dd"},
		{"附加数据", []byte("hello"), mustHex(t, "a1a2a3"), "e19e646c46", "1c058502f3bef6fe13240770443dcb53"},
		{
			"六十四字节",
			sequence(64),
			nil,
			"89fa0a032d12a347bf8a35f89410006cd961a0f44561bbaefe8e35de69ddb823" +
				"cca10ed0c23b97bf1f1b5cf349b9a10c4eb59b47c91d8eac2a81e33cac72a0e9",
			"12ccd3f87522adf137c06f3496d9562e",
		},
		{
			"六十五字节",
			sequence(65),
			[]byte("header"),
			"89fa0a032d12a347bf8a35f89410006cd961a0f44561bbaefe8e35de69ddb823" +
				"cca10ed0c23b97bf1f1b5cf349b9a10c4eb59b47c91d8eac2a81e33cac72a0e939",
			"65ec3e88d222d0d28bdd62fe956f40e0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ciphertext, tag, err := CC2PEncrypt(key, nonce, test.plaintext, test.aad)
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(ciphertext); got != test.wantCipherHex {
				t.Fatalf("密文 = %s，期望 %s", got, test.wantCipherHex)
			}
			if got := hex.EncodeToString(tag); got != test.wantTagHex {
				t.Fatalf("标签 = %s，期望 %s", got, test.wantTagHex)
			}

			decoded, err := CC2PDecrypt(key, nonce, ciphertext, tag, test.aad)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(decoded, test.plaintext) {
				t.Fatalf("解密结果 = %x，期望 %x", decoded, test.plaintext)
			}
		})
	}
}

func TestCC2PRejectsChangedInputs(t *testing.T) {
	t.Parallel()

	key := sequence(32)
	nonce := sequence(12)
	ciphertext, tag, err := CC2PEncrypt(key, nonce, []byte("hello"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}

	changedTag := bytes.Clone(tag)
	changedTag[0] ^= 1
	if _, err := CC2PDecrypt(key, nonce, ciphertext, changedTag, []byte("aad")); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("错误标签返回：%v", err)
	}
	if _, err := CC2PDecrypt(key, nonce, ciphertext, tag[:15], []byte("aad")); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("短标签返回：%v", err)
	}
	if _, err := CC2PDecrypt(key, nonce, ciphertext, tag, []byte("bad")); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("错误附加数据返回：%v", err)
	}
}

func TestCC2PInputRules(t *testing.T) {
	t.Parallel()

	if _, err := ChaCha20(make([]byte, 31), make([]byte, 12), nil, 1); !errors.Is(err, ErrKeyTooShort) {
		t.Fatalf("短密钥返回：%v", err)
	}
	if _, err := ChaCha20(make([]byte, 32), make([]byte, 11), nil, 1); !errors.Is(err, ErrNonceTooShort) {
		t.Fatalf("短 nonce 返回：%v", err)
	}

	key := append(sequence(32), 0xff)
	nonce := append(sequence(12), 0xff)
	got, err := ChaCha20(key, nonce, make([]byte, 64), 1)
	if err != nil {
		t.Fatal(err)
	}
	want := mustHex(t,
		"89fb08002917a540b7833ff3981d0e63c970b2e75174adb9e6972fc575c0a63c"+
			"ec802cf3e61eb198373276d865948f237e84a974fd28b89b12b8d907904f9ed6")
	if !bytes.Equal(got, want) {
		t.Fatalf("过长输入兼容结果不一致：得到 %x", got)
	}
}

func TestCC2PDoesNotModifyInputs(t *testing.T) {
	t.Parallel()

	key := sequence(32)
	nonce := sequence(12)
	plaintext := sequence(65)
	keyCopy := bytes.Clone(key)
	nonceCopy := bytes.Clone(nonce)
	plaintextCopy := bytes.Clone(plaintext)

	if _, _, err := CC2PEncrypt(key, nonce, plaintext, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, keyCopy) || !bytes.Equal(nonce, nonceCopy) || !bytes.Equal(plaintext, plaintextCopy) {
		t.Fatal("CC2PEncrypt 修改了输入切片")
	}
}

func mustHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func sequence(size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}
