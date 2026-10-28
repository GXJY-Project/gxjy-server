package protocol

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// TestBiliClientWireVectors 使用 B 服更新脚本实际生成的合成帧验证传输层
// 来源与复现方式见 docs/bili-script-analysis.md 和 scripts/check_bili_wire.cjs
func TestBiliClientWireVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/bili_wire_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SourceHash string `json:"source_sha256"`
		KeyHex     string `json:"key_hex"`
		Cases      []struct {
			Name      string `json:"name"`
			Command   uint16 `json:"command"`
			Sequence  uint32 `json:"sequence"`
			RSALength uint64 `json:"rsa_length"`
			PacketHex string `json:"packet_hex"`
		} `json:"cases"`
		Server struct {
			Command  uint16 `json:"command"`
			Sequence uint32 `json:"sequence"`
			NonceHex string `json:"nonce_hex"`
			FrameHex string `json:"frame_hex"`
		} `json:"server"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SourceHash != "a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0" {
		t.Fatal("客户端证据版本不匹配")
	}
	want := map[string]Value{
		"first": Object(Named("msg", Object(
			Named("openid", String("synthetic-user")),
			Named("token", String("synthetic-token")),
			Named("is_reconnected", Bool(false)))), Named("seq", Int(1))),
		"later": Object(Named("msg", Object(
			Named("clientTimestamp", Int(1700000000000)))), Named("seq", Int(2))),
		"reconnect": Object(Named("msg", Object(
			Named("is_reconnected", Bool(true)))), Named("seq", Int(3))),
	}
	if len(fixture.Cases) != len(want) {
		t.Fatal("客户端帧用例缺失")
	}
	key := mustDecodeHex(t, fixture.KeyHex)
	for _, sample := range fixture.Cases {
		t.Run(sample.Name, func(t *testing.T) {
			packet, err := ParseClientPacket(mustDecodeHex(t, sample.PacketHex))
			if err != nil {
				t.Fatal(err)
			}
			if packet.EncryptionType != 1 || packet.RSALength != sample.RSALength {
				t.Fatal("客户端首包或重连 RSA 布局不匹配")
			}
			body, err := DecodeClientBody(packet.Ciphertext, key)
			if err != nil {
				t.Fatal(err)
			}
			if body.Command != sample.Command || body.Sequence != sample.Sequence ||
				body.UsedTextFallback || !body.Message.Equal(want[sample.Name]) {
				t.Fatal("客户端帧的消息内容、命令或序号不一致")
			}
		})
	}

	// 下行预期由独立加密器生成，并已通过原客户端 unpackMsg 验证
	server := fixture.Server
	frame, err := BuildServerFrame(server.Command,
		Object(Named("result", Int(0)), Named("synthetic", Bool(true))),
		server.Sequence, key, 1, bytes.NewReader(mustDecodeHex(t, server.NonceHex)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(frame, mustDecodeHex(t, server.FrameHex)) {
		t.Fatal("Go 下行与原客户端可读帧不一致")
	}
}
