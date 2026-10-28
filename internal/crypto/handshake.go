package gxjycrypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

// RSAKeySource 表示运行时选中的私钥来源。
type RSAKeySource string

const (
	RSAKeySourcePEM      RSAKeySource = "pem"
	RSAKeySourceFallback RSAKeySource = "fallback-json"
)

var (
	ErrInvalidPEM          = errors.New("无效的 RSA PEM 私钥")
	ErrInvalidFallbackKey  = errors.New("无效的 RSA 回退密钥")
	ErrInvalidPKCS1Block   = errors.New("无效的 PKCS#1 区块")
	ErrInvalidPKCS1Padding = errors.New("无效的 PKCS#1 填充")
	ErrEmptySessionBlock   = errors.New("会话密钥区块为空")
	ErrSessionKeyFormat    = errors.New("会话密钥格式无效")
)

// RSAKey 同时支持完整 PEM 私钥和只有 N、D、E 的旧回退格式。
type RSAKey struct {
	private        *rsa.PrivateKey
	modulus        *big.Int
	exponent       *big.Int
	publicExponent int
	size           int
}

type fallbackRSAJSON struct {
	NHex string          `json:"n_hex"`
	DHex string          `json:"d_hex"`
	E    json.RawMessage `json:"e"`
}

// LoadRSAKeyFiles 优先读取 PEM，失败后读取 N、D、E JSON。
func LoadRSAKeyFiles(pemPath, fallbackPath string) (*RSAKey, RSAKeySource, error) {
	var pemErr error
	if pemPath != "" {
		data, err := os.ReadFile(pemPath)
		if err == nil {
			key, parseErr := ParseRSAPrivatePEM(data)
			if parseErr == nil {
				return key, RSAKeySourcePEM, nil
			}
			pemErr = parseErr
		} else {
			pemErr = err
		}
	}

	if fallbackPath == "" {
		if pemErr != nil {
			return nil, "", pemErr
		}
		return nil, "", ErrInvalidFallbackKey
	}
	data, err := os.ReadFile(fallbackPath)
	if err != nil {
		return nil, "", errors.Join(pemErr, err)
	}
	key, parseErr := ParseRSAFallbackJSON(data)
	if parseErr != nil {
		return nil, "", errors.Join(pemErr, parseErr)
	}
	return key, RSAKeySourceFallback, nil
}

// ParseRSAPrivatePEM 读取未加密的 PKCS#1 或 PKCS#8 RSA 私钥。
func ParseRSAPrivatePEM(data []byte) (*RSAKey, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, ErrInvalidPEM
	}

	if private, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return newPEMRSAKey(private)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrInvalidPEM, err)
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w：PKCS#8 内容不是 RSA", ErrInvalidPEM)
	}
	return newPEMRSAKey(private)
}

// ParseRSAFallbackJSON 读取 Python server_rsa.json 使用的字段。
func ParseRSAFallbackJSON(data []byte) (*RSAKey, error) {
	var raw fallbackRSAJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrInvalidFallbackKey, err)
	}
	modulus, ok := parsePositiveHex(raw.NHex)
	if !ok {
		return nil, fmt.Errorf("%w：n_hex", ErrInvalidFallbackKey)
	}
	exponent, ok := parsePositiveHex(raw.DHex)
	if !ok {
		return nil, fmt.Errorf("%w：d_hex", ErrInvalidFallbackKey)
	}
	publicExponent, err := parsePublicExponent(raw.E)
	if err != nil {
		return nil, err
	}
	return &RSAKey{
		modulus:        modulus,
		exponent:       exponent,
		publicExponent: publicExponent,
		size:           (modulus.BitLen() + 7) / 8,
	}, nil
}

// DecryptPKCS1v15 解密 Python 使用的 RSA PKCS#1 v1.5 区块。
func (key *RSAKey) DecryptPKCS1v15(ciphertext []byte) ([]byte, error) {
	if key == nil || key.modulus == nil || key.exponent == nil || key.size == 0 {
		return nil, ErrInvalidFallbackKey
	}
	if key.private != nil {
		plaintext, err := rsa.DecryptPKCS1v15(nil, key.private, ciphertext)
		if err != nil {
			return nil, fmt.Errorf("%w：%v", ErrInvalidPKCS1Padding, err)
		}
		return plaintext, nil
	}

	cipherInteger := new(big.Int).SetBytes(ciphertext)
	messageInteger := new(big.Int).Exp(cipherInteger, key.exponent, key.modulus)
	encoded := messageInteger.FillBytes(make([]byte, key.size))
	if len(encoded) < 2 || encoded[0] != 0 || encoded[1] != 2 {
		return nil, ErrInvalidPKCS1Block
	}
	separator := -1
	for i := 2; i < len(encoded); i++ {
		if encoded[i] == 0 {
			separator = i
			break
		}
	}
	if separator < 10 {
		return nil, ErrInvalidPKCS1Padding
	}
	return append([]byte(nil), encoded[separator+1:]...), nil
}

// RecoverSessionKey 保持 Python recover_session_key 的密钥格式规则。
func RecoverSessionKey(block []byte, key *RSAKey) ([]byte, error) {
	if len(block) == 0 {
		return nil, ErrEmptySessionBlock
	}
	plaintext, err := key.DecryptPKCS1v15(block)
	if err != nil {
		return nil, err
	}
	if len(plaintext) == CC2PKeySize {
		return append([]byte(nil), plaintext...), nil
	}

	for _, current := range plaintext {
		if current > 0x7f {
			return nil, ErrSessionKeyFormat
		}
	}
	text := strings.TrimSpace(string(plaintext))
	if len(text) < 32 || !isASCIIHex(text) {
		return nil, ErrSessionKeyFormat
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrSessionKeyFormat, err)
	}
	return decoded, nil
}

func newPEMRSAKey(private *rsa.PrivateKey) (*RSAKey, error) {
	if private == nil || private.N == nil || private.D == nil || private.N.Sign() <= 0 || private.D.Sign() <= 0 {
		return nil, ErrInvalidPEM
	}
	if err := private.Validate(); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrInvalidPEM, err)
	}
	return &RSAKey{
		private:        private,
		modulus:        new(big.Int).Set(private.N),
		exponent:       new(big.Int).Set(private.D),
		publicExponent: private.E,
		size:           private.Size(),
	}, nil
}

func parsePositiveHex(value string) (*big.Int, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0x")
	value = strings.TrimPrefix(value, "0X")
	parsed, ok := new(big.Int).SetString(value, 16)
	return parsed, ok && parsed.Sign() > 0
}

func parsePublicExponent(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("%w：e", ErrInvalidFallbackKey)
	}
	text := strings.TrimSpace(string(raw))
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return 0, fmt.Errorf("%w：e", ErrInvalidFallbackKey)
		}
		text = decoded
	}
	value, err := strconv.ParseUint(text, 10, 31)
	if err != nil || value < 2 {
		return 0, fmt.Errorf("%w：e", ErrInvalidFallbackKey)
	}
	return int(value), nil
}

func isASCIIHex(value string) bool {
	for i := range len(value) {
		current := value[i]
		if !((current >= '0' && current <= '9') ||
			(current >= 'a' && current <= 'f') ||
			(current >= 'A' && current <= 'F')) {
			return false
		}
	}
	return true
}
