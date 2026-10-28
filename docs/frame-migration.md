# 游戏帧迁移报告

状态：官方帧和社区明文帧已完成首轮迁移。

## 1. 迁移范围

Python 基线：

- `gxjy_protocol.py:288-292 split_cc2p_payload`
- `gxjy_protocol.py:297-314 build_server_frame`
- `gxjy_protocol.py:317-334 parse_client_packet`
- `gxjy_protocol.py:337-351 decode_client_body`
- `gxjy_server.py:406-440 build_plain_frame`
- `gxjy_server.py:443-452 looks_like_plain`
- `gxjy_server.py:455-469 parse_plain_frame`

Go 实现：

- `internal/protocol/frame.go`
- `internal/protocol/frame_test.go`

## 2. 官方帧布局

服务端下行：

```text
[varint(1 + payloadLen)]
[encType]
[payload]
```

加密正文：

```text
[seq:u32 大端]
[cmd:u16 大端]
[msgpack]
```

加密载荷：

```text
[ciphertext]
[tag:16]
[nonce:12]
```

客户端上行：

```text
[encByte]
[varint(rsaLength)]
[rsaBlock]
[cc2p payload]
```

首包的 RSA 区块和认证消息位于同一包。

## 3. 社区明文帧

短帧：

```text
[length:u8]
[encByte:0x10]
[reserved:u32]
[seq:u32 大端]
[cmd:u16 大端]
[msgpack]
```

长帧：

```text
[0xff]
[length:u32 大端]
[encByte:0x10]
[reserved:u32]
[seq:u32 大端]
[cmd:u16 大端]
[msgpack]
```

协议判断保留 Python 规则。

首字节 `0x01` 视为官方协议。

其他普通长度视为社区协议。

## 4. 实现决定

### 固定 nonce 测试

生产默认使用 `crypto/rand.Reader`。

测试可注入固定 nonce 来源。

这样能逐字节比较 Python 输出。

### 有序 MessagePack

帧编码继续使用有序 `Value`。

Python 字典字段顺序得以保留。

### 错误回退

MessagePack 解码失败时，Python 返回替换后的 UTF-8 文本。

Go 保留此回退，并记录 `UsedTextFallback=true`。

### 安全差异

Python 声明超长 RSA 区块时会静默切片。

Go 会返回 `ErrRSABlockTruncated`。

成功输入完全保持一致。

会话层后续决定该错误如何影响连接。

## 5. Python 差分向量

本轮实际运行 Python 生成帧向量。

| 向量 | 长度 | Go 结果 |
|---|---:|---|
| 官方明文下行 | 17 | 字节一致 |
| 官方加密下行 | 63 | 字节一致 |
| 社区短帧 | 21 | 字节一致 |
| 社区长帧 | 274 | SHA-256 一致 |
| 官方首包 | 185 | SHA-256 一致 |
| 官方后续包 | 56 | SHA-256 一致 |

固定官方加密帧：

```text
3e0189fb08072afe27e6c5e64c86f4690ec7bb1fde82d0d0c3d88bf2899110b3d2599e09fd9f6838f9d527d2d754cbbb347f90000102030405060708090a0b
```

## 6. 测试范围

已测试：

- 官方明文下行帧。
- 官方加密下行帧。
- 固定 nonce。
- CC2P 载荷切分。
- 首包 128 字节 RSA 区块。
- 后续包零长度 RSA 区块。
- 客户端正文解密。
- 大端 seq 和 cmd。
- MessagePack 文本回退。
- 篡改标签拒绝。
- 截断 Varint。
- 截断 RSA 区块。
- 截断 CC2P 载荷。
- 社区长短帧。
- 社区协议识别。

## 7. 验证结果

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -cover ./...
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./...
```

| 检查 | 结果 |
|---|---|
| 协议测试 | 通过 |
| 协议覆盖率 | 76.2% |
| 加密测试 | 通过 |
| 加密覆盖率 | 94.4% |
| `go vet` | 通过 |
| `go build` | 通过 |
| `gofmt` | 通过 |

## 8. 后续状态

以下功能已在 RSA 阶段迁移：

- RSA PKCS#1 v1.5 解密。
- PEM 私钥加载。
- N/D/E 回退。
- 会话密钥恢复。

尚未迁移：

- WebSocket 帧。
- `Session.run`。

下一步按计划迁移 REF 与 XXTEA。
