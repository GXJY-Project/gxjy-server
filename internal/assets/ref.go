package assets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

var (
	refPrefix    = [4]byte{'R', 'E', 'F', ':'}
	refSignature = [8]byte{'R', 'E', 'F', ':', 0, 0, 0, 2}

	ErrREFSignature       = errors.New("REF 签名无效")
	ErrREFBodyTooShort    = errors.New("REF 正文长度不足")
	ErrREFEncryptedLength = errors.New("REF 加密长度无效")
	ErrREFUnknownLayout   = errors.New("无法识别 REF 容器布局")
)

// DecodeREFFile 读取并解码一个 REF 文件。
func DecodeREFFile(path string, key []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeREF(data, key)
}

// DecodeREF 先尝试无头布局，再回退到六字节内层头布局。
func DecodeREF(data, key []byte) ([]byte, error) {
	if len(data) < len(refSignature) || !bytes.Equal(data[:4], refPrefix[:]) {
		return nil, ErrREFSignature
	}

	body := data[len(refSignature):]
	var headerlessErr error
	alignedLength := len(body) / 4 * 4
	if alignedLength > 0 {
		plaintext, err := XXTEADecrypt(body[:alignedLength], key)
		if err == nil && recognizedREFPlaintext(plaintext) {
			return plaintext, nil
		}
		headerlessErr = err
	}

	plaintext, err := decodeHeaderedREF(data, key)
	if err == nil {
		return plaintext, nil
	}
	return nil, fmt.Errorf("%w：%v", ErrREFUnknownLayout, errors.Join(headerlessErr, err))
}

func decodeHeaderedREF(data, key []byte) ([]byte, error) {
	if len(data) < len(refSignature) || !bytes.Equal(data[:len(refSignature)], refSignature[:]) {
		return nil, ErrREFSignature
	}
	body := data[len(refSignature):]
	if len(body) < 6 {
		return nil, ErrREFBodyTooShort
	}
	encryptedLength := uint64(binary.BigEndian.Uint32(body[2:6]))
	payload := body[6:]
	if encryptedLength > uint64(len(payload)) {
		return nil, fmt.Errorf("%w：声明 %d，只剩 %d", ErrREFEncryptedLength, encryptedLength, len(payload))
	}
	if encryptedLength%4 != 0 {
		return nil, fmt.Errorf("%w：%d 不是四的倍数", ErrREFEncryptedLength, encryptedLength)
	}
	plaintext, err := XXTEADecrypt(payload[:int(encryptedLength)], key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(plaintext)+len(payload)-int(encryptedLength))
	out = append(out, plaintext...)
	out = append(out, payload[int(encryptedLength):]...)
	return out, nil
}

func recognizedREFPlaintext(plaintext []byte) bool {
	return len(plaintext) >= 2 && plaintext[0] == 0x1f && plaintext[1] == 0x8b ||
		len(plaintext) >= 1 && (plaintext[0] == '{' || plaintext[0] == '[')
}
