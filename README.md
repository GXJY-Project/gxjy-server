# gxjy-server

《光隙解语》兼容服务端的 Go 实现。

项目代码可以独立构建和启动。

Go 代码不依赖 Python 模块。

当前本机配置使用 `data/` 下的资源、RSA 和协议表本地副本。

## 当前能力

- Varint 和 MessagePack。
- ChaCha20、Poly1305 和 CC2P。
- 官方与社区游戏帧。
- RSA 握手和密钥回退。
- REF、XXTEA 和 Cocos 配置表。
- Gin HTTP 登录入口和 WebSocket 游戏服务。
- 基础认证与登录推送。
- 与 Python 对齐的英雄快照、起手装备、编队、背包和章节查询。
- 名字池、角色创建/改名、客户端设置、体力及序章标记。
- GORM 玩家数据存储，开发默认使用 SQLite，可配置 PostgreSQL。
- 健康检查和安全关闭。

完整游戏业务仍在迁移。

B 服更新主脚本已完成离线解码与登录、帧格式及资源加载分析，见 [B 服脚本与协议证据](docs/bili-script-analysis.md)。原客户端公钥与服务端 RSA 密钥不匹配，本轮本地测试客户端已替换匹配公钥，登录结果见下文。

B 服启动“无网络”的本轮排查已修复本地测试客户端的登录地址和公钥，并处理了空更新清单删除补丁的问题，维护者已确认登录成功。当前仍使用模拟器中的本地脚本补丁，完整业务兼容尚未验收，见 [B 服连接排查](docs/bili-connection-diagnosis.md)。

B 服 33 张业务配置已实际加载，独立存档验证发现了装备引用、英雄属性、觉醒及背包响应编号差异，见 [B 服配置验证](docs/bili-runtime-validation.md)。与 Python 原型一致不代表符合 B 服。

当前支持的请求、差分范围及已知差异见 [已有路径对齐记录](docs/existing-paths-alignment.md)。官方客户端尚未进行本轮实机验收。

B 服 224 登录及主界面相关查询当前共受理 43 种请求，包含看板、公告、邮箱和维护者确认的协战/Rogue 初态。最新实现边界、验证和可复现的协议定位步骤见 [主界面查询与协作方法](docs/protocol-porting-workflow.md)，完整静态缺口见 [B224 CMD 清单](docs/bili224-command-coverage.md)。受理 CMD 不等于完整游戏系统或真实主界面已经验收。

最新加载排查确认客户端发生英雄、任务、装备和卡池配置断言，已按维护者授权切换 B224 资源配置并隔离现有存档中的无效旧记录。备份、验证范围及重启验收步骤见 [B224 加载阻塞修复](docs/bili224-loading-fix.md)，实机进入主界面仍待重测。

## 目录

```text
gxjy-server/
├── main.go              # 初始化与启动入口
├── config.json          # 默认运行配置
├── config/              # 配置、日志及数据库初始化
├── controllers/         # Gin HTTP 请求处理
├── global/              # 共享 Logger 和 DB
├── middlewares/         # Gin 请求日志
├── flag/                # 参数定义、解析、配置覆盖和命令执行
├── models/              # GORM 数据模型
├── repositories/player/ # 玩家数据操作
├── router/              # Gin 路由注册、资源组装和服务启动
├── data/                # 本地资源、密钥和存档
├── docs/                # 迁移和兼容报告
├── internal/assets/     # Cocos、REF 和 XXTEA
├── internal/crypto/     # CC2P 和 RSA
├── internal/protocol/   # 帧和 MessagePack
├── internal/server/     # HTTP、WebSocket 和游戏会话
├── internal/game/       # 登录相关业务与可注入时钟、随机源
├── internal/storage/    # 游戏状态序列化与历史差分存档工具
└── scripts/             # Windows 启动和检查脚本
```

## 依赖

| 包 | 固定版本 | 用途 |
|---|---:|---|
| `github.com/gin-gonic/gin` | `v1.12.0` | HTTP 路由及中间件 |
| `github.com/gorilla/websocket` | `v1.5.3` | WebSocket |
| `github.com/google/uuid` | `v1.6.0` | UUID |
| `github.com/sirupsen/logrus` | `v1.10.2` | 日志 |
| `gorm.io/gorm` | `v1.31.2` | ORM |
| `github.com/ncruces/go-sqlite3/gormlite` | `v0.34.0` | 无 CGO 的 SQLite 驱动 |
| `gorm.io/driver/postgres` | `v1.6.3` | PostgreSQL 驱动 |

## 启动

PowerShell：

```powershell
Set-Location 'E:\Go Project\gxjy\gxjy-server'
go run . -db
go run .
```

首次启动前用 `-db` 创建表；后续修改模型时也通过该命令同步，命令完成后退出，不加载游戏资源或监听端口。

查看版本或指定配置文件：

```powershell
go run . -v
go run . -f config.json
```

`scripts/run.ps1` 仍可作为命令包装使用。资源密钥优先读取 `asset_key_env` 指定的环境变量（默认 `GXJY_ASSET_KEY`），未设置时读取 `asset_key_file` 指定的本地文件（默认 `data/keys/asset.key`）。

默认地址：

