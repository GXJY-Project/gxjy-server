# B224 登录初始化查询实现记录

日期：2026-10-26（Asia/Shanghai）

2026-10-27 更新：本文的 28 请求及「仍未接入」表是第一批实现记录
当前已受理 50 种，新增探索入口等 7 请求见 [探索协议移植](explore-protocol.md)；2601/2551 正式系统仍未实现
维护者已反馈本批重新登录“已进入主界面”；以下历史缺口与未知项需结合最新记录阅读
最新实现与操作方法见 [主界面查询补齐](protocol-porting-workflow.md)

本文保留成组补齐查询时的实现记录；后续已根据实机四处配置断言，获授权调整 B224 配置并修复现有存档
最新变更、备份与验证见 [B224 加载阻塞修复](bili224-loading-fix.md)，本文关于“没有修改配置/存档”的表述仅指前一阶段

## 实施依据与授权

**已验证**：维护者在分析报告交付后要求成组补齐初始化 CMD，并明确选择 B 服 224 的消费结构
沿用现有 `CommandName → Runtime.Handle → []Message → session.send`，没有新增网络框架、数据库或依赖，没有修改运行配置、客户端、已有存档和初始化解锁规则

协议来源和散列见 [分析报告](login-init-analysis.md) 的 P1、C224、T224；其原有 13 种受理统计是本补丁前的基线

## 已接入的查询

以下为 **已验证的实现状态**，新增 15 种受理请求，总数从 13 增至 28
“已接入”限定为下述查询契约，不表示对应完整游戏系统已经实现

| CMD | 响应契约 | 来源与限制 |
|---|---|---|
| 10101 MSG_GET_FRIENDS | friends:{}、max_count:0 | P1:10097；C224:270302/270697；无好友案例，不继续分页 |
| 10124 MSG_REQ_BLACKLIST | blacklist:[] | P1:8970；C224:270851；空黑名单案例 |
| 10129 MSG_GET_CHAT_LIST | contactList:{} | C224:270323/270729；按维护者决定替代原型 chat_list，空联系人案例 |
| 2016 MSG_GET_PVP_ROOM | room_info:null | C224:386084；按维护者决定使用明确无房间值，false 请求回调可继续登录 |
| 10151 MSG_GET_CLUB | version:1、member_info:{invite_num:0,quit_cd:0,new_post_set:0}、clubs:{} | P1:10000；C224:288852；无公会案例，不能只有 result |
| 2101 MSG_REQ_TOWER_INFO | points:{}、cum_rewards:{}、end_time:0 | P1:9844；C224:456986；无塔进度案例，已有塔进度明确失败 |
| 1151 MSG_REQ_RESOURCE_MAP | version:1、resource_maps、auto_info | P1:5855/5876/9512；C224:395502/395619；按 ResourcesMap/ResourcesPoint 构造，读取 resource_pass |
| 1601 MSG_DRAWCARD_SYSTEM | total、ssr_probable、card_pool | P1:4712/7541；C224:208898/209045；读取 gacha.total/pity/pool_times，卡池 ID 来自实际 DrawCard 表 |
| 1701 MSG_REAL_ARENA_BASIC | arena.season、arena.arena_info | P1:10106；C224:116271/116505；下述本地原型赛季模型 |
| 1702 MSG_REAL_ARENA_INFO | 与 1701 一致的 arena_info，包含 rank/score/rewards | P1 的 1701 状态与 C224:116532 消费点组合；新增的 B224 查询适配，不声称原型已有 1702；仅登录 type=0 |
| 1810 MSG_ARENA_BASIC | arena.season/rank/score/opponents | P1:10115；C224:110661；下述本地原型赛季模型 |
| 1802 MSG_ARENA_INFO | 与 1810 相同的原型查询结构 | P1:10115；C224:110661；不含防守阵容编辑、对手生成或战斗 |
| 2351 MSG_FAIR_ARENA_INFO | arena.season_config_id/status/win_time/max_win_num/lose_time/reset_time/rewards/heroes | P1:10123；C224:254611；下述本地原型赛季模型 |
| 1951 MSG_REQ_ACTIVITIES | 复用现有 activities() 快照 | P1:9775；C224:49201；此前只有登录推送，此次补请求方向；未扩展活动业务或过滤规则 |
| 905 MSG_CLIENT_TRACKING | 已有 Base 确认响应 | P1:9871 明确 pass 后确认；C224:431343 只读 result；不持久化统计，仅此已确认命令使用 ACK，未知命令仍报错 |

所有业务字段位于响应顶层，统一保留 `result` 和 `cmd`，没有引入统一 data 空壳
保持现有登录推送顺序和背包 1254 同号响应

请求字段沿用分析报告：10101 的 start/stop/type、10124 的 start/stop、10129 的 chat_id、2016 的 show_error/version、10151 和 1151 的 version、2351 的 req_pool、905 的 track_type/track_info
1702 接受登录查询 type=0；1601、1701、1810、2101 使用空对象查询
1951 沿用原型返回活动快照的行为，没有为 activity_id 新增过滤语义；以上空状态/固定模型不额外分页、发放物品或改变社交关系，1151 的巡逻数据仍使用既有内存初始化行为

Go 位置：`internal/game/commands.go` 注册编号，`runtime.go` 进入 `initializationQuery`，`init_queries.go` 构造相应数据，`internal/server/session.go` 顺序回包
业务和边界验证见 `internal/game/init_queries_test.go`，网络验证见 `internal/server/init_queries_test.go`；固定预期来自 `scripts/login_init_baseline.py` 及其合成 fixture，客户端消费验证见 `scripts/check_login_init_client.cjs`

## 模型边界与错误行为

