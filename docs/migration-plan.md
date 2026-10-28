# Go 分阶段迁移计划

状态更新（2026-10-25）：协议底座和已有 Go 登录路径已有实现及差分证据，见 [已有路径对齐记录](existing-paths-alignment.md)。下列分阶段内容保留为后续完整迁移规划，不能当作当前完成状态。

## 1. 总原则

每个阶段都按以下循环执行：

1. 锁定 Python 代码范围。
2. 记录输入、输出和副作用。
3. 生成固定测试向量。
4. 编写最小 Go 实现。
5. 运行 Go 单元测试。
6. 运行 Python/Go 差分。
7. 记录差异和证据。
8. 通过后再进入下一模块。

不会先生成全部 Go 代码。

## 2. 阶段零：工程准备

### 工作

- 安装或配置 Go。
- 确认目标 Go 版本。
- 创建 `go.mod`。
- 创建严格 `.gitignore`。
- 建立 fixture 生成规则。
- 只引用原 Python 路径，不复制敏感资产。

### 验收

```powershell
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

### 门槛

- APK、资源、存档、日志、PEM、keystore 不被 Git 跟踪。
- Go 版本被记录。
- 没有业务占位实现。

## 3. 阶段一：协议底座

### 1A Varint 与 MessagePack

迁移：

- `write_varint`
- `read_varint`
- `mp_pack`
- `mp_unpack`

测试：

- Python 现有自测向量。
- 所有整数边界。
- 空值、空容器和嵌套结构。
- 非法类型和截断输入。
- 原始字节差分。

通过标准：确定输入下，Go 字节必须等于 Python 字节。

### 1B ChaCha20-Poly1305

迁移：

- `_chacha_block`
- `chacha20`
- `_poly1305_mac`
- `cc2p_tag`
- `cc2p_encrypt`
- `cc2p_decrypt`

测试：

- RFC 7539 向量。
- 空明文。
- 多个 64 字节边界。
- 错误 tag。
- 错误 nonce 长度。
- Python/Go 字节差分。

### 1C 游戏帧

迁移：

- `build_server_frame`
- `parse_client_packet`
- `decode_client_body`
- plain 帧函数

测试：

- 大端 seq/cmd。
- 首包 RSA 区块加密文。
- 后续 `u=0` 帧。
- encType 0、1、2。
- 截断和篡改帧。

### 1D RSA 握手

迁移：

- PEM 私钥加载。
- PKCS#1 v1.5 解密。
- 原始 32 字节密钥。
- 十六进制文本密钥。
- N/D/E 回退兼容。

测试只能使用测试密钥。

## 4. 阶段二：资产与配置表

### 2A REF 与 XXTEA

迁移两种容器规则：

- 配置 REF 容器。
- JSC 容器仅在需要时迁移。

验证：固定样本解密后 SHA-256 一致。

### 2B Cocos 表定位

迁移 UUID 解压和 bundle 路径映射。

验证：已知表定位路径一致。

### 2C 表加载与合并

迁移：

- 普通 JSON。
- Cocos 数组容器。
- `_subJsonCount`。
- `_subFiles` 递归。
- 缓存读取和失效。

验证：选择关键表比较行数、键集合和代表行：

- `SERVER_Protocol`
- `RewardConfig`
- `TaskBase`
- `HeroBase`
- `FightPoint`
- `ExploreCaseDo`
- `UnlockSystem`

## 5. 阶段三：存档与基础状态

迁移：

- `PLAYER_DEFAULTS`。
- 旧存档字段合并。
- currency 默认值补齐。
- custom 子结构归一化。
- 序章标记处理。
- 真实进度防覆盖。
- `.prev` 和十份备份。

Go 版增加原子写和锁。

差分比较：

- 不存在的存档。
- 空存档。
- 老版本缺字段存档。
- 非法类型存档。
- 有进度存档防覆盖。
- 序章开关组合。

## 6. 阶段四：网络与会话

迁移：

- HTTP 登录端点。
- 热更新空 manifest。
- WebSocket Upgrade。
- WS 帧读写。
- `Session.run`。
- sequence 管理。
- 官方和 plain 双协议。
- 登录主动推送顺序。

验证：

- `net.Pipe` 单元测试。
- 原始 HTTP 请求响应测试。
- Python 和 Go 各启动临时端口。
- 同一客户端探针分别连接。
- 比较命令序列和解密 payload。

管理 `/gm` 默认关闭。兼容测试显式开启。

## 7. 阶段五：无随机基础业务

按依赖顺序：

1. 玩家与角色快照。
2. 系统解锁。
3. 编队。
4. 背包基础。
5. 奖励展开与发放。
6. 英雄养成。
7. 装备养成。
8. 体力恢复。

每个命令建立独立 fixture。输入包含配置快照和起始存档。输出包含回包、主动推送和结束存档。

## 8. 阶段六：关卡与探索

按顺序：

1. 章节列表和进度。
2. 主线战斗准备。
3. 试炼战斗准备。
4. 塔战斗准备。
5. 战斗结算。
6. 世界地图。
7. 场景单位。
8. case 状态机。
9. 探索战斗和宝箱。
10. 任务联动。

必须保留登录、1851、1012、1301 的顺序约束。

验证分两层：

- 无头协议探针。
- 模拟器实机流程。

探索三件套未验收前只能标记“代码已迁移，行为待验证”。

## 9. 阶段七：随机业务

迁移：

- 抽卡池。
- 权重选择。
- 保底和 UP。
- 随机角色名。
- 随机装备属性。

测试使用注入 RNG。差分测试比较同一抽样输入后的决策，不直接比较 Python 与 Go 的伪随机序列。

需要另做统计测试，确认大样本分布没有偏差。

## 10. 阶段八：商店、活动和辅助契约

迁移：

- 商店刷新与购买。
- 本地支付派生规则。
- 活动与抽卡开放。
- 巡逻奖励。
- 羁绊和好感。
- 社交、邮件、公会、竞技场空契约。
- GM 命令。

空契约必须准确保留字段形状。不能用统一空对象代替。

## 11. 阶段九：完整集成

### 自动检查

```powershell
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

