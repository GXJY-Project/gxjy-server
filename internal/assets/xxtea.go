package assets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const xxteaDelta uint32 = 0x9e3779b9

var (
	ErrXXTEADataLength  = errors.New("XXTEA 数据长度无效")
	ErrXXTEAPlainLength = errors.New("XXTEA 明文长度无效")
)

// XXTEAEncrypt 使用 Python 工具相同的长度尾字编码。
func XXTEAEncrypt(plaintext, key []byte) ([]byte, error) {
	if uint64(len(plaintext)) > math.MaxUint32 {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrXXTEADataLength, len(plaintext))
	}
	words := bytesToWords(plaintext, true)
	keyWords := normalizedKeyWords(key)
	n := len(words) - 1
	z := words[n]
	y := words[0]
	rounds := 6 + 52/(n+1)
	var sum uint32

	for range rounds {
		sum += xxteaDelta
		e := (sum >> 2) & 3
		for p := range n {
			y = words[p+1]
			words[p] += xxteaMix(sum, y, z, uint32(p), e, keyWords)
			z = words[p]
		}
		y = words[0]
		words[n] += xxteaMix(sum, y, z, uint32(n), e, keyWords)
		z = words[n]
	}
	return wordsToBytes(words, false)
}

// XXTEADecrypt 解密 Python xxtea_decrypt_raw 生成的数据。
func XXTEADecrypt(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) < 4 || len(ciphertext)%4 != 0 {
		return nil, fmt.Errorf("%w：得到 %d 字节", ErrXXTEADataLength, len(ciphertext))
	}
	words := bytesToWords(ciphertext, false)
	keyWords := normalizedKeyWords(key)
	n := len(words) - 1
	z := words[n]
	y := words[0]
	rounds := 6 + 52/(n+1)
	sum := uint32(rounds) * xxteaDelta

	for sum != 0 {
		e := (sum >> 2) & 3
		for p := n; p > 0; p-- {
			z = words[p-1]
			words[p] -= xxteaMix(sum, y, z, uint32(p), e, keyWords)
			y = words[p]
		}
		z = words[n]
		words[0] -= xxteaMix(sum, y, z, 0, e, keyWords)
		y = words[0]
		sum -= xxteaDelta
	}
	plaintext, err := wordsToBytes(words, true)
	if errors.Is(err, ErrXXTEAPlainLength) {
		return []byte{}, nil
	}
	return plaintext, err
}

func xxteaMix(sum, y, z, position, e uint32, key [4]uint32) uint32 {
	return ((z>>5 ^ y<<2) + (y>>3 ^ z<<4)) ^
		((sum ^ y) + (key[(position&3)^e] ^ z))
}

func normalizedKeyWords(key []byte) [4]uint32 {
	var normalized [16]byte
	copy(normalized[:], key)
	return [4]uint32{
		binary.LittleEndian.Uint32(normalized[0:4]),
		binary.LittleEndian.Uint32(normalized[4:8]),
		binary.LittleEndian.Uint32(normalized[8:12]),
		binary.LittleEndian.Uint32(normalized[12:16]),
	}
}

func bytesToWords(data []byte, includeLength bool) []uint32 {
	wordCount := (len(data) + 3) / 4
	words := make([]uint32, wordCount, wordCount+1)
	for i, current := range data {
		words[i/4] |= uint32(current) << (8 * uint(i%4))
	}
	if includeLength {
		words = append(words, uint32(len(data)))
	}
	return words
}

func wordsToBytes(words []uint32, includeLength bool) ([]byte, error) {
	if len(words) == 0 {
		return nil, ErrXXTEADataLength
	}
	length := len(words) * 4
	if includeLength {
		maximum := (len(words) - 1) * 4
		declared := uint64(words[len(words)-1])
		minimum := 0
		if maximum > 3 {
			minimum = maximum - 3
		}
		if declared < uint64(minimum) || declared > uint64(maximum) {
			return nil, fmt.Errorf("%w：声明 %d，范围 %d..%d", ErrXXTEAPlainLength, declared, minimum, maximum)
		}
		length = int(declared)
	}
	out := make([]byte, len(words)*4)
	for i, word := range words {
		binary.LittleEndian.PutUint32(out[i*4:], word)
	}
	return out[:length], nil
}
