# Go 已有路径与 Python 对齐记录

日期：2026-10-25

范围由维护者确定为现有 Go 路径，不包括新增战斗、探索交互、抽卡、商店等请求
AGENTS.md 未由本次任务修改，原 Python 发布包只读

## 当前实现

- 已有请求：101、901、1003、1004、1011、1013、1015、1016、1254、1301
- 登录推送：101 → 1001 → 1017 → 1051 → 1201 → 1851 → 1012 → 906 → 903 → 2419 → 1951
- 1015 回包后执行上述从 1051 开始的推送；1301 回包前先推送 1851
- 1254 回包仍为 1254，不改成 1251；每次调用递增 bag_version，不立即写盘
- 默认玩家、旧档合并、队长映射、序章标记、体力恢复和等级上限遵循原函数
- 英雄快照包含养成进度、外观、技能、好感与包含装备加成的队长属性
- 启动时补齐起手装备，背包包含装备实例与普通道具，编队保留普通队伍及 expand_teams
- 系统解锁遵循原版条件；世界、区域、主地图分别按原版规则构建
- 章节含 points、星级、pass_rewards、挂机奖励及领取状态
- 活动与巅峰主动推送按配置与状态生成；不因此宣称对应独立请求已迁移
- 下载奖励、提醒、红点的空结构是原函数真实行为，不用这些结构代替未实现业务
- HTTP 保留 base_version/version 的 Python 真值回退与值类型，默认本地令牌与 Python 一致

业务位于 internal/game，网络会话负责解包、调用业务和按顺序发包
配置加载保留原 JSON 键顺序和子表合并优先级，并对嵌套缓存做深复制
所有依赖版本保持不变，未增加数据库或第三方依赖

## 可重复证据

原始函数来源：
`gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py`

`scripts/python_baseline.py` 通过 AST 提取原函数及实际协议名称映射，避免执行原服务的顶层启动代码
每个案例先运行 load_player 和 ensure_starter_equips，再执行原 Session.dispatch
原存档函数写入临时目录，记录每一步的完整回包、推送顺序、内存状态和实际落盘内容
样本包含原文件 SHA-256、输入配置和原始对象键顺序

测试场景共 6 组、37 个请求步骤：新档、带养成进度旧档、缺字段旧档、重播序章、跳过引导、边界输入
边界包括重复背包请求、空/缺失设置、空改名、无效队长、超限等级和体力、数值/文本角色 ID、非默认头像及未知存档字段
HTTP 另有 8 个样本，覆盖登录、网关、两种 manifest、版本回显和无效 JSON

时间固定为样本 options.now，名字抽样注入选择索引 0
比较业务对象时忽略映射键排列，不忽略数组顺序、字段类型、缺失值、null、空对象与空数组
原有 Varint、MessagePack、CC2P、RSA、帧、REF/XXTEA 的字节向量测试继续保留
官方会话测试使用临时生成的 RSA 测试密钥；网络测试仅在回环临时端口进行

```powershell
# 默认合成配置和合成存档样本，可运行普通 Go 测试
python -B scripts/python_baseline.py
go test ./...

# 发布包存在时，直接读取真实资源进行完整差分
python -B scripts/check_python_diff.py
```

本机 Python 应使用实际安装的 python.exe，PATH 中的 WindowsApps 别名不可用
脚本不修改 Python 源码、缓存、真实存档或客户端
真实表导出与结果位于被现有 .gitignore 忽略的 data/assets/diff/，不得加入 Git
脚本只将资源密钥放入测试子进程环境，不打印或写入样本

主要测试：

| 测试 | 验证内容 |
|---|---|
| TestExistingPathsMatchPython | 37 步回包、状态、启动存档与逐步落盘差分 |
| TestWebSocketMatchesPython | 同样 37 步分别经过社区明文及官方 RSA/CC2P 会话，检查序号、完整消息和存档 |
| TestHTTPMatchesPython | 8 个 HTTP 输出与原函数比较 |
| TestActualTablesMatchPython | 33 张真实配置表的完整内容和顶层键顺序 |
| TestTableLoaderPreservesNestedOrderAndCache | 嵌套对象顺序与缓存隔离 |
| TestRoleUpdatePreservesUnmigratedSaveFields | 未迁移字段及获准保留的未知 null、custom 扩展分组 |

## 保留的兼容边界

维护者已明确选择：旧档中未知 null 字段、custom 下额外分组仅在落盘时保留，不加入业务回包
这与 Python 丢弃这些字段的行为有意不同，单独用存档回归测试验证，不通过差分归一化掩盖

沿用 Go 的显式失败边界：必要资源/密钥缺失、坏存档、保存失败或拒绝覆盖有进度存档时返回错误
Python 某些路径会记录日志后继续成功回包；该失败语义不属于本轮“完全相同”的声明
未迁移命令继续明确报错并结束当前会话，没有虚构协议错误码
配置仍允许 Go 部署端口与监听地址；默认地址下的 HTTP 差分已验证
存档沿用 Go 原子替换、上一版与快照机制，不要求与 Python 备份文件名或格式逐字节相同

原版单玩家模型仍然保留
业务差分与自动化加密连接通过不等于官方客户端实机通过
当前没有新增客户端实机验收证据

后续 [B 服真实配置验证](bili-runtime-validation.md) 已确认：加载 33 张 B 表和现有 Python 差分通过，但 B 原属性函数、装备等级/星级引用、觉醒消费及背包响应编号存在差异
本记录的 Python 对齐结论不应解释为 B 服业务兼容通过；B 服适配以 B 资源与客户端证据为准

## 检查结果

- 合成配置及真实配置的业务、HTTP、双协议 WebSocket 差分通过
- 33 张真实配置表完整比较通过
- 竞态检查尝试执行 `go test -race ./...`，当前 Go 环境禁用 CGO，命令未能运行测试；PATH 中未找到 C 编译器
- 仓库 check.ps1 通过：gofmt、go test ./...、go vet ./...、go build ./...
