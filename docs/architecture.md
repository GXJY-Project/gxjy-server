# Go 迁移架构

以下为早期设计草案。2026-10-25 的实际结构是 `internal/server` 处理网络、`internal/game` 处理已有登录相关业务、`internal/assets` 读取配置、`internal/storage` 负责落盘，见 [已有路径对齐记录](existing-paths-alignment.md)。其余目录不因本草案预先创建。

## 1. 设计目标

1. 保留 Python 当前可观察行为。
2. 先迁移底层，再迁移业务。
3. 每个模块都能独立差分测试。
4. 原 Python 发布包保持只读。
5. 不引入无证据的业务改写。
6. 默认保护密钥、存档和管理端点。

## 2. 建议目录

```text
gxjy-server/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── assets/
│   │   ├── cocos.go
│   │   ├── ref.go
│   │   └── xxtea.go
│   ├── config/
│   │   ├── runtime.go
│   │   ├── table.go
│   │   ├── cache.go
│   │   └── value.go
│   ├── crypto/
│   │   ├── cc2p.go
│   │   └── handshake.go
│   ├── protocol/
│   │   ├── varint.go
│   │   ├── msgpack.go
│   │   ├── object.go
│   │   ├── frame.go
│   │   └── mapping.go
│   ├── transport/
│   │   ├── httpserver/
│   │   └── websocket/
│   ├── storage/
│   │   ├── player.go
│   │   └── file_store.go
│   ├── session/
│   │   ├── session.go
│   │   └── login.go
│   ├── dispatcher/
│   │   ├── dispatcher.go
│   │   └── registry.go
│   └── game/
│       ├── player/
│       ├── hero/
│       ├── team/
│       ├── reward/
│       ├── inventory/
│       ├── equipment/
│       ├── battle/
│       ├── explore/
│       ├── chapter/
│       ├── task/
│       ├── gacha/
│       ├── shop/
│       ├── stamina/
│       └── admin/
├── tests/
│   ├── fixtures/
│   │   ├── protocol/
│   │   ├── config/
│   │   ├── saves/
│   │   └── messages/
│   └── integration/
├── docs/
│   ├── migration-analysis.md
│   ├── architecture.md
│   ├── migration-plan.md
│   └── compatibility.md
├── go.mod
└── README.md
```

这些目录按真实边界建立。业务包只在迁移对应功能时创建。不会一次生成空目录或占位实现。

## 3. 依赖规则

```text
assets <- config
protocol <- crypto
storage <- config
session <- protocol + storage
business <- config + storage
scheduler/dispatcher <- session + business
transport <- session
cmd <- all wiring
```

禁止规则：

- `protocol` 不依赖 `game`。
- `storage` 不依赖 `transport`。
- `game` 不直接写 socket。
- `game` 不读取环境变量。
- `config` 不修改玩家状态。
- 子业务包不互相循环依赖。

跨业务操作通过小型服务组合，不预先创建大型接口层。

## 4. 核心类型

### 4.1 动态配置值

配置表来自反编译 JSON，字段形状不稳定。初期使用受控联合值：

```go
type Kind uint8

type Value struct {
    Kind   Kind
    Bool   bool
    Int    int64
    Uint   uint64
    Float  float64
    String string
    Bytes  []byte
    Array  []Value
    Object Object
}

type Field struct {
    Key   string
    Value Value
}

type Object []Field
```

`Object` 保存插入顺序。它用于协议 Golden 和需要精确顺序的输出。业务稳定后可转换为结构体。

### 4.2 玩家状态

玩家状态使用一个根结构体，并保留扩展字段：

```go
type PlayerState struct {
    RoleName string
    Level    int
    Currency map[string]int64
    Items    map[string]int64
    // 迁移到对应模块时逐步补齐。
    Extra    map[string]config.Value
}
```

不允许把整个业务继续做成全局 `map[string]any`。

### 4.3 状态存储

