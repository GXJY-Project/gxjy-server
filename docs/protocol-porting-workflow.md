# B224 登录到主界面：补齐记录与协作方法

日期：2026-10-27；目标客户端：B 服 1.1.224.1 本地补丁

最新进度：17:37:48 现场确认 1852 断开，现已新增 7 种探索请求，受理数量 43 → 50
场景字段、Python 顺序、差分和当前 323 项静态缺口见 [探索协议与后续计划](explore-protocol.md)
下文 43 请求、30 步和当时 ADB 状态为前一批记录；本轮真实存档副本已扩展为 36 步
本批实机验收：维护者重新登录后回复“已进入主界面”，登录路径已确认打通，剩余业务逐模块验收

## 为什么之前仍然卡住

**已验证**：上一阶段只受理 28 种请求，尚未覆盖 2601/2551 和看板等请求
维护者提供的 23:14:04 日志明确显示 2601 未受理，会话因此结束；另报告了 1006
此前 22:06:57 的客户端还发生过四处业务表引用断言，已单独修复，见 [现场证据](bili224-loading-fix.md)
这两类问题都会阻塞后续流程，不能用「服务器没有 Unknown CMD」作为进入主界面的验收

**已验证**：台服 Python 原型有大量 Go 尚未移植的分支，但也不是 B224 的完整规范
本轮原型没有 2551/2601，且 10129/2016 消费结构已由维护者明确选择 B224
没有把 Python 的默认未知命令 `{result:0,data:{}}` 兜底搬进 Go

**已验证，维护者实机反馈**：最新补丁重新登录后已进入主界面；具体新增入站时序未附日志，仍未知
上一批未连接模拟器；本轮已连接 emulator-5554，游戏进程最近日志未命中所筛选的脚本异常，这不证明资源和异步加载均正常

## 已确认的初始化调用链

原脚本路径：`data/assets/bili/analysis/local-client/base224-local/original-index.js`
SHA-256：`89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6`
下列为 **已验证的静态调用关系**，不代表所有条件在每次登录都满足

| 入口 | 调用/消息 | 依赖与消费位置 |
|---|---|---|
| 101 认证 | 现有 101 → 1001 → 1017 → 1051 → 1201 → 1851 → 1012 → 906 → 903 → 2419 → 1951 推送 | 服务端顺序沿用原型；主动推送不等于同号请求已经实现 |
| 1017 回调 | syncLoginInfo 完成后才发 EVT_MSG_LOGIN_INFO | PlayerModel 381895-381898；中间抛异常会丢失权重 5 的完成事件 |
| syncLoginInfo | 月卡、任务、协战、Rogue 查询 | 382029-382070；U=TaskModel、oe=FightTogetherModel、ae=RoguelikeModel |
| 协战/Rogue 加载 | 2601 / 2551 | FightTogetherModel 261649；RoguelikeModel 404865；实际解锁等级分别 20/17 |
| EVT_MSG_LOGIN_INFO | LoginModel.requestEntranceInfo | LoginController 337122-337132；入口方法 337858-337880 |
| requestEntranceInfo | 公会、塔、背包、聊天、好友/黑名单/联系人、实时/离线/公平竞技场、战斗技能脚本、抽卡池 | 不只请求一个下一 CMD；依各模型 checkEnable 决定是否发送 |
| 1001/1012 解锁变化 | 资源地图、竞技场、Rogue 等补充查询 | PlayerModel 的解锁回调需继续追踪；不能按服务端推送顺序等同客户端请求顺序 |
| 加载完成 | 2016 → 新手引导/剧情/探索或 backHome | LoginController.enterHomeView 337266-337301；即使 2016 正确也有条件分支 |
| 主界面公告 | 1955 | HomeViewController.showActivityAnnounce 327927-327949；空公告必须含 notices |

加载权重仍是登录信息 5、英雄 1、背包 1、自定义设置 1、战斗技能资源 1，共 9
缺少 2016 时应先确认这五类事件；如果已经收到 2016，再查其回调、新手分支、主城预制体和资源异常
不应在这些情况下随意添加一个“加载完成 CMD”或修改解锁条件

## 本轮补齐的范围

**已验证的实现状态**：28 → 43 种受理请求，本轮新增 15 种
1005/1006/2601 详见 [看板及协战记录](board-and-ft-init.md)；2551 详见 [已获确认的初态草案](rogue-init-proposal.md)
2551/2601 是维护者认可的 B224 临时初态，不是原型差分完成的正式系统

以下 11 项执行原 Python 分支生成固定预期，再对比 Go 输出、状态和落盘行为：

