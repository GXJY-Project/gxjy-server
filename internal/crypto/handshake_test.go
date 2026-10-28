package gxjycrypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePEMAndRecoverRawSessionKey(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)})
	key, err := ParseRSAPrivatePEM(pemData)
	if err != nil {
		t.Fatal(err)
	}

	want := sequence(32)
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecoverSessionKey(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("会话密钥 = %x，期望 %x", got, want)
	}
}

func TestParsePKCS8PEM(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	der, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ParseRSAPrivatePEM(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		t.Fatal(err)
	}
	assertRSADecrypts(t, private, key, []byte("pkcs8"))
}

func TestFallbackJSONAndHexSessionKey(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	fallback := marshalFallbackKey(t, private)
	key, err := ParseRSAFallbackJSON(fallback)
	if err != nil {
		t.Fatal(err)
	}

	want := sequence(32)
	encoded := []byte(hex.EncodeToString(want))
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, encoded)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecoverSessionKey(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("十六进制会话密钥 = %x，期望 %x", got, want)
	}
}

func TestRecoverSessionKeyRawLengthPrecedesHex(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	key, err := ParseRSAFallbackJSON(marshalFallbackKey(t, private))
	if err != nil {
		t.Fatal(err)
	}

	// Python 会先接受任意 32 字节，再检查十六进制文本。
	plaintext := []byte("00112233445566778899aabbccddeeff")
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecoverSessionKey(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("Python 原始密钥顺序结果 = %x", got)
	}
}

func TestLoadRSAKeyFilesPreferenceAndFallback(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	directory := t.TempDir()
	pemPath := filepath.Join(directory, "gate.pem")
	fallbackPath := filepath.Join(directory, "fallback.json")

	validPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)})
	if err := os.WriteFile(pemPath, validPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallbackPath, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, source, err := LoadRSAKeyFiles(pemPath, fallbackPath)
	if err != nil {
		t.Fatal(err)
	}
	if source != RSAKeySourcePEM {
		t.Fatalf("密钥来源 = %q", source)
	}
	assertRSADecrypts(t, private, key, []byte("pem-first"))

	if err := os.WriteFile(pemPath, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallbackPath, marshalFallbackKey(t, private), 0o600); err != nil {
		t.Fatal(err)
	}
	key, source, err = LoadRSAKeyFiles(pemPath, fallbackPath)
	if err != nil {
		t.Fatal(err)
	}
	if source != RSAKeySourceFallback {
		t.Fatalf("回退来源 = %q", source)
	}
	assertRSADecrypts(t, private, key, []byte("fallback"))
}

func TestManualPKCS1PaddingChecks(t *testing.T) {
	t.Parallel()

	private := generateTestRSAKey(t)
	key, err := ParseRSAFallbackJSON(marshalFallbackKey(t, private))
	if err != nil {
		t.Fatal(err)
	}

	wrongType := make([]byte, private.Size())
	wrongType[1] = 1
	if _, err := key.DecryptPKCS1v15(encryptEncodedBlock(private, wrongType)); !errors.Is(err, ErrInvalidPKCS1Block) {
		t.Fatalf("错误区块类型返回：%v", err)
	}

	shortPadding := make([]byte, private.Size())
	shortPadding[1] = 2
	for i := 2; i < 5; i++ {
		shortPadding[i] = 0xff
	}
	if _, err := key.DecryptPKCS1v15(encryptEncodedBlock(private, shortPadding)); !errors.Is(err, ErrInvalidPKCS1Padding) {
		t.Fatalf("短填充返回：%v", err)
	}
}

func TestRSAHandshakeErrors(t *testing.T) {
	t.Parallel()

	if _, err := ParseRSAPrivatePEM([]byte("invalid")); !errors.Is(err, ErrInvalidPEM) {
		t.Fatalf("PEM 错误：%v", err)
	}
	for _, input := range [][]byte{
		[]byte(`{}`),
		[]byte(`{"n_hex":"01","d_hex":"02","e":1}`),
		[]byte(`{"n_hex":"zz","d_hex":"02","e":65537}`),
	} {
		if _, err := ParseRSAFallbackJSON(input); !errors.Is(err, ErrInvalidFallbackKey) {
			t.Fatalf("回退密钥错误：%v", err)
		}
	}
	if _, err := RecoverSessionKey(nil, nil); !errors.Is(err, ErrEmptySessionBlock) {
		t.Fatalf("空会话区块错误：%v", err)
	}

	private := generateTestRSAKey(t)
	key, err := ParseRSAFallbackJSON(marshalFallbackKey(t, private))
	if err != nil {
		t.Fatal(err)
	}
	for _, plaintext := range [][]byte{
		[]byte("short"),
		append(bytes.Repeat([]byte{'a'}, 32), 'g'),
		[]byte("00112233445566778899aabbccddeeff0"),
		append(bytes.Repeat([]byte{'a'}, 32), 0xff),
	} {
		ciphertext, encryptErr := rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, plaintext)
		if encryptErr != nil {
			t.Fatal(encryptErr)
		}
		if _, recoverErr := RecoverSessionKey(ciphertext, key); !errors.Is(recoverErr, ErrSessionKeyFormat) {
			t.Fatalf("会话文本 %x 返回：%v", plaintext, recoverErr)
		}
	}
}

func TestPythonRSAFallbackVector(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/python_rsa_vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		NHex         string `json:"n_hex"`
		DHex         string `json:"d_hex"`
		E            int    `json:"e"`
		RawCipherHex string `json:"raw_cipher_hex"`
		RawKeyHex    string `json:"raw_key_hex"`
		HexCipherHex string `json:"hex_cipher_hex"`
		HexText      string `json:"hex_text"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	fallback, err := json.Marshal(map[string]any{
		"n_hex": fixture.NHex,
		"d_hex": fixture.DHex,
		"e":     fixture.E,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, err := ParseRSAFallbackJSON(fallback)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		cipherHex string
		wantHex   string
	}{
		{"原始密钥", fixture.RawCipherHex, fixture.RawKeyHex},
		{"十六进制文本", fixture.HexCipherHex, fixture.RawKeyHex},
	} {
		t.Run(test.name, func(t *testing.T) {
			ciphertext, decodeErr := hex.DecodeString(test.cipherHex)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			got, recoverErr := RecoverSessionKey(ciphertext, key)
			if recoverErr != nil {
				t.Fatal(recoverErr)
			}
			if hex.EncodeToString(got) != test.wantHex {
				t.Fatalf("会话密钥 = %x，期望 %s", got, test.wantHex)
			}
		})
	}
}

func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return private
}

func marshalFallbackKey(t *testing.T, private *rsa.PrivateKey) []byte {
	t.Helper()
	value := map[string]any{
		"n_hex": private.N.Text(16),
		"d_hex": private.D.Text(16),
		"e":     private.E,
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertRSADecrypts(t *testing.T, private *rsa.PrivateKey, key *RSAKey, plaintext []byte) {
	t.Helper()
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &private.PublicKey, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	got, err := key.DecryptPKCS1v15(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("RSA 明文 = %x，期望 %x", got, plaintext)
	}
}

func encryptEncodedBlock(private *rsa.PrivateKey, encoded []byte) []byte {
	message := new(big.Int).SetBytes(encoded)
	ciphertext := new(big.Int).Exp(message, big.NewInt(int64(private.E)), private.N)
	return ciphertext.FillBytes(make([]byte, private.Size()))
}
