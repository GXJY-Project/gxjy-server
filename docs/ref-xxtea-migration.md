# REF 与 XXTEA 迁移报告

状态：纯算法和两种 REF 容器已完成首轮迁移。

## 1. 迁移范围

Python 基线：

- `gxjy_decrypt.py:15-23 _long2str`
- `gxjy_decrypt.py:26-33 _str2long`
- `gxjy_decrypt.py:36-52 xxtea_decrypt_raw`
- `gxjy_decrypt.py:55-70 decode_file`
- `read_config_table.py:217-238 unpack_ref`
- `patch_cvguard.py:49-103 xxtea_dec/xxtea_enc`

Go 实现：

- `internal/assets/xxtea.go`
- `internal/assets/ref.go`
- `internal/assets/xxtea_test.go`
- `internal/assets/ref_test.go`

## 2. XXTEA 行为

实现保留：

- 小端 32 位字。
- `0x9E3779B9` delta。
- `6 + 52/(n+1)` 轮数。
- 32 位自然溢出。
- 明文长度尾字。
- 密钥不足十六字节时补零。
- 密钥超过十六字节时只用前十六字节。

Go 同时提供加密和解密。

服务端运行只需要解密。

加密用于固定差分测试。

## 3. Python 宽容行为

Python `_long2str(..., True)` 遇到无效长度尾字时返回空字节。

它不会抛出错误。

Go 已保持这个结果：

```text
无效长度尾字 -> 空字节，错误为空
```

未按四字节对齐的密文仍返回明确错误。

该差异用于区分容器损坏和 Python 宽容尾长。

## 4. REF 容器

### 无头布局

```text
[REF:]
[版本四字节]
[XXTEA 数据]
```

解密结果必须以以下内容开头：

- GZip：`1f 8b`
- JSON 对象：`{`
- JSON 数组：`[`

### 六字节内层头布局

```text
[REF:\0\0\0\2]
[保留两字节]
[加密长度:u32 大端]
[XXTEA 数据]
[未加密尾部]
```

Go 会先尝试无头布局。

失败后再尝试六字节内层头。

这与 Python `unpack_ref` 顺序一致。

## 5. 密钥处理

Go 源码没有硬编码资源密钥。

调用方必须传入密钥。

测试使用 `test-key`。

本轮没有读取真实资源密钥文件。

## 6. Python 差分向量

本轮实际运行 Python `xxtea_enc` 生成向量。

| 明文 | Python 密文 | Go 结果 |
|---|---|---|
| 空 | `885774a9` | 一致 |
| `hello` | `8907b60356a3f8ef0bd99a13` | 一致 |
| `12345678` | `09e1a1a674e921da55094cdd` | 一致 |
| `{"a":1}` | `1be88660968d0cce1284e88b` | 一致 |

REF 向量：

| 布局 | 内容 | 结果 |
|---|---|---|
| 无头 | GZip JSON | 字节一致 |
| 有头 | GZip JSON 加尾部 | 字节一致 |
| 无头 | 明文 JSON | 字节一致 |

## 7. 错误测试

已测试：

- 空密文。
- 非四字节密文。
- 无效长度尾字。
- 错误 REF 签名。
- 空 REF 正文。
- 声明长度超过剩余数据。
- 声明长度未按四字节对齐。
- 文件读取。
- 输入切片不被修改。

## 8. 验证结果

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -cover ./internal/assets
& 'C:\Program Files\Go\bin\go.exe' test -cover ./...
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./...
```

| 检查 | 结果 |
|---|---|
| Python XXTEA 向量 | 通过 |
| 无头 REF | 通过 |
| 有头 REF | 通过 |
| 资产包覆盖率 | 95.3% |
| 全部测试 | 通过 |
| `go vet` | 通过 |
| `go build` | 通过 |
| `gofmt` | 通过 |

## 9. 尚未验证

- 真实 APK 配置资源。
- 原始资源密钥的运行时注入。
- GZip 解压后的配置 JSON。
- Cocos 数组容器。
- `_subJsonCount` 合并。
- `_subFiles` 递归。

下一步应迁移 Cocos UUID 和表定位。
