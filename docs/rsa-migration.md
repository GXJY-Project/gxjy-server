# RSA 握手迁移报告

状态：RSA 私钥加载、PKCS#1 v1.5 解密和会话密钥恢复已完成首轮迁移。

## 1. 迁移范围

Python 基线：

- `gxjy_server.py:105-117 rsa_decrypt_pkcs1`
- `gxjy_server.py:8758-8785 recover_session_key`
- 模块启动时的 PEM 优先加载。
- `server_rsa.json` 的 N、D、E 回退。

Go 实现：

- `internal/crypto/handshake.go`
- `internal/crypto/handshake_test.go`
- `internal/crypto/testdata/python_rsa_vector.json`

## 2. 私钥来源

Go 按以下顺序加载：

1. 未加密 PEM 私钥。
2. N、D、E JSON 回退。

支持的 PEM 格式：

- PKCS#1 RSA 私钥。
- PKCS#8 RSA 私钥。

PEM 私钥会执行结构校验。

回退 JSON 使用字段：

```json
{
  "n_hex": "...",
  "d_hex": "...",
  "e": 65537
}
```

代码没有硬编码生产密钥路径。

## 3. 解密行为

PEM 路径使用 Go 标准库 RSA PKCS#1 v1.5。

N、D、E 路径复制 Python 手写行为：

1. 密文按大端整数读取。
2. 执行 `c^d mod n`。
3. 补齐到模数长度。
4. 要求前缀为 `00 02`。
5. 查找第一个零分隔符。
6. 分隔符索引必须至少为 10。
7. 返回分隔符后的字节。

没有改成 OAEP。

## 4. 会话密钥规则

恢复顺序严格保持 Python：

1. 解密结果正好 32 字节时，直接返回。
2. 否则要求 ASCII 文本。
3. 去掉文本两端空白。
4. 文本至少包含 32 个字符。
5. 所有字符必须是十六进制。
6. 按十六进制解码并返回。

重要边界：

32 字节 ASCII 十六进制文本会先命中原始密钥分支。

它不会被解码为 16 字节。

该顺序已由测试固定。

## 5. Python 差分向量

测试夹具由独立 Python 标准库脚本生成。

夹具只含测试专用 RSA 密钥。

测试覆盖：

- 32 字节原始会话密钥。
- 64 字符十六进制会话密钥。
- 固定 PKCS#1 v1.5 编码。
- 固定 RSA 密文。
- N、D、E 回退解密。

Go 解密结果与 Python 完全一致。

没有读取项目中的真实私钥内容。

## 6. 其他测试

已测试：

- PKCS#1 PEM。
- PKCS#8 PEM。
- PEM 优先规则。
- 无效 PEM 后 JSON 回退。
- 原始密钥恢复。
- 十六进制密钥恢复。
- 32 字节分支优先级。
- 错误区块类型。
- 过短填充。
- 空 RSA 区块。
- 非 ASCII 文本。
- 非十六进制文本。
- 奇数长度十六进制文本。
- 无效 N、D、E。

## 7. 明确差异

| 场景 | Python | Go |
|---|---|---|
| PEM 私钥结构 | 由 `cryptography` 校验 | 由 `crypto/x509` 校验 |
| 回退错误 | 抛异常并由上层吞掉 | 返回可分类错误 |
| 日志 | 函数内部或上层记录 | 交给会话层记录 |
| 真实生产密钥 | 发布包运行时加载 | 本轮没有读取验证 |

成功路径保持一致。

错误如何影响连接，将在迁移 `Session.run` 时固定。

## 8. 验证结果

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -cover ./...
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./...
```

| 检查 | 结果 |
|---|---|
| Python RSA 向量 | 通过 |
| PKCS#1 PEM | 通过 |
| PKCS#8 PEM | 通过 |
| N/D/E 回退 | 通过 |
| 填充错误测试 | 通过 |
| 加密包覆盖率 | 87.4% |
| 协议包覆盖率 | 76.2% |
| `go vet` | 通过 |
| `go build` | 通过 |
| `gofmt` | 通过 |

## 9. 尚未验证

- 发布包真实 PEM 的 Go 加载。
- 发布包真实客户端 RSA 首包。
- 完整 WebSocket 登录握手。
- 密钥轮换包。

原因：本轮不读取真实私钥，也未启动模拟器。
