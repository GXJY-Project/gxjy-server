# 协议底座试迁移报告

状态：Varint、MessagePack 与 CC2P 已完成首轮迁移。

CC2P 详情见 `cc2p-migration.md`。

## 1. 本轮范围

已迁移 Python：

- `gxjy_protocol.py:14-22 write_varint`
- `gxjy_protocol.py:25-35 read_varint`
- `gxjy_protocol.py:40-94 mp_pack`
- `gxjy_protocol.py:97-201 mp_unpack`

Go 实现：

- `internal/protocol/value.go`
- `internal/protocol/varint.go`
- `internal/protocol/msgpack.go`

测试：

- `internal/protocol/varint_test.go`
- `internal/protocol/msgpack_test.go`

没有修改原 Python 文件。

## 2. 设计决定

### 2.1 不使用第三方 MessagePack

Python 实现只支持一个明确子集。自行迁移可以保持：

- 最小整数宽度。
- 字符串和二进制边界。
- float64 编码。
- float32 解码。
- 数组和映射头格式。
- 不支持标记的错误。

协议底座没有依赖第三方 MessagePack。当前服务层使用固定版本的 WebSocket 和 UUID 库。

### 2.2 使用显式 `Value`

没有用任意 `map[string]any` 作为协议核心值。

整数使用符号和绝对值。这样同时覆盖 Python 任意整数在 MessagePack 64 位范围内的表示。

映射使用有序 `[]Field`。这保留 Python `dict` 的插入顺序，并支持字节级 Golden 测试。

### 2.3 安全错误

Python 对截断输入通常抛出 `IndexError` 或 `struct.error`。Go 返回可分类错误：

- `ErrInvalidOffset`
- `ErrTruncated`
- `ErrVarintOverflow`
- `ErrInvalidValue`
- `ErrUnsupportedValue`
- `ErrNestingTooDeep`

成功输入的编码行为保持一致。错误类型不同，后续由会话层映射为 Python 的连接行为。

### 2.4 已知边界

- Go 当前限制嵌套深度为 128。
- Go 拒绝编码非法 UTF-8 字符串。
- 解码非法 UTF-8 时替换错误字节。
- 复合值不能作为映射键。
- 正整数最大为 `uint64`。
- 负整数最小为 `int64`。

这些限制覆盖当前协议数据。若真实流量违反，必须补充样本后调整。

## 3. Python 差分向量

本轮从原 `gxjy_protocol.py` 实际生成向量，再写入 Go 测试。

覆盖：

- `nil`、布尔值。
- 正整数所有宽度边界。
- 负整数所有宽度边界。
- float64。
- UTF-8 中文。
- 二进制。
- 数组。
- 有序映射。
- 嵌套对象。
- Varint 边界。

代表结果：

| 输入 | Python 字节 | Go 结果 |
|---|---|---|
| `128` | `cc80` | 一致 |
| `65536` | `ce00010000` | 一致 |
| `2^32` | `cf0000000100000000` | 一致 |
| `-33` | `d0df` | 一致 |
| `-32769` | `d2ffff7fff` | 一致 |
| `1.5` | `cb3ff8000000000000` | 一致 |
| `中文测试` | `ace4b8ade69687e6b58be8af95` | 一致 |
| Varint `300` | `ac02` | 一致 |

## 4. 实际验证

工具：Go 1.27.3，路径 `C:\Program Files\Go\bin\go.exe`。

执行：

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -cover ./...
```

结果：通过，语句覆盖率 71.2%。

执行：

```powershell
& 'C:\Program Files\Go\bin\go.exe' test ./...
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./...
```

结果：全部通过。

执行：

```powershell
& 'C:\Program Files\Go\bin\go.exe' test -race ./internal/protocol
```

结果：未运行。当前 `CGO_ENABLED=0`，Go race 检查要求 CGO。

格式检查：`gofmt -d` 无输出。

## 5. 下一模块

协议编码、CC2P、游戏帧和 RSA 握手已完成试迁移。

建议按计划继续：

1. Cocos UUID。
2. Bundle 路径映射。
3. GZip 和 JSON 载荷。
4. 配置表定位。

下一模块先使用固定资源样本和 SHA-256。
