package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
)

func TestBuildServerFramePythonVectors(t *testing.T) {
	t.Parallel()

	plain, err := BuildServerFrame(901, Object(Named("result", Int(0))), 1, nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertHex(t, plain, "100000000001038581a6726573756c7400")

	key := byteSequence(32)
	nonce := byteSequence(12)
	message := Object(
		Named("result", Int(0)),
		Named("role", Object(Named("name", String("Tester")))),
	)
	encrypted, err := BuildServerFrame(1001, message, 7, key, 1, bytes.NewReader(nonce))
	if err != nil {
		t.Fatal(err)
	}
	assertHex(t, encrypted,
		"3e0189fb08072afe27e6c5e64c86f4690ec7bb1fde82d0d0c3d88bf2899110b3"+
			"d2599e09fd9f6838f9d527d2d754cbbb347f90000102030405060708090a0b")

	length, offset, err := ReadVarint(encrypted, 0)
	if err != nil {
		t.Fatal(err)
	}
	if length != uint64(len(encrypted)-offset) || encrypted[offset] != 1 {
		t.Fatalf("帧长度或加密类型错误：长度=%d，偏移=%d", length, offset)
	}
}

func TestBuildServerFrameNonceError(t *testing.T) {
	t.Parallel()

	_, err := BuildServerFrame(1, Nil(), 1, byteSequence(32), 1, bytes.NewReader([]byte{1, 2}))
	if err == nil {
		t.Fatal("短 nonce 源应返回错误")
	}
}

func TestSplitCC2PPayload(t *testing.T) {
	t.Parallel()

	payload := append([]byte{1, 2, 3}, make([]byte, gxjycrypto.CC2PTagSize+gxjycrypto.CC2PNonceSize)...)
	ciphertext, tag, nonce, err := SplitCC2PPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ciphertext, []byte{1, 2, 3}) || len(tag) != 16 || len(nonce) != 12 {
		t.Fatalf("切分结果错误：%x %x %x", ciphertext, tag, nonce)
	}
	if _, _, _, err := SplitCC2PPayload(make([]byte, 27)); !errors.Is(err, ErrCC2PPayloadTooShort) {
		t.Fatalf("短载荷错误：%v", err)
	}
}

func TestParseAndDecodeClientPackets(t *testing.T) {
	t.Parallel()

	cipherPayload := mustDecodeHex(t,
		"89fb0803297227e3daf058723e727e06a719d64629d7dedc9794f7b0efdea4379"+
			"b1cd760806034b0d68b000102030405060708090a0b")
	first := append([]byte{1, 0x80, 0x01}, make([]byte, 128)...)
	first = append(first, cipherPayload...)
	later := append([]byte{1, 0}, cipherPayload...)

	if got := sha256.Sum256(first); hex.EncodeToString(got[:]) != "a2d801d05027714ea66f94cd7c1d6a7b606ab37cd8788b2936f7f645be250b43" {
		t.Fatalf("首包基准散列错误：%x", got)
	}
	if got := sha256.Sum256(later); hex.EncodeToString(got[:]) != "874b3c6d4a5e4146883153db6f360d8a1276e1d706b1f4409647ecfb3c264d42" {
		t.Fatalf("后续包基准散列错误：%x", got)
	}

	for _, test := range []struct {
		name      string
		packet    []byte
		rsaLength uint64
	}{
		{"首包", first, 128},
		{"后续包", later, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			packet, err := ParseClientPacket(test.packet)
			if err != nil {
				t.Fatal(err)
			}
			if packet.EncryptionType != 1 || packet.RSALength != test.rsaLength {
				t.Fatalf("外层字段错误：%#v", packet)
			}
			if !bytes.Equal(packet.Ciphertext, cipherPayload) {
				t.Fatalf("密文载荷错误：%x", packet.Ciphertext)
			}

			decoded, err := DecodeClientBody(packet.Ciphertext, byteSequence(32))
			if err != nil {
				t.Fatal(err)
			}
			want := Object(
				Named("msg", Object(Named("openid", String("x")))),
				Named("seq", Int(3)),
			)
			if decoded.Sequence != 3 || decoded.Command != 101 || !decoded.Message.Equal(want) {
				t.Fatalf("正文错误：%#v", decoded)
			}
		})
	}
}