| CMD | 名称/请求 | 响应字段 | 证据与范围 |
|---:|---|---|---|
| 1002 | MSG_STAMINA_RECOVERY；透传查询参数 | next_recover_time:0 | Python 9509；B224 382417；只查询，不扣费/恢复体力 |
| 1307 | MSG_GET_CHAPTER_AUTO_INFO；{} | auto_info、rewards:[] | Python 9043；B224 172857；复用 patrol_auto 和已有配置奖励展示，不领取奖励 |
| 1502 | MSG_REQ_MAILS；start/stop/open | mails:{}、max_count:0 | Python 9505；B224 339666/339727/339735；仅无邮件案例 |
| 1752 | MSG_REQ_CHECK_BUY_MONTH_CARD；id/month_card_id/card_id，按该顺序取 truthy 值 | buy:bool、inactive_time:int秒 | Python 9008；B224 416259；读取 month_card[id].until，截止时间相等视为到期；不实际购买 |
| 1955 | MSG_REQ_FACE_NOTICES；{} | notices:[] | Python 9502；B224 49311/327936；仅无公告案例 |
| 2401 | MSG_GET_SUMMIT_BASE_INFO；{} | summitBaseId、stage、actInfo、level/rank/score、heroes/records | Python 9833/8212；B224 376815；复用已有巅峰关闭状态模型，要求 Summit_1 存在；2401 请求与 2419 推送分开 |
| 10102 | MSG_GET_RECOMMEND_FRIENDS；start/stop 等分页参数 | friends:{}、max_count:0 | Python 9700；B224 270713；原型无推荐好友案例 |
| 10152 | MSG_GET_RECOMMEND_CLUBS；{} | version:1、member_info、clubs:{} | Python GUILD_LIST_CMDS 596/10000；B224 280632 |
| 10157 | MSG_GET_CLUB_INVITES；{} | 同上 | 同原型分组；B224 280677 |
| 10167 | MSG_SEARCH_CLUB；keyword | 同上 | 同原型分组；B224 280642；空结果客户端展示无结果提示，不回成功搜索回调 |
| 10175 | MSG_LAST_RECOMMEND_CLUBS_REQ；{} | 同上 | 同原型分组；B224 280658 |

member_info 为 `{invite_num:0,quit_cd:0,new_post_set:0}`，每个响应均有顶层 result/cmd
社交/公会/邮件/公告的相关非空或损坏状态明确失败，不丢弃后返回空列表；没有实现写入、领取、邀请或匹配
1752 对 until 非法整数回退 0 是原型已确认的局部规则，不是忽略存储 I/O 错误
1307 和 1752 按原型可在内存初始化 patrol_auto/month_card 空对象，不主动保存；差分包含此变化

**未知 / 有限兼容**：邮件的 count/unread_count/onekey_receive 等正式字段和非空邮件规则
原型的空响应没有这些字段；B224 现有空列表回调可以执行，但完整邮件界面与后续操作未验收
巅峰关闭模型、全背景列表和竞技场固定本地赛季均保留已记录的原型局限

## 如何发现并补一条 CMD

1. 在 `gxjy-server` 目录运行 `python -B scripts/scan_client_commands.py`
   生成 [完整静态清单](bili224-command-coverage.md)：369 个字面量命令、412 处调用、34 处动态/其他调用待人工追踪
   其中 40 个字面量命令已受理、329 个未受理；Go 总受理 43 个，另 3 个走动态/底层路径
   10 个字面量名称在 B224 表中未找到编号，保持“未知”，不能分配编号
   “未受理”不是“登录必需”，先按调用链和同次日志筛选
2. 编号只取实际 B224 `SERVER_Protocol[name].ProtocolCode`，原型编号表作为对照
   `repair_bili224_save.load_tables` 只读解码并合并授权本地表，密钥不输出，客户端/表数据不复制进文档
3. 用清单里的 `模块.方法:行` 打开原脚本，沿 request 的回调找到 synchronize/updateRecordData/事件
   记录请求字段、响应顶层结构、对象/数组/null 的区别、回调条件、是否继续分页以及配置表引用
4. 搜索 Python 的 `Session.dispatch` 同名分支及其辅助函数，记录状态更新、落盘和推送顺序
   例：1006 在分支里先 send(1001)，函数末尾才 send(1006)；只返回 1006 会漏推送
   例：2601 的名称来自实际表，字段来自 FightTogetherModel；Python 没有分支，不能说是“照搬 Python”
5. 在 `commands.go` 注册编号/名字，`runtime.go` 接入分发，业务放在已有对应模块或 `init_queries.go`
   不给所有未知 CMD 增加 result:0 通配处理；关键错误向上传递，失败时不交付成功消息
6. 用合成配置/存档执行原 Python 函数生成 fixture，再跑 Go 状态及 WS 差分
   新增基线脚本为 `board_baseline.py`、`home_queries_baseline.py`，已有 `python_baseline.py` 通过 AST 提取函数，不启动原服务
   B224 与原型冲突应保留原始预期并记录维护者决定，不在归一化中隐藏差异
7. 用 Node VM 执行散列固定的原客户端消费方法：`node scripts/check_login_init_client.cjs`
   回调验证只能证明该输入不阻断所覆盖的方法；全客户端资源/异步加载仍需实机
