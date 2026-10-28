package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
)

const (
	officialEncryptionType = 1
	communityEncodingByte  = 0x10
)

var (
	ErrCC2PPayloadTooShort = errors.New("CC2P 载荷长度不足")
	ErrFrameBodyTooShort   = errors.New("游戏帧正文长度不足")
	ErrRSABlockTruncated   = errors.New("RSA 区块被截断")
	ErrFrameTooLarge       = errors.New("游戏帧长度过大")
)

// ClientPacket 是官方客户端上行帧的外层结构。
type ClientPacket struct {
	EncryptionType byte
	RSALength      uint64
	RSA            []byte
	Ciphertext     []byte
	Raw            []byte
}

// DecodedBody 是解密后的游戏消息。
type DecodedBody struct {
	Sequence         uint32
	Command          uint16
	Message          Value
	UsedTextFallback bool
}

// SplitCC2PPayload 按“密文、标签、nonce”切分载荷。
func SplitCC2PPayload(payload []byte) (ciphertext, tag, nonce []byte, err error) {
	if len(payload) < gxjycrypto.CC2PTagSize+gxjycrypto.CC2PNonceSize {
		return nil, nil, nil, ErrCC2PPayloadTooShort
	}
	ciphertextEnd := len(payload) - gxjycrypto.CC2PTagSize - gxjycrypto.CC2PNonceSize
	tagEnd := len(payload) - gxjycrypto.CC2PNonceSize
	return bytes.Clone(payload[:ciphertextEnd]),
		bytes.Clone(payload[ciphertextEnd:tagEnd]),
		bytes.Clone(payload[tagEnd:]), nil
}

// BuildServerFrame 构造官方服务端下行帧。
// nonceSource 为空时使用系统加密随机源。
func BuildServerFrame(command uint16, message Value, sequence uint32, key []byte, encryptionType byte, nonceSource io.Reader) ([]byte, error) {
	packed, err := MarshalMessagePack(message)
	if err != nil {
		return nil, err
	}

	body := make([]byte, 0, 6+len(packed))
	body = binary.BigEndian.AppendUint32(body, sequence)
	body = binary.BigEndian.AppendUint16(body, command)
	body = append(body, packed...)
	payload := body

	if encryptionType == officialEncryptionType && key != nil {
		if nonceSource == nil {
			nonceSource = rand.Reader
		}
		nonce := make([]byte, gxjycrypto.CC2PNonceSize)
		if _, err := io.ReadFull(nonceSource, nonce); err != nil {
			return nil, fmt.Errorf("读取 CC2P nonce：%w", err)
		}
		ciphertext, tag, err := gxjycrypto.CC2PEncrypt(key, nonce, body, nil)
		if err != nil {
			return nil, err
		}
		payload = make([]byte, 0, len(ciphertext)+len(tag)+len(nonce))
		payload = append(payload, ciphertext...)
		payload = append(payload, tag...)
		payload = append(payload, nonce...)
	}

	if len(payload) > math.MaxInt64-1 {
		return nil, ErrFrameTooLarge
	}
	frame := WriteVarint(int64(1 + len(payload)))
	frame = append(frame, encryptionType)
	frame = append(frame, payload...)
	return frame, nil
}

// ParseClientPacket 解析官方客户端上行帧。
func ParseClientPacket(data []byte) (ClientPacket, error) {
	if len(data) == 0 {
		return ClientPacket{}, ErrFrameBodyTooShort
	}

	rsaLength, offset, err := ReadVarint(data, 1)
	if err != nil {
		return ClientPacket{}, err
	}
	if rsaLength > uint64(len(data)-offset) {
		return ClientPacket{}, fmt.Errorf("%w：声明 %d 字节，只剩 %d 字节", ErrRSABlockTruncated, rsaLength, len(data)-offset)
	}
	rsaEnd := offset + int(rsaLength)
	return ClientPacket{
		EncryptionType: data[0] & 0x0f,
		RSALength:      rsaLength,
		RSA:            bytes.Clone(data[offset:rsaEnd]),
		Ciphertext:     bytes.Clone(data[rsaEnd:]),
		Raw:            bytes.Clone(data),
	}, nil
}

// DecodeClientBody 解密并解析官方客户端消息正文。
func DecodeClientBody(payload, key []byte) (DecodedBody, error) {
	ciphertext, tag, nonce, err := SplitCC2PPayload(payload)
	if err != nil {
		return DecodedBody{}, err
	}
	body, err := gxjycrypto.CC2PDecrypt(key, nonce, ciphertext, tag, nil)
	if err != nil {
		return DecodedBody{}, err
	}
	return DecodeBody(body)
}

// BuildPlainFrame 构造社区客户端使用的明文 MessagePack 帧。
func BuildPlainFrame(command uint16, message Value, sequence uint32) ([]byte, error) {
	body, err := MarshalMessagePack(message)
	if err != nil {
		return nil, err
	}
	if uint64(len(body))+10 > math.MaxUint32 {
		return nil, ErrFrameTooLarge
	}

	shortLength := len(body) + 10
	prefixSize := 1
	if shortLength >= 255 {
		prefixSize = 5
	}
	out := make([]byte, 0, prefixSize+11+len(body))
	if shortLength < 255 {
		out = append(out, byte(shortLength))
	} else {
		out = append(out, 0xff)
		out = binary.BigEndian.AppendUint32(out, uint32(shortLength))
	}
	out = append(out, communityEncodingByte)
	out = binary.BigEndian.AppendUint32(out, 0)
	out = binary.BigEndian.AppendUint32(out, sequence)
	out = binary.BigEndian.AppendUint16(out, command)
	out = append(out, body...)
	return out, nil
}

// LooksLikePlain 保持 Python 的首包协议判断规则。
func LooksLikePlain(data []byte) bool {
	if len(data) == 0 || data[0] == 0x01 {
		return false
	}
	if data[0] == 0xff {
		return len(data) >= 5
	}
	return true
}

// ParsePlainFrame 解析社区客户端的明文帧。
func ParsePlainFrame(data []byte) (DecodedBody, byte, error) {
	if len(data) == 0 {
		return DecodedBody{}, 0, ErrFrameBodyTooShort
	}
	offset := 1
	if data[0] == 0xff {
		offset += 4
	}
	if len(data)-offset < 11 {
		return DecodedBody{}, 0, ErrFrameBodyTooShort
	}

	encryptionType := data[offset] & 0x0f
	offset++
	offset += 4
	sequence := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	command := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	message, _, err := UnmarshalMessagePack(data, offset)
	if err == nil {
		return DecodedBody{Sequence: sequence, Command: command, Message: message}, encryptionType, nil
	}
	return DecodedBody{
		Sequence:         sequence,
		Command:          command,
		Message:          String(strings.ToValidUTF8(string(data[offset:]), "\uFFFD")),
		UsedTextFallback: true,
	}, encryptionType, nil
}

// DecodeBody 解析大端序号、命令号和 MessagePack 正文。
func DecodeBody(body []byte) (DecodedBody, error) {
	if len(body) < 6 {
		return DecodedBody{}, ErrFrameBodyTooShort
	}
	sequence := binary.BigEndian.Uint32(body[0:4])
	command := binary.BigEndian.Uint16(body[4:6])
	message, _, err := UnmarshalMessagePack(body, 6)
	if err == nil {
		return DecodedBody{Sequence: sequence, Command: command, Message: message}, nil
	}
	return DecodedBody{
		Sequence:         sequence,
		Command:          command,
		Message:          String(strings.ToValidUTF8(string(body[6:]), "\uFFFD")),
		UsedTextFallback: true,
	}, nil
}