func TestClientPacketErrors(t *testing.T) {
	t.Parallel()

	if _, err := ParseClientPacket(nil); !errors.Is(err, ErrFrameBodyTooShort) {
		t.Fatalf("空包错误：%v", err)
	}
	if _, err := ParseClientPacket([]byte{1, 0x80}); !errors.Is(err, ErrTruncated) {
		t.Fatalf("截断 Varint 错误：%v", err)
	}
	if _, err := ParseClientPacket([]byte{1, 3, 1}); !errors.Is(err, ErrRSABlockTruncated) {
		t.Fatalf("截断 RSA 错误：%v", err)
	}
	if _, err := DecodeClientBody(make([]byte, 27), byteSequence(32)); !errors.Is(err, ErrCC2PPayloadTooShort) {
		t.Fatalf("短 CC2P 错误：%v", err)
	}
}

func TestDecodeClientBodyFallbackAndErrors(t *testing.T) {
	t.Parallel()

	key := byteSequence(32)
	nonce := byteSequence(12)
	shortCipher, shortTag, err := gxjycrypto.CC2PEncrypt(key, nonce, []byte{1, 2, 3, 4, 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	shortPayload := append(append(shortCipher, shortTag...), nonce...)
	if _, err := DecodeClientBody(shortPayload, key); !errors.Is(err, ErrFrameBodyTooShort) {
		t.Fatalf("短正文错误：%v", err)
	}

	body := make([]byte, 0, 7)
	body = binary.BigEndian.AppendUint32(body, 9)
	body = binary.BigEndian.AppendUint16(body, 901)
	body = append(body, 0xc1)
	ciphertext, tag, err := gxjycrypto.CC2PEncrypt(key, nonce, body, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := append(append(ciphertext, tag...), nonce...)
	decoded, err := DecodeClientBody(payload, key)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.UsedTextFallback || decoded.Message.Kind != KindString || decoded.Message.Text != "�" {
		t.Fatalf("文本回退错误：%#v", decoded)
	}

	payload[len(ciphertext)] ^= 1
	if _, err := DecodeClientBody(payload, key); !errors.Is(err, gxjycrypto.ErrAuthentication) {
		t.Fatalf("标签错误：%v", err)
	}
}

func TestPlainFramePythonVectors(t *testing.T) {
	t.Parallel()

	shortFrame, err := BuildPlainFrame(901, Object(Named("result", Int(0))), 1)
	if err != nil {
		t.Fatal(err)
	}
	assertHex(t, shortFrame, "13100000000000000001038581a6726573756c7400")

	decoded, encryptionType, err := ParsePlainFrame(shortFrame)
	if err != nil {
		t.Fatal(err)
	}
	if encryptionType != 0 || decoded.Sequence != 1 || decoded.Command != 901 ||
		!decoded.Message.Equal(Object(Named("result", Int(0)))) {
		t.Fatalf("短明文帧错误：类型=%d，正文=%#v", encryptionType, decoded)
	}

	longFrame, err := BuildPlainFrame(1001, Object(Named("text", String(strings.Repeat("x", 250)))), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(longFrame) != 274 || longFrame[0] != 0xff {
		t.Fatalf("长明文帧长度错误：%d", len(longFrame))
	}
	if got := sha256.Sum256(longFrame); hex.EncodeToString(got[:]) != "835525090c858625a75a43f94c618a74fb9b27573bb5e3f2509bf6f9b3068a45" {
		t.Fatalf("长明文帧散列错误：%x", got)
	}
	decoded, _, err = ParsePlainFrame(longFrame)
	if err != nil || decoded.Sequence != 7 || decoded.Command != 1001 {
		t.Fatalf("长明文帧解析错误：%#v，%v", decoded, err)
	}
}

func TestPlainFrameDetectionAndErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		data []byte
		want bool
	}{
		{nil, false},
		{[]byte{1}, false},
		{[]byte{0xff, 0, 0, 0}, false},
		{[]byte{0xff, 0, 0, 0, 1}, true},
		{[]byte{2}, true},
	}
	for _, test := range cases {
		if got := LooksLikePlain(test.data); got != test.want {
			t.Fatalf("LooksLikePlain(%x) = %v，期望 %v", test.data, got, test.want)
		}
	}
	if _, _, err := ParsePlainFrame([]byte{1, 2}); !errors.Is(err, ErrFrameBodyTooShort) {
		t.Fatalf("短明文帧错误：%v", err)
	}
}

func assertHex(t *testing.T, got []byte, wantHex string) {
	t.Helper()
	want := mustDecodeHex(t, wantHex)
	if !bytes.Equal(got, want) {
		t.Fatalf("字节不一致：\n得到 %x\n期望 %x", got, want)
	}
}

func mustDecodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func byteSequence(size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}