| 服务 | 地址 |
|---|---|
| 健康检查 | `http://127.0.0.1:8080/healthz` |
| HTTP 登录 | `http://127.0.0.1:8080/player/login` |
| WebSocket | `ws://127.0.0.1:8765/game` |

停止服务请按 `Ctrl+C`。

`router.SetupRouter()` 注册登录接口，`router.SetupGameRouter()` 注册独立端口的 `/game`。控制器使用 `*gin.Context`；WebSocket 通过 Gin 的 `WrapH` 接入现有 gorilla 处理器。两个 Gin 路由器由 `http.Server` 承载，保留原有超时、退出信号和关闭流程。

请求日志继续使用 `global.Logger`，只记录方法与路径。登录响应沿用 `scripts/python_baseline.py` 中 `make_login_response` 的差分样本及已有 Go 行为，未知路径保留原有响应回退；WebSocket 入口保持 `/game` 精确匹配。

## 配置

程序默认读取工作目录中的 `config.json`；通过 `-f` 指定文件，`--config` 为同义参数。所有命令行参数均在 `flag/` 中定义和处理。

相对路径按配置文件所在目录解析。

```json
{
  "http_host": "127.0.0.1",
  "http_port": 8080,
  "ws_host": "127.0.0.1",
  "ws_port": 8765,
  "gate_host": "127.0.0.1",
  "database": {
    "driver": "sqlite",
    "dsn": "data/db/gxjy.sqlite",
    "max_idle_conns": 1,
    "max_open_conns": 1
  },
  "asset_root": "",
  "bundle_config": "",
  "asset_key_env": "GXJY_ASSET_KEY",
  "asset_key_file": "data/keys/asset.key",
  "rsa_pem": "",
  "rsa_json": "",
  "protocol_map": ""
}
```

以下参数示例在 `gxjy-server` 目录执行，命令行参数会覆盖配置值：

```powershell
go run . --http-port 18080 --ws-port 18765
```

## 可选资源

当前本机配置会加载已有 RSA 和协议表。

缺少资源目录、Bundle 或资源密钥时启动失败；不会以空英雄、空背包继续登录。

官方加密客户端需要匹配的 RSA 私钥。

真实配置表需要授权资源和资源密钥。

建议目录：

```text
data/assets/
data/keys/gate.pem
data/keys/server_rsa.json
data/protocol/protocol_map.json
```

启用资源表：

```powershell
$env:GXJY_ASSET_KEY = '<本地授权资源密钥>'
go run . `
  --asset-root '.\data\assets' `
  --bundle-config '.\data\assets\resources\config.json'
```

启用 RSA：

```powershell
go run . --rsa-pem '.\data\keys\gate.pem'
```

不要提交私钥、资源密钥、APK 或真实玩家存档。

## 手机连接

```powershell
go run . `
  --gate-host 192.168.1.10 `
  --http-host 0.0.0.0 `
  --ws-host 0.0.0.0
```

这会开放局域网访问。

只在可信测试网络使用。

## 数据库

默认使用 `data/db/gxjy.sqlite`，相对文件路径按配置文件目录解析，数据库文件及伴随文件已被 Git 忽略。

`config.InitLog` 和 `config.InitDB` 在启动时分别初始化 `global.Logger`、`global.DB`，服务运行期间统一复用。`repositories/player.Repo` 使用共享连接池查询和保存 `models.Player`，事务内传入 GORM 事务句柄。开发阶段通过 `-db` 调用 GORM `AutoMigrate` 同步模型。

切换 PostgreSQL 时，只需修改 `database.driver` 和 `database.dsn`，repo 和业务代码共用：

```json
"database": {
  "driver": "postgres",
  "dsn": "${GXJY_DATABASE_DSN}",
  "max_idle_conns": 2,
  "max_open_conns": 10
}
```

在本地环境变量 `GXJY_DATABASE_DSN` 中设置连接信息，不将实际密码写入配置或提交到 Git。连接切换不会自动搬运另一个数据库里的数据。

当前游戏服务仍共享一个角色 Runtime，启动入口沿用现有角色编号 `10001`；repo 支持按角色编号区分记录，尚未增加多账号会话路由。玩家状态按当前业务结构存于 `players.state` 的 `text` 字段，没有在本次改动中拆分英雄、背包等业务表。

服务启动使用数据库，不读取或导入旧 JSON 存档。`JSONStore` 和 `save_dir` 目前仅供已有差分测试及离线脚本使用。

## 检查

```powershell
.\scripts\check.ps1
```

脚本依次运行：

```text
gofmt -l <源码文件>
go test . ./config/... ./controllers/... ./flag/... ./global/... ./middlewares/... ./models/... ./repositories/... ./router/... ./internal/...
go vet . ./config/... ./controllers/... ./flag/... ./global/... ./middlewares/... ./models/... ./repositories/... ./router/... ./internal/...
go build . ./config/... ./controllers/... ./flag/... ./global/... ./middlewares/... ./models/... ./repositories/... ./router/... ./internal/...
```

详细边界见 `docs/standalone.md`。

Python 差分可以通过 `scripts/python_baseline.py` 重新生成；真实资源差分入口为 `scripts/check_python_diff.py`。运行时服务不依赖 Python。
