# B 服真实配置与 Go 业务验证

日期：2026-10-25

**结论：Go 可以实际加载 B 服全部 33 张业务表，但现有角色、英雄、装备及背包路径尚未通过 B 服兼容验收**

本轮以 B 服更新资源和客户端原函数为准
Python 原型只提供差分基线；它与 Go 一致不代表它们符合 B 服
没有修改正式配置、正常存档、Python 原文件或客户端，没有连接正式服务
本次修改是独立验证工具、测试和报告，尚未修改业务实现或选择新的起手装备策略

## 来源与隔离

| 证据 | 标识 |
|---|---|
| 授权资源快照 | `com.bilibili.gxjy`，`data/assets/bili/resolved/` |
| 主资源版本 | `project.manifest` 的 `1.1.281.1.1` |
| 更新客户端 JavaScript | `data/assets/bili/analysis/verified/index.js` |
| JavaScript SHA-256 | `a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0` |
| 原 Python 业务 | `gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py` |
| 原 Python SHA-256 | `13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25` |
| 原 Python 表读取 SHA-256 | `9e0f0777b1b5b37fdf52088c1d06ea1018cbd93836488bf577452230d8ed2cf9` |
| 原解密模块 SHA-256 | `5afd80bae5169ef97cfe05cadb8375911c500b53a4cc4d66e5b5507cbd0da0be` |

来源、授权及资源边界沿用 [B 服资源基线](bili-resource-baseline.md) 和 [B 服脚本分析](bili-script-analysis.md)
`scripts/check_bili_runtime.py` 固定原型版本，显式指定此次 B 服资源目录，解密缓存只写入临时目录
不启用指向其他历史资源包的 `GXJY_BUILD=bili`

每次验证在已忽略的 `data/assets/bili/analysis/runtime-validation/run-*/` 新建独立 `config.json`
配置中的相对 `save_dir=go-saves` 由真实 `config.LoadRuntime` 按配置目录解析
Go 使用 `assets.TableLoader → game.New → storage.JSONStore`，每个案例又写入新的独立存档目录
资源密钥仅进入测试子进程的 `GXJY_BILI_TEST_ASSET_KEY`，不写进配置、报告或日志

## 实际表加载

状态：已验证

33 张表共 65,701 行，读取过程涉及 Bundle 与分表、引用表共 434 个资源文件
逐项核对主清单大小与 MD5，并记录 SHA-256
Go 与 Python 对所有合并后的表内容及顶层键顺序进行完整比较，不能用行数相同代替内容比较
验证器还检查 `_subFiles` 的读取错误，避免原型的跳过错误行为掩盖缺表

| 表 | 行数 | 表 | 行数 |
|---|---:|---|---:|
| HeroBase | 117 | HeroData | 385 |
| HeroQuality | 742 | HeroStar | 810 |
| HeroAwaken | 21 | HeroFavor | 1,001 |
| SkillBase | 8,533 | SkillSpecial | 1,362 |
| LeaderBase | 9 | LeaderTeamSkill | 16 |
| HeroCostAttr | 18 | BaseConfig | 956 |
| EquipBase | 58 | EquipExp | 200 |
| EquipQuality | 464 | EquipStar | 464 |
| SkillAttr | 3,095 | PlayerLevel | 80 |
| UnlockSystem | 188 | ExploreWorldMap | 12 |
| ExploreWorldRegion | 17 | ExploreWorldMain | 4 |
| ResourcesPoint | 128 | ResourcesMap | 6 |
| FightMap | 18 | FightPoint | 1,198 |
| RewardConfig | 44,305 | DrawCard | 72 |
| Activity | 1,010 | SummitBase | 1 |
| NameFront | 204 | NameMiddle | 1 |
| NameLast | 206 | | |

B 客户端 `JsonReader._createJsonObj` 使用后行覆盖，Python 与 Go 的当前合并规则保留前行
此次 33 表的实际主表、分表、引用表没有同名行冲突，两种规则产生相同结果
因此本快照通过，不据此宣称其他配置表或未来更新的合并规则已兼容

## 合成存档与字段消费

9 组存档、46 个请求步骤：新档、养成旧档、缺字段旧档、重播序章、跳过引导、边界输入、全部可见英雄、全部可见英雄的养成状态、全部真实装备
覆盖 77 名可见英雄；其中 7 名具备 B 服觉醒阶段配置，另有 58 种真实装备模板
时间固定为 `1800000000`，名字抽样固定为索引 0

逐步比较启动内存、启动落盘、完整回包及顺序、每一步内存和磁盘、再次加载后的内存与磁盘
保留数组顺序、字段类型、缺失值、null、空对象和空数组的差别
重复背包请求递增版本但不立刻落盘的行为也参加比较
章节、队伍、体力、系统解锁、活动与巅峰推送参加现有 Python 差分；未将这些结果提升为完整 B 服规则

