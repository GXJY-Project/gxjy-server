// Package gxjycrypto 实现游戏协议使用的加密算法。
package gxjycrypto

import (
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
)

const (
	CC2PKeySize   = 32
	CC2PNonceSize = 12
	CC2PTagSize   = 16
)

var (
	ErrKeyTooShort       = errors.New("CC2P 密钥长度不足")
	ErrNonceTooShort     = errors.New("CC2P nonce 长度不足")
	ErrAuthentication    = errors.New("CC2P 标签校验失败")
	poly1305ClampMask, _ = new(big.Int).SetString("0ffffffc0ffffffc0ffffffc0fffffff", 16)
	poly1305Prime        = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 130), big.NewInt(5))
	poly1305Modulo128    = new(big.Int).Lsh(big.NewInt(1), 128)
)

// ChaCha20 使用 RFC 7539 的 32 字节密钥和 12 字节 nonce。
// 为兼容 Python，过长输入只读取前 32 字节和前 12 字节。
func ChaCha20(key, nonce, data []byte, counter uint32) ([]byte, error) {
	if len(key) < CC2PKeySize {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrKeyTooShort, len(key))
	}
	if len(nonce) < CC2PNonceSize {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrNonceTooShort, len(nonce))
	}

	out := make([]byte, len(data))
	for offset := 0; offset < len(data); offset += 64 {
		block := chacha20Block(key[:CC2PKeySize], counter+uint32(offset/64), nonce[:CC2PNonceSize])
		end := min(offset+64, len(data))
		for i := offset; i < end; i++ {
			out[i] = data[i] ^ block[i-offset]
		}
	}
	return out, nil
}

// CC2PTag 计算 RFC 7539 ChaCha20-Poly1305 标签。
func CC2PTag(key, nonce, additionalData, ciphertext []byte) ([]byte, error) {
	if len(key) < CC2PKeySize {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrKeyTooShort, len(key))
	}
	if len(nonce) < CC2PNonceSize {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrNonceTooShort, len(nonce))
	}

	oneTimeKey := chacha20Block(key[:CC2PKeySize], 0, nonce[:CC2PNonceSize])[:32]
	mac := make([]byte, 0, paddedLength(len(additionalData))+paddedLength(len(ciphertext))+16)
	mac = appendPadded16(mac, additionalData)
	mac = appendPadded16(mac, ciphertext)
	mac = binary.LittleEndian.AppendUint64(mac, uint64(len(additionalData)))
	mac = binary.LittleEndian.AppendUint64(mac, uint64(len(ciphertext)))
	return poly1305MAC(mac, oneTimeKey), nil
}

// CC2PEncrypt 返回密文和独立的 16 字节标签。
func CC2PEncrypt(key, nonce, plaintext, additionalData []byte) ([]byte, []byte, error) {
	ciphertext, err := ChaCha20(key, nonce, plaintext, 1)
	if err != nil {
		return nil, nil, err
	}
	tag, err := CC2PTag(key, nonce, additionalData, ciphertext)
	if err != nil {
		return nil, nil, err
	}
	return ciphertext, tag, nil
}

// CC2PDecrypt 先校验标签，再解密密文。
func CC2PDecrypt(key, nonce, ciphertext, tag, additionalData []byte) ([]byte, error) {
	expected, err := CC2PTag(key, nonce, additionalData, ciphertext)
	if err != nil {
		return nil, err
	}
	if len(tag) != CC2PTagSize || subtle.ConstantTimeCompare(expected, tag) != 1 {
		return nil, ErrAuthentication
	}
	return ChaCha20(key, nonce, ciphertext, 1)
}

func chacha20Block(key []byte, counter uint32, nonce []byte) []byte {
	state := [16]uint32{
		0x61707865, 0x3320646e, 0x79622d32, 0x6b206574,
		binary.LittleEndian.Uint32(key[0:4]),
		binary.LittleEndian.Uint32(key[4:8]),
		binary.LittleEndian.Uint32(key[8:12]),
		binary.LittleEndian.Uint32(key[12:16]),
		binary.LittleEndian.Uint32(key[16:20]),
		binary.LittleEndian.Uint32(key[20:24]),
		binary.LittleEndian.Uint32(key[24:28]),
		binary.LittleEndian.Uint32(key[28:32]),
		counter,
		binary.LittleEndian.Uint32(nonce[0:4]),
		binary.LittleEndian.Uint32(nonce[4:8]),
		binary.LittleEndian.Uint32(nonce[8:12]),
	}
	working := state

	for range 10 {
		quarterRound(&working, 0, 4, 8, 12)
		quarterRound(&working, 1, 5, 9, 13)
		quarterRound(&working, 2, 6, 10, 14)
		quarterRound(&working, 3, 7, 11, 15)
		quarterRound(&working, 0, 5, 10, 15)
		quarterRound(&working, 1, 6, 11, 12)
		quarterRound(&working, 2, 7, 8, 13)
		quarterRound(&working, 3, 4, 9, 14)
	}

	out := make([]byte, 64)
	for i := range working {
		binary.LittleEndian.PutUint32(out[i*4:], working[i]+state[i])
	}
	return out
}

func quarterRound(state *[16]uint32, a, b, c, d int) {
	state[a] += state[b]
	state[d] = bits.RotateLeft32(state[d]^state[a], 16)
	state[c] += state[d]
	state[b] = bits.RotateLeft32(state[b]^state[c], 12)
	state[a] += state[b]
	state[d] = bits.RotateLeft32(state[d]^state[a], 8)
	state[c] += state[d]
	state[b] = bits.RotateLeft32(state[b]^state[c], 7)
}

func poly1305MAC(message, key []byte) []byte {
	r := littleEndianInteger(key[:16])
	r.And(r, poly1305ClampMask)
	s := littleEndianInteger(key[16:32])
	accumulator := new(big.Int)

	for offset := 0; offset < len(message); offset += 16 {
		end := min(offset+16, len(message))
		block := make([]byte, end-offset+1)
		copy(block, message[offset:end])
		block[len(block)-1] = 1
		n := littleEndianInteger(block)
		accumulator.Add(accumulator, n)
		accumulator.Mul(accumulator, r)
		accumulator.Mod(accumulator, poly1305Prime)
	}

	accumulator.Add(accumulator, s)
	accumulator.Mod(accumulator, poly1305Modulo128)
	return fixedLittleEndian(accumulator, CC2PTagSize)
}

func littleEndianInteger(data []byte) *big.Int {
	reversed := make([]byte, len(data))
	for i := range data {
		reversed[len(data)-1-i] = data[i]
	}
	return new(big.Int).SetBytes(reversed)
}

func fixedLittleEndian(value *big.Int, size int) []byte {
	bigEndian := value.Bytes()
	out := make([]byte, size)
	for i := 0; i < len(bigEndian) && i < size; i++ {
		out[i] = bigEndian[len(bigEndian)-1-i]
	}
	return out
}

func appendPadded16(out, data []byte) []byte {
	out = append(out, data...)
	if remainder := len(data) % 16; remainder != 0 {
		out = append(out, make([]byte, 16-remainder)...)
	}
	return out
}

func paddedLength(size int) int {
	if remainder := size % 16; remainder != 0 {
		return size + 16 - remainder
	}
	return size
}
