# 独立运行说明

## 目标

`github.com/GXJY-Project/gxjy-server` 不导入或执行 Python 代码。

当前本机配置引用 `data/` 下的资源、RSA 和协议表本地副本，不再依赖外部发布包路径。

这些路径用于保持官方客户端测试能力。

## 默认模式

默认配置只监听 `127.0.0.1`。

当前本机配置会加载已有 RSA 和协议表。

登录业务要求配置资源目录、Bundle 和资源密钥；缺失时明确启动失败。

它支持：

- HTTP 登录接口。
- `/healthz` 健康检查。
- WebSocket 社区明文协议。
- 与 Python 对齐的角色、英雄、编队、地图、系统解锁、活动和巅峰登录推送。
- 角色创建和改名存档。
- 客户端设置、名字池、背包和章节查询。

## 可选数据

| 配置 | 用途 | 缺少时 |
|---|---|---|
| `asset_root` | Cocos 配置资源 | 启动失败 |
| `bundle_config` | Bundle 映射 | 启动失败 |
| `rsa_pem` | 官方加密会话 | 只支持明文协议 |
| `rsa_json` | RSA 回退参数 | 不使用回退 |
| `protocol_map` | 完整命令名称 | 使用内置名称 |

资源和密钥必须来自已授权的本地副本。

RSA 私钥必须与测试客户端的公钥匹配。

## 数据库

当前入口为根目录 `main.go`，默认使用 SQLite `data/db/gxjy.sqlite`。
启动时由 `config` 初始化共享的 `global.Logger` 和 `global.DB`。
首次启动前运行 `go run . -db` 同步模型，再运行 `go run .` 启动服务。
命令行参数、配置覆盖及迁移命令均在 `flag/` 中处理，`router/` 注册 Gin 路由并组装 HTTP 与 WebSocket 服务。
`controllers/` 处理 HTTP 请求，`middlewares/` 通过共享 Logger 记录脱敏请求日志。
SQLite/PostgreSQL 共用 GORM 模型和 `repositories/player.Repo`，切换时修改连接配置。

## 当前边界

角色、英雄和装备快照、编队、背包、章节查询及相关存档副作用已进行 Python 差分。

战斗、探索交互、任务、抽卡、商店等请求仍未迁移。独立 1851、1951、2401 请求尚未接入，相应登录推送已对齐。

`GXJY_SKIP_GUIDE`、`GXJY_REPLAY_PROLOGUE` 与 Python 使用相同的开关语义，默认关闭。

这仍不是完整游戏服务端。

检查证据和差异边界见 [已有路径对齐记录](existing-paths-alignment.md)。