`scripts/check_bili_runtime_client.cjs` 核对原脚本哈希后，只执行选定模块 factory
它运行 B 服原 `HeroAttribute.calculationBaseAtt`、`calculationEquipAtt`、`Equip` 读取方法与 `BagModel.initPush`
产生 231 组英雄属性和 290 组装备属性、等级与星级引用预期
随后用原 `Hero.synchronize` 与 `Equip.synchronize` 消费 Go 的合成输出
Hero 的技能模块和事件发送使用隔离替身，只对基础字段与觉醒消费作出结论，不执行游戏入口或网络

## B 服验收发现

### 起手装备与等级、星级

结论：当前代码会在 B 服新档中生成不存在的装备，真实装备的默认等级和星级编号也不符合 B 表
状态：已验证配置引用和原客户端读取结果

来源：`internal/game/equipment.go` 的 `starterIDs`、`ensureStarterEquips`、`equipData`；B 表 `EquipBase`、`EquipExp`、`EquipStar` 与原 `Equip.getLevel/getStarId/getStarNum`

- 当前优先寻找 `Equip_S3_1/2/3`，再寻找 `HeroType=["All"]`，最后回退 `Equip_S6_1`
- B 服 58 个模板使用 `Equip_N_Weapon_*`、`Equip_S_Clothes_*` 等命名，没有旧模板，也没有 `HeroType=["All"]` 的行
- 新档仍生成 `Equip_S6_1`，导致 `EquipBase`、`EquipQuality`、`EquipStar` 引用无效；原 `Equip.synchronize` 会在查询模板时失败
- B 表 `BaseConfig/Equip_Exp.Config` 按稀有度给出 `N → Equip_N_1`、`S → Equip_S_1`，然后沿 `NextId` 升级；Go 却从装备模板 ID 开始找 `EquipExp`
- 58 种模板的 290 个等级/品质输入全部出现 `level_id` 差异，例如 `Equip_N_Weapon_1` 应指向 `Equip_N_1`
- B 的初始星级应按 `InitStar` 在 `StarEffect` 链中查找；N 模板初始 1 星，S 模板初始 4 星，不能把链起点固定当作初始星级
- Go 缺少星级进度时构造 `_0`，58 个默认星级输入均与 B 原 `getStarId` 不同

290 组装备基础属性计算与 B 原 `calculationEquipAtt` 的正值输出相同
这只证明该快照当前基础数值字段可计算，不覆盖装备被动技能、职业适用性、附魔和服务端发放策略
没有为避免失败而给所有角色随意分配其他模板

### 英雄品质与星级字段

结论：Go 与 Python 都累计 0 到当前品质的基础和成长，而 B 原函数只读取当前品质行
状态：已验证原属性函数差异

来源：`internal/game/heroes.go:heroAttr`；B `HeroAttribute.calculationBaseAtt`

231 个输入中，品质 0 的 77 组一致，品质 1/3 的 154 组共出现 616 个四维属性差异
示例：H004，等级 7、品质 1、星级输入 1，Go 的 Atk 为 237、Hp 为 2344，B 原函数分别为 172、1700
差异不能通过更改预期值或归一化隐藏

另一个可观察问题：Go 回包的 `star_level=保存进度+1`，同时 `star_id` 指向保存进度对应的行
B `Hero.synchronize` 直接存储 `star_level`，`getStar` 直接返回它；属性函数又用该数值构造 `HeroStar` 行名
本轮原客户端消费的 170 个英雄记录中，这个数值均与所指行的 `Star` 不同
星级编码需结合 B 服服务端合法消息样本确认，不能继续仅因与 Python 相同就认为正确

### 觉醒

结论：B 的 `HeroBase.Awaken` 是阶段 ID 数组，现有旧觉醒字段不能表达 B 的阶段与节点进度
状态：配置类型与原字段消费已验证；完整养成协议待验证

来源：B `Hero.getAwaken`、`Hero.synchronize`、`Hero.getAwakeIsOpen`；`HeroAwaken` 的 `HeroAwakenPoint` 引用

B 所有 117 个 HeroBase 行的 Awaken 均为数组，77 个可见英雄中 7 个有非空阶段列表
原客户端从 `awakenInfo.open/stages[].awakenBaseId/points[].pointId/level` 读取状态，不读取旧 `base_attr.awaken_id`
这 7 个英雄设置旧存档 `awaken=1` 后，Go 与 Python 都没有输出 B 的 `awakenInfo`，原客户端仍判断觉醒未开启
当前 33 表之外的 `HeroAwakenPoint` 及对应业务状态还需要专项验证
第一轮验证器把 Awaken 当作单个 ID 的检查已纠正，未将该验证器假设当成业务缺陷