```go
type Store struct {
    mu    sync.RWMutex
    state PlayerState
    file  FileStore
}
```

命令处理使用显式事务函数。需要保持原顺序时，事务包含状态修改、存档决定和待发送事件生成。真正的网络发送在锁外顺序执行。

### 4.4 可注入环境

```go
type Clock interface {
    Now() time.Time
}

type RNG interface {
    Intn(int) int
    Uint64() uint64
}
```

游戏随机和加密随机必须分离。加密 nonce 只使用 `crypto/rand.Reader`。

## 5. 协议设计

### 5.1 编码顺序

1. 业务生成有序协议对象。
2. MessagePack 编码。
3. 添加大端 seq 和 cmd。
4. CC2P 加密。
5. 拼接 tag 和 nonce。
6. 添加 encType 和 Varint 长度。
7. WebSocket 封装。

### 5.2 解码顺序

1. WebSocket 解帧。
2. 读取 encType 和 RSA 长度。
3. 首包或轮换包恢复密钥。
4. 切分 ciphertext、tag、nonce。
5. 先验 tag。
6. 解密。
7. 读取大端 seq 和 cmd。
8. MessagePack 解码。
9. 分发。

### 5.3 错误兼容

协议函数返回带分类的错误。会话层决定原版行为：记录、忽略、失败回包或断开。协议层不能自行关闭连接。

## 6. 分发设计

Python 使用超长 `if/elif`。Go 版采用显式注册表：

```go
type Handler func(*Context, protocol.Value) (Result, error)

type Registry map[uint16]Handler
```

`Result` 同时表达：

- 当前命令回包。
- 按顺序发送的主动推送。
- 是否存档。
- 是否关闭连接。

注册表不会自动推导业务含义。每个 handler 必须对应 Python 分支，并在测试中记录 Python 行号。

未知消息保持当前行为：记录 `UNHANDLED`，返回成功外壳和空 `data`。以后改变该行为必须单独评审。

## 7. 存档设计

第一版先复制 Python 的字段合并和保护规则。

写入流程建议：

1. 在同目录创建临时文件。
2. 写入完整 JSON。
3. `Sync` 文件。
4. 复制当前有效存档到 `.prev`。
5. 原子替换主存档。
6. 更新差异备份。

原子写是安全修复。它会改变故障行为，但不改变成功路径。兼容报告必须单列。

JSON 编码要求：

- UTF-8。
- 两空格缩进。
- 不转义中文。
- 明确区分缺失、null 和空容器。
- 数字不得先经过 float64。

## 8. 配置表设计

表加载保持以下规则：

1. 从 bundle `paths` 查 `config/data/<Table>`。
2. 解压 22 字符 Cocos UUID。
3. 定位 import JSON。
4. 解 REF 容器。
5. 解 GZip。
6. 读取普通 JSON 或 Cocos 数组容器。
7. 合并主表。
8. 合并 `_sub1..N`。
9. 递归合并 `_subFiles`。
10. 删除元字段。

缓存键暂时保留 `name.mtime_size.json.gz`。完成兼容验证后，再考虑内容哈希。

`TableStore` 对加载成功和失败都应有明确状态。Python 的预热会记录失败并继续。Go 首版保持此行为，但启动报告必须列出失败表。

## 9. 网络与安全模式

提供两个运行模式：

| 模式 | 默认监听 | GM | 用途 |
|---|---|---|---|
| 安全默认 | `127.0.0.1` | 关闭 | 本机运行 |
| Python 兼容 | 显式配置 | 可显式开启 | 差分测试 |

管理凭据、私钥路径、存档路径和资源路径全部来自参数或环境。仓库只放示例配置。

## 10. 不在首轮迁移范围

- 重写反编译客户端 JS。
- 替换或重新分发 APK。
- 修复没有证据支持的业务规则。
- 把空社交契约扩展为完整多人系统。
- 数据库、多用户和分布式部署。
- Web 管理后台。

这些功能会扩大行为面，不能混入兼容迁移。
