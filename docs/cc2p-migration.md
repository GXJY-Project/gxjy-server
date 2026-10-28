# CC2P 加密迁移报告

状态：首轮迁移完成。

## 1. 迁移范围

Python 基线：

- `gxjy_protocol.py:206-207 _rotl32`
- `gxjy_protocol.py:210-226 _chacha_block`
- `gxjy_protocol.py:229-235 chacha20`
- `gxjy_protocol.py:255-264 _poly1305_mac`
- `gxjy_protocol.py:267-272 cc2p_tag`
- `gxjy_protocol.py:275-278 cc2p_encrypt`
- `gxjy_protocol.py:281-285 cc2p_decrypt`

Go 实现：

- `internal/crypto/cc2p.go`
- `internal/crypto/cc2p_test.go`

## 2. 实现方式

使用 Go 标准库实现。

没有新增第三方依赖。

实现包含：

- RFC 7539 ChaCha20。
- 二十轮块函数。
- 32 位自然溢出。
- Poly1305 clamp。
- AAD 十六字节填充。
- 小端长度尾部。
- 十六字节标签。
- 常量时间标签比较。
- 先验标签，再解密。

为兼容 Python，过长密钥只取前 32 字节。

过长 nonce 只取前 12 字节。

短输入返回明确错误。

## 3. 差分向量

本轮实际运行 Python 基线生成以下向量：

| 向量 | 密文 | 标签 |
|---|---|---|
| 空消息 | 空 | `295a498b8841a1c5f55d4d606f731159` |
| `hello` | `e19e646c46` | `f5939695c670586d63a69f58c41396dd` |
| `hello` 加 AAD | `e19e646c46` | `1c058502f3bef6fe13240770443dcb53` |
| 64 字节 | Python 固定向量 | `12ccd3f87522adf137c06f3496d9562e` |
| 65 字节加 AAD | Python 固定向量 | `65ec3e88d222d0d28bdd62fe956f40e0` |

Go 测试逐字节比较密文和标签。

所有向量一致。

## 4. RFC 验证

测试包含 RFC 7539 第 2.8.2 节 AEAD 向量。

密文和标签完全一致。

测试还包含 RFC ChaCha20 块向量。

64 字节密钥流完全一致。

## 5. 错误测试

已测试：

- 篡改标签。
- 缩短标签。
- 篡改 AAD。
- 短密钥。
- 短 nonce。
- 过长密钥兼容。
- 过长 nonce 兼容。
- 输入切片不被修改。

Go 使用常量时间比较。

Python 当前使用普通字节比较。

成功行为一致。

失败结果都拒绝解密。

## 6. 验证命令

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -cover ./internal/crypto
& 'C:\Program Files\Go\bin\go.exe' test -cover ./...
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./...
```

结果：

| 检查 | 结果 |
|---|---|
| CC2P 测试 | 通过 |
| CC2P 覆盖率 | 94.4% |
| 全部测试 | 通过 |
| `go vet` | 通过 |
| `go build` | 通过 |
| `gofmt` | 通过 |

## 7. 后续状态

以下功能已在游戏帧阶段迁移：

- `split_cc2p_payload`
- `build_server_frame`
- `parse_client_packet`
- `decode_client_body`

尚未迁移 RSA 会话密钥恢复。