**已验证**：社交、公会、无房间和塔采用原型已存在的空状态案例
如存档含相关非空或损坏状态，返回明确错误，不将其静默清空后返回成功；这些查询不写玩家存档
非空好友列表、联系人历史、公会和房间恢复的完整结构仍为 **未知 / 未实现**

**已验证**：竞技场沿用已确认 Python 原型的本地查询模型：

- 实时赛季 `ArenaRealTime_1`、离线赛季 `Arena_Season1`、公平赛季 `ArenaFairPlay_Week1`
- 原型赛季 ID、结束时间、开放状态、rank=1、score=1000，以及公平竞技场的初始空奖励和英雄集合均保留原型取值
- 查询前必须成功读取对应配置行；缺表、加载失败、缺行直接返回错误，不替换成别的赛季、不发成功壳
- 已有竞技场非空状态不由该本地模型解释，明确报错；1702 非 type=0 请求也报错

这组字段是 **已验证的原型行为**，正式 B224 的赛季时间、排名、匹配和奖励规则仍为 **未知**
本补丁没有把固定赛季模型包装成完整竞技场服务，也没有实现匹配或扣费

**已验证**：1151 从实际资源地图和关卡表生成数据，关键关卡引用不存在或存档记录损坏时失败
挑战次数按原型配置优先级查找，没有配置时保留原型默认 2；这是兼容模型，不是正式每日次数证明
不会为查询增加奖励或战斗记录；auto_info 复用既有巡逻数据构造

**已验证**：1601 只读取卡池、累计抽数与保底计数，0.02 概率和 80 保底阈值来自原型常量
保留 normal 类型优先、同组按 ID 排序的卡池顺序，并通过局部有序对象编码使其到达 MessagePack；其他 map 的编码方式不变
空卡池配置、损坏计数、非法数组元素和溢出返回错误；没有实现抽卡、扣费、兑换或奖励分配
正式 B224 卡池开放策略和概率仍为 **未知**

**已验证**：会话发现 `UsedTextFallback` 立即返回 MessagePack 解码错误
底层协议解析器的历史文本回退 API 保留，但游戏会话不再将该结果当成空请求进入业务层
明文及加密异常包均在分发前失败，不返回成功响应；现有未受理 CMD 的致命错误行为保留

## 仍未接入的请求

下列项没有随意用成功包填补：

| CMD | 状态 | 尚缺内容 |
|---|---|---|
| 2551 MSG_REQUEST_ROGUE_INFO | 后续已接入获确认的 B224 未开局初态 | 正式 Rogue 难度、节点、英雄、图鉴和重置规则仍未知；见最新记录 |
| 2601 MSG_GET_FT_INFO | 后续已接入获确认的 B224 初态 | 正式协战 Boss、协作者、额度与重置规则仍未知；见最新记录 |
| 108 MSG_TYPE_RESUME_RESPONSE | 已验证未受理；重放契约未知 | 实际历史消息窗口、补发及重连语义，不能只 ACK |
| 1851 的请求方向 | 已验证未受理；地图推送保留 | 原型请求包含场景、位置、units，不能将已有解锁推送冒充完整场景初始化 |

历史出现的 1014、1854、1955 等其他业务请求，也不在此次已接入的 15 项之内
已有英雄、装备和解锁配置与 B224 的差异继续按分析报告处理，此次没有为掩盖问题改动存档、资源或解锁条件

## 验证记录

**已验证，实际执行**：

- 修改前 `go test ./internal/game ./internal/server -count=1` 通过
- `python -B scripts/login_init_baseline.py` 生成 2 个合成案例、42 步登录及查询样本；原型散列固定，不执行原服务入口，不读取真实玩家存档
- 仅 10129、2016 根据已确认 B224 契约调整预期，同时保留原始 `python_messages`，不将差异隐藏成原型完全一致
- `go test ./internal/game ./internal/server -count=1` 通过：业务状态/存档差分、明文与官方加密 WS 连续查询、1701→1702 查询、异常包无成功响应、错误传播、无效引用和卡池 MessagePack 顺序
- `node scripts/check_login_init_client.cjs` 通过：在散列固定的 B224 原脚本上隔离执行好友、公会、无房间回调及实时竞技场原方法，2 个案例共 14 次消费检查
- `TestInitializationLocalAssets` 通过：使用默认配置和 B224 原底包两套实际本地表，各执行 15 种新增查询；只读资源密钥和配置，使用新内存角色，不读取 save_dir，不生成资源副本或业务存档。耗时约 167 秒
- `go test ./cmd/... ./internal/... -count=1` 通过，覆盖全部项目源码包；`go vet ./cmd/... ./internal/...` 通过
- 修改的 Go 文件已执行 gofmt；全仓 `gofmt -l .` 仍列出既有 `internal/game/integer_test.go`、`internal/game/values.go`，本补丁没有顺手格式化这些文件

**验证限制**：`go test ./... -count=1` 在枚举旧忽略资源目录 `data/assets/bili/analysis/runtime-validation/run-bakqdz_h` 时被文件系统拒绝访问，因此使用上面的全部源码包范围完成测试，不声称全仓通配命令通过
首次受影响测试还遇到本机 go-build 缓存访问拒绝，经工具权限批准重跑成功
独立 `go build` 的权限请求被拒绝，命令未执行；没有改用其他路径规避拒绝。源码包测试已经编译相应包，但不将其描述为独立构建通过

**未知 / 未验收**：真实客户端进入主城、全部非空业务状态、活动和赛季真实性、场景加载、重连补发，以及配置版本差异对本次角色的具体影响

下一次实机验证应使用本次源码的重新启动程序，记录从 101 到首次失败的同一次日志及客户端首条异常
无须反复测试旧日志中的 10115；本补丁默认留在工作区，不提交、不推送、不修改现有运行配置