### 背包响应路由

结论：Go 目前以请求编号回复背包，B 客户端注册的是另一个响应编号
状态：B 编号表、原注册函数与 Go 合成输出已验证

来源：B 合并协议表的 `MSG_REQ_BAG=1254`、`MSG_BAG_RES=1251`；`BagModel.initPush`；`internal/game/runtime.go:Handle`

原 `BagModel.initPush` 注册 `MSG_BAG_RES` 并交给 `synchronize`
本轮 11 次背包请求都得到 Go 的 1254，符合旧 Python，但不符合 B 注册的 1251
即使修正装备字段，仍需修正路由并单独核实版本、分页与增量更新语义

### 角色初始配置

结论：当前默认角色不使用 B 的若干初始配置
状态：表值与 Go 新档输出差异已验证；B 服务端实际新号策略待验证

| 项目 | 当前 Go 新档 | B 配置 |
|---|---|---|
| 默认持有英雄 | H004、H002 | `Player_Init_Hero`: H007、H016、H015 |
| 头像 | PHM_H001 | `Player_Init_Head`: PHM_H015 |
| wife | H002 | `Player_Init_Wife`: H015 |
| 初始资源 | 空映射 | `Player_Init_Resource`: IC_Diamond=100，IC_Gold=10000 |

`Player_Init_HeroModel=HBoy` 是模型配置，不能未经证据把它当作 `LeaderBase` 主键
这 4 项不能只凭表名推断完整登录、创建角色、性别选择和发放时机，需要 B 合法响应或相应服务端证据

## 复现与结果文件

在 `gxjy-server` 根目录运行，Python 必须是可执行的本地解释器
需要已授权资源、对应本地资源密钥、已解码脚本以及先前核实的 B 合并编号表

```powershell
python -B scripts/check_bili_runtime.py `
  --go 'C:\Program Files\Go\bin\go.exe' `
  --node 'C:\Program Files\nodejs\node.exe'
```

本轮实际使用缓存运行时的 Python：
`C:\Users\WANGYU\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe`

输出目录包括：

- `config.json`：独立配置，没有资源密钥值、RSA 文件或正式服务地址
- `resource-evidence.json`：434 个资源的完整性标识、表行数、字段类型及合并冲突检查
- `python-fixture.json`：真实 B 配置与固定合成行为样本，始终保留在忽略目录
- `client-expectations.json`：原 B 函数产生的属性和引用预期
- `go-runtime-report.json`：每例差分、重载结果、无效引用、路由与属性差异
- `client-consume-report.json`：原客户端消费 Go 输出的结果及初始化差异
- `protected-files.json`：正常配置和存档的前后完整性检查
- `go-test.log` 和 `go-saves/`：本次真实测试日志及合成存档

成功生成报告不等于兼容通过
只要表内容、状态差分、B 引用、属性或客户端消费存在差异，验证脚本就以非零状态退出
默认 `go test ./...` 不读取这些本地授权材料，选择性测试须由本脚本显式启用

## 本轮检查状态

- 全部 33 表内容与顺序比较通过
- 9 组、46 步启动/业务/落盘差分和 9 次重新加载通过
- B 原函数校验及客户端消费发现上述差异，B 兼容验收失败
- `go-runtime-report.json` 记录 742 个无效引用、616 个英雄属性差异、290 个装备等级引用差异、58 个默认装备星级差异及 11 次背包响应编号差异
- 原客户端消费检查覆盖 170 个英雄记录、229 个装备实例，记录 571 次配置读取失败、7 个未被识别的旧格式觉醒进度及 170 个星级字段/配置行差异
- 正常配置和存档共 3 个文件的前后 SHA-256 检查通过，文件集合也未变化
- 仓库 `scripts/check.ps1` 通过：gofmt、`go test ./...`、`go vet ./...`、`go build ./...`；此处默认测试不启用本地 B 服专项验收

最终完整运行目录：`data/assets/bili/analysis/runtime-validation/run-bakqdz_h/`
专项 Go 测试和原客户端消费检查都返回 1，验证脚本也返回 1，原因是上述 B 服兼容差异
早期运行发现的验证器类型假设和 Windows 日志编码问题已修正；最终保存的 `go-test.log` 是有效 UTF-8

没有进行 B 原客户端实机登录、正式账号认证、真实私钥握手、完整觉醒/技能/背包分页业务验证
RSA 不匹配仍是独立的实机接入限制
本轮没有改动并发实现；未重复执行此前因 CGO/C 编译器缺失而无法运行的 race 检查
本任务完成的是可复现验证和差异定位，不能将其状态记作“B 服业务适配完成”