### 差分检查

- HTTP 登录 JSON。
- 登录推送顺序。
- 关键命令 MessagePack 语义。
- 协议原始字节。
- 存档前后状态。
- 错误回包。
- 未处理命令兜底。

### 实机检查

- 首次登录。
- 已有存档登录。
- 主城。
- 英雄和装备。
- 抽卡。
- 商店。
- 章节。
- 试炼。
- 塔。
- 探索。
- 重启后存档。

## 12. 提交建议

建议拆分为可审核提交：

1. `docs: record Python migration baseline`
2. `build: scaffold Go module and ignore runtime assets`
3. `feat: port protocol primitives`
4. `feat: port cc2p and frame codec`
5. `feat: port config table loader`
6. `feat: port player storage`
7. `feat: add transport and session loop`
8. 后续每个业务域一个或少量提交。

本任务未获提交授权。因此当前不创建提交。

## 13. 预计难点排序

| 等级 | 模块 | 原因 |
|---|---|---|
| 最高 | `Session.dispatch` | 90 个消息和严格推送顺序 |
| 最高 | 奖励与任务 | 动态嵌套和派生规则 |
| 最高 | 探索 | 状态机仍待实机验收 |
| 高 | MessagePack | 字节宽度和映射顺序 |
| 高 | 存档 | 兼容旧字段和并发安全 |
| 高 | 抽卡 | 随机算法和配置权重 |
| 中 | 配置加载 | REF、GZip、子表递归 |
| 中 | HTTP/WS | 手写网络行为 |
| 中 | 客户端构建 | 大文件和外部工具 |

## 14. 开始第二阶段前的条件

1. 安装 Go，并确定版本。
2. 确认新 Go 项目路径。
3. 确认敏感资产只通过外部路径加载。
4. 确认派生规则先照搬，再单独评审。
5. 确认首个试迁移模块为协议底座。
