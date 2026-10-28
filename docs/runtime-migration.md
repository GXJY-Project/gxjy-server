# Cocos 配置与可运行服务迁移报告

> 本文保留早期迁移记录。2026-10-25 的现状见 [已有路径对齐记录](existing-paths-alignment.md)：已有业务占位已替换，资源缺失改为启动失败，真实配置与双协议自动化差分已通过，尚无本轮实机验收。

## 范围

本轮在不修改 Python 原版的前提下，完成以下 Go 功能：

- Cocos 压缩 UUID 解码。
- Bundle `paths` 与 `uuids` 映射。
- 普通 JSON、gzip 和 REF 配置读取。
- Cocos `cc.JsonAsset` 数组载荷解包。
- `_subJsonCount` 子表合并。
- `_subFiles` 递归合并。
- HTTP 登录服务。
- WebSocket 游戏服务。
- 官方 RSA、CC2P 会话接入。
- 社区明文 MessagePack 会话接入。
- 基础认证和登录推送。
- 心跳、角色名和基础空数据响应。
- 基础角色字段本地存档。
- 原子替换、上一版备份和十份快照。
- 信号关闭和双端口启动。

## 代码位置

| 功能 | Go 文件 |
|---|---|
| Cocos 配置表 | `internal/assets/table.go` |
| HTTP 登录 | `internal/server/login.go` |
| WebSocket 帧 | `internal/server/websocket.go` |
| 游戏会话 | `internal/server/session.go` |
| 登录载荷 | `internal/server/protocol_values.go` |
| 服务生命周期 | `internal/server/server.go` |
| 命令入口 | `cmd/server/main.go` |

## 第三方依赖

| 包 | 固定版本 | 替代内容 |
|---|---:|---|
| `github.com/gorilla/websocket` | `v1.5.3` | 手写握手、控制帧和消息帧 |
| `github.com/google/uuid` | `v1.6.0` | 手写 UUID 格式化 |
| `github.com/sirupsen/logrus` | `v1.10.2` | 标准库 Slog 日志 |

Cocos 的 22 字符内容使用标准 Base64 解码。

解码后的 16 字节交给 UUID 库处理。


## 已实现登录链

```text
HTTP getResource/login/enter
              ↓
       WebSocket /game
              ↓
  官方 RSA+CC2P 或社区明文
              ↓
        MSG_TYPE_AUTH
              ↓
角色、登录、英雄、队伍、地图、解锁等推送
```

登录推送顺序保持 Python 当前实现：

1. `MSG_TYPE_AUTH`。
2. `MSG_ROLE_INFO_RES`。
3. `MSG_LOGIN_INFO_RES`。
4. `MSG_HEROS_RES`。
5. `MSG_TEAMS_RES`。
6. `MSG_WORLD_MAP`。
7. `MSG_SYSTEM_UNLOCK_RES`。
8. `MSG_PUSH_REMINDS`。
9. `MSG_RED_POINTS_RES`。
10. `MSG_SYNC_BASE_INFO`。
11. `MSG_REQ_ACTIVITIES`。

## 安全边界

- 默认仅监听 `127.0.0.1`。
- 登录令牌每次启动随机生成。
- 没有迁移无认证 `/gm` 接口。
- 资源密钥只从 `GXJY_ASSET_KEY` 读取。
- Go 源码和测试不含真实资源密钥。
- 测试没有读取真实资源密钥。

## 验证结果

| 检查 | 结果 |
|---|---|
| Cocos UUID 测试 | 通过 |
| 主表和子表合并 | 通过 |
| gzip 配置载荷 | 通过 |
| REF 缺少密钥报错 | 通过 |
| HTTP 资源版本回显 | 通过 |
| HTTP gate 地址 | 通过 |
| 协议重名过滤 | 通过 |
| 社区明文认证链 | 11 条顺序通过 |
| 新包定向测试 | 通过 |
| 临时程序构建 | 通过 |
| 本机 HTTP 启动冒烟 | 通过 |

真实 Cocos 表尚未读取。

缺少条件是授权资源密钥。

官方客户端 RSA 登录尚未实机验证。

缺少条件是真实客户端连接测试。

## 当前限制

当前程序是可启动的迁移底座。

它还不是完整游戏服务端。

以下内容仍未完整迁移：

- 完整玩家存档，仅基础角色字段已保存。
- 英雄、装备和背包业务。
- 章节与战斗数据。
- 探索场景业务。
- 商店、任务和抽卡。
- 完整请求响应形状。
- 客户端端到端验收。

空载荷只保证基础请求能收到响应。

不得将这些功能标记为与 Python 一致。