8. 更新清单和证据文档，注明已验证/推测/未知及模型边界，再做同次客户端验收

可复制的定位命令：

```powershell
Set-Location 'E:\Go Project\gxjy\gxjy-server'
python -B scripts/scan_client_commands.py
rg -n 'MSG_GET_FT_INFO|requestForLoading|synchronize' data/assets/bili/analysis/local-client/base224-local/original-index.js
rg -n 'MSG_SAVE_BOARD_CHANGE|GUILD_LIST_CMDS' ../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py
go test ./internal/game ./internal/server -count=1
node scripts/check_login_init_client.cjs
```

以上 python/node/go 需已在 PATH；也可使用本机已安装工具的绝对路径
不要修改完整解包脚本或把它添加到 Git；本轮新增脚本只读该文件

## 可以同时进行的工作

| 工作项 | 当前依据 | 建议交付 |
|---|---|---|
| 实机登录验证 | 本轮 43 请求和当前 B224 配置 | 同次首个客户端异常、加载进度、是否出现 2551/2601/2016，以及最后几条服务端 CMD |
| 商店查询 1651/1656/1658/1660/1663/1668 | 原型有分支，清单给出消费方法 | 先核对 B224 字段差异，例如 1658 客户端读取 shop_info，而原型返回 surface_shop，不能直接搬 |
| 英雄查询 1060/1067/1070、收藏 2301 | 原型有分支，但依赖更多配置/进度 | 请求、响应、辅助函数和合成差分样本，尚未在本轮接入 |
| 编队 1202、战斗/探索 1302/1351/1851/1852 等 | 原型实现含真实状态变化 | 单独核对扣费、关卡/场景、单位和奖励；不要为进入主界面先发空成功 |
| 动态/未知编号 | 静态清单末尾及未知编号行 | 追踪调用参数到协议映射，确认版本与编号，不自行命名 |

建议同时工作时认领具体模块文件，避免同时编辑 commands.go/runtime.go；先提交字段证据，再集中接入注册
本轮仅保留工作区补丁，没有提交或推送

## 实机排查与请求查看

服务端最有价值的第一处断点是 `session.go` 的 decode 返回后、dispatch 前；此时 decoded.Command/Message 已解密并解析
响应在 `Runtime.Handle` 返回的有序 Message 列表和 `session.send` 查看
优先用本地调试器检查字段名和类型，不把密钥、认证包、整份玩家回包写入公共日志
游戏帧在 WebSocket 内还有应用层加密，单纯抓 HTTP/HTTPS 或 Wireshark 二进制不能直接得到业务字段

模拟器连通后可只读检查：

```powershell
adb devices
adb -s emulator-5554 shell pidof com.bilibili.gxjy
# 将下面的 12345 换为上一步游戏进程号；只查看客户端错误，日志不要直接提交仓库
adb -s emulator-5554 logcat --pid=12345 -d -v threadtime -t 10000 |
  Select-String 'TypeError|ReferenceError|Assertion failed|ScriptEngine::evalString|Failed to load' -Context 0,8
```

优先保留首次 TypeError/配置断言的字段、资源 ID 和脚本栈，去除认证信息、账号信息、设备标识及 URL 查询参数再分享
不要仅提供后续重复心跳；它只能证明连接仍活着，不能证明英雄、背包或资源加载事件已经完成

## 验证记录

**已验证，实际执行**：

- 补丁前受影响 game/server 测试通过
- Python 看板 3 案例/23 步、主界面查询 2 案例/40 步；Go 响应、内存和存档差分通过
- 明文及官方加密 WebSocket 连续请求通过，包括 1001→1006 顺序及 2551/2601 初态
- 非 B224、非空/损坏进度、配置失败和存储失败不交付成功响应的测试通过
- `go test ./cmd/... ./internal/... -count=1` 通过全部源码包；`go vet ./cmd/... ./internal/...` 通过
- `node scripts/check_login_init_client.cjs`：61 项原脚本消费检查通过；使用合成状态，不启动游戏
- `scan_client_commands.py` 已实际运行；未发现当前 43 种受理请求在已提取字面量调用中的编号冲突

**已验证**：显式启用 `GXJY_B224_VERIFY_LOCAL` 后，`TestBili224LocalRepairedInitialization` 使用真实 B224 表和当前 80 级存档临时副本，连续 30 步查询全部通过，耗时约 135 秒
覆盖认证、原登录请求、2551/2601、1005/1006、本轮 11 查询、心跳及 2016；测试结束检查隔离区保留和原存档字节未变化
这仍是 Go 数据/引用与临时持久化验证，不等同于完整客户端消费和资源加载验收
全仓 `go test ./...` 仍有既有忽略资源目录枚举限制，使用全部源码包范围，不声称该全仓通配命令通过
Go 缓存权限不足的测试经工具授权后重跑；独立构建未运行
**未知 / 未验收**：非空邮件/社交/协战/Rogue 业务、商店、完整战斗及未覆盖的资源异步链路；主界面已由维护者实机确认
