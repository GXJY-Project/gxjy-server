# B224 探索入口、交互及剩余命令计划

日期：2026-10-27；目标：B 服 1.1.224.1 本地补丁

## 结论和当前阻塞

**已验证，维护者提供的现场**：17:37:48 会话完成认证的 11 条响应/推送后，客户端立即发送 1852，服务端因未受理而关闭连接
这是这一次会话已经证明的协议阻塞；不能把它归因为资源加载，也不能据此推定此前所有卡加载只有同一原因

**已验证，静态消费链**：ExploreModel.tryEnterFromLogin 在引导条件成立时请求进入初始探索场景，然后切换探索加载视图
80 级不会自动完成客户端引导标记，所以进入探索分支本身不证明角色等级或主界面资源错误

**已验证，实现与隔离测试**：新增 1851 请求方向、1852、1853、1854、1857、1863、1864，共 7 种请求，Go 受理数量 43 → 50
1856 为主动推送，不计入受理请求数量
**已验证，维护者实机反馈**：在本轮要求重启服务、B224 重新登录并先不点击其他功能的验收中，维护者回复“已进入主界面”
该结论只覆盖本轮客户端及现有存档的登录到主界面，不扩展为探索战斗或剩余业务已兼容

## 证据来源

只读使用本地已有原型和维护者指定的解包资料，不将原脚本、配置表和玩家存档加入样本

| 来源 | 定位 | 状态 |
|---|---|---|
| Python 原型 | `../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py`；SHA-256 `13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25` | 已验证原文件散列 |
| B224 原始脚本 | `data/assets/bili/analysis/local-client/base224-local/original-index.js`；SHA-256 `89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6` | 已验证原文件散列 |
| B224 协议表 | `SERVER_Protocol` 的 ProtocolCode 字段 | 已验证下表编号，未由编号猜测业务 |
| B224 场景配置 | BaseConfig.Initial_ExploreScene、ExploreScene、ExploreObject、ExploreCaseDo/End、ExploreDPTask | 已验证初始场景 Scene_SP_0；Spawn 来自 InitialPos.Transfer.Pos；场景标注物件 99 行，不代表最终回包数 |
| 本轮固化样本 | `internal/game/testdata/explore_python.json` | 已验证只包含合成表与临时存档，由原型执行生成，预期不是由 Go 生成 |

Python 辅助函数：initial_scene_id 844、explore_object_states 1527、active_task_object_cases 1659、_explore_case_fields 1709、ensure_men_door_state 1763、scene_units 1813、_object_xy 1954、case_done 6982、explore_dp_payload 7030
B224：tryEnterFromLogin 235724；synchronizeWorldMap 236156；synchronizeScene 236181；requestEnter 236866；requestPosition 236907；requestQuitScene 237158；requestSceneMini 237190；ExploreScene.synchronize 243571

## 请求、响应与状态变化

下表均为 **已验证的原型行为及本次实现范围**，并不表示完整探索系统或整客户端验收
响应均有顶层 `result:0` 和对应 `cmd`；1856 推送沿用原型只含业务字段
与已有请求一致，可接受 `{msg:请求,seq:序号}` 包装

| CMD / 名称 | 请求 | 响应与顺序 | 状态变化及原型位置 |
|---|---|---|---|
| 1851 MSG_WORLD_MAP | 无必需业务字段 | scene_id、可选 pos、units、teleporte_map:{}、guide_case_do、三个 unlock_*_map | 读存档场景或默认场景；可能触发序章机关重置；9791 |
| 1852 MSG_ENTER_SCENE | scene_id；可选 pos:{x,y}；客户端还发 face/team_dir/is_reconnected/fog_size，原型不消费这些附加项 | **先 1856 后 1852**；scene_id、grid_speed:0.3、units、all_light:true、fog_size:0、fog:[] | 场景单位构造；可能触发机关重置；8900 |
| 1853 MSG_SCENE_MOVE | pos 等客户端移动参数 | 仅 result/cmd，**不回 pos**；客户端将回包无 pos 时的请求位置作为当前位置 | 不保存移动坐标；8946 |
| 1854 MSG_SCENE_MINI | scene_id、fog_size | scene_id（缺失时 null）、all_light:true、fog_size:0、fog:[]、mini_map:{} | 原型固定小地图模型，不构造地图物件列表；9859 |
| 1856 MSG_SCENE_EXPLORE_DP | 无，本项是推送 | explore_dp_map:{区域:{DP任务:完成数}} | 以 cases 中存在的键计数，正 CaseDoNum 为上限；7030 |
| 1857 MSG_SCENE_FINISH_CASE | case_do_id | 首次：1856 → 1857；end_ids、box_reward、guide_case_do、scene_id、units；重复：只有 1857，end_ids/box_reward:[] 且省略 scene_id/units | 首次记事件、同步任务并保存，随后展开/发放 GetReward 奖励；重复不发奖励；9874 |
| 1863 MSG_SCENE_SPEEDUP | 原型不消费业务字段 | 仅 result/cmd | 原型明确 pass，不扣费或改变状态；9498 |
| 1864 MSG_QUIT_SCENE | {} | 仅 result/cmd | 原型明确 pass，不删除场景进度；8957 |

1853/1863/1864 的确认回包有明确原型分支，不是未知 CMD 的默认成功兜底

## 单位构造与初始化依赖

**已验证**：1852 需要有效场景配置及当前队长英雄；单位 `1` 的结构为 `{type:"PLAYER",unit_detail:{model_id:英雄ID,face,可选pos}}`
位置优先请求 pos，再兼容社区表 PlayerPos，最后读取 InitialPos 对应 ExploreObject.Transfer.Pos
没有从 CMD 编号或其他游戏猜测出生点

**已验证**：B224 的 ExploreObject 以 Scene 字段关联场景；遍历所有匹配且非 IsDel 的有效坐标物件
THING 的 unit_detail 有 thing_id、pos，可选 state、going_case_list、case_do_map、case_do_password
状态来自已完成事件的 ChangeSpecialState1；未完成交互合并物件 Cases 与未领取任务的 ExploreCaseDo 绑定
所有交互均已存在于 cases 的物件会被过滤；false 值也算存在，但 guide_case_do 只包含真值

**已验证**：Python 按存档事件插入顺序覆盖同一物件状态，故 Go 读取和保存 cases 时保留该顺序，并测试重启后不改变最终状态
社区表 NormalObject 只追加前 6 项是原型行为；B224 场景标注物件遍历不受这个 6 项限制

依赖关系：队长/英雄配置 → 场景与出生点 → 事件及任务绑定 → 单位快照与探索度 → 1856 推送 → 1852 回调 → 客户端场景构建/资源加载
1857 另依赖 TaskBase、RewardConfig、物品与现有任务奖励实现，消息在整个处理完成后由现有 session 有序发送

## 已保留的原型局限

以下是 **已验证的代码行为**，正式服务是否应如此仍然未知，不据此擅自重写规则：

- 普通 1852 不将新 scene_id/pos 写入存档；1857 使用存档场景或默认场景；只有原型 GM 入口另有 scene_id 赋值
- Scene_SP_0 中两个机关真值且 Boss 未完成时，会撤销两机关及开门标记并保存；world map 的 guide_case_do 在此重置前取快照
- 原型全亮地图、固定 grid_speed，未恢复正式迷雾、速度或次数消耗规则
- active_task_object_cases 遍历 status<2 的任务绑定，没有额外判断任务当前可见性；未臆加触发门槛
- 原型虽然存在 explore_added_objects 辅助函数，scene_units 实际未调用它；没有据注释自行添加动态物件过滤
- 本批没有实现 1855 探索战斗、1351 战后结算、服务端位置/交互权限校验以及全部事件业务

**已验证，错误边界**：配置加载、保存、非法事件引用及关键数值解析错误向上传递，不发送缓冲的成功响应
不存在的场景/事件明确失败；原型可能容忍的无效引用不作为有效协议样本
原型 _object_xy 对坏坐标跳过物件的局部规则已保留并注释
与原型不相同的异常配置边界：SpecialState / CaseDoNum 非法数值在 Go 明确失败，未采用原型跳过/回退
没有凭空定义线上错误码；沿用现有会话错误边界
这不提供跨多个保存动作的事务回滚；保存或发奖途中失败仍可能留有原型式部分状态，不能描述为原子事务

## 还有多少命令、怎么继续

**已验证，静态扫描**：369 个字面量 CMD、412 个调用点、34 个动态/其他 request
Go 受理 50 种，其中静态字面量命中 46 种，字面量清单仍有 323 个未受理
“原来 329 项”是调用清单缺口，不是已经掌握结构的 329 个协议；本轮 7 新请求减少其中 6 项，另一个不在静态字面量调用中

| 缺口类别 | 数量 | 证据状态与下一步 |
|---|---:|---|
| 有 B224 编号及 Python 显式分支 | 57 | 已验证分支位置，尚未全部逐字段移植；按状态变化及客户端读取结构差分 |
| 有 B224 编号，未找到 Python 显式分支 | 256 | 未知完整响应/业务；继续追辅助函数、客户端消费及脱敏样本，不能用原型默认成功兜底当规范 |
| 未在 B224 协议表找到编号 | 10 | 未知编号，先区分客户端废弃路径/动态映射/版本差异，禁止猜编号 |

明细及每条客户端 模块.方法:行、Python 分支行见 [自动扫描清单](bili224-command-coverage.md)
静态未找到分支并不能证明不存在动态处理；34 处动态 request 也未被以上数字覆盖

**推测，实现优先级**：

1. 登录到主界面已获维护者实机确认，保留这批补丁作为基线；后续按功能逐项验收，避免同时更换客户端或存档
2. 有证据的探索后续链：1855 探索战斗 ↔ 1351 战后结算 ↔ 1555 任务奖励应成组实现，涵盖入场消耗、team_info/entry_skills、随机种子、重复结算及保存失败；不能只补 battle_info 中一个 ID
3. 独立查询模块：1060 档案、1067/1070 收集、1651/1656/1658/1663/1668 商店；需要检查目标版本配置引用与查询引起的刷新/落盘
4. 英雄/装备/队伍/物品写入及奖励分支按模块差分，支付原型有本地点击即发放行为，移植前记录重复发放风险与范围
5. 256 项缺少明确分支及 10 项未知编号另列证据任务；取得服务端样本或新的合法参考后再落实实现

这份优先级不代表这些未实现系统已经返回成功；未知请求继续明确暴露缺口

## 如何配合最有效

**建议**：每次只改变一批服务端补丁，固定 B224 补丁版本与存档；更新后重启服务并重新登录
结果反馈同时写“做了什么、屏幕停在哪里、服务端首个失败 CMD、客户端第一条异常”，比只贴 901 更能定位

测试本批：保持原配置及 80 级存档，重启 Go 服务，B224 重新登录后先不点其他功能
期望看到 `收到1852 → 发1856 → 发1852`；记录最终主界面、探索场景或卡住状态
若继续失败，取本次从 101 到首个失败的脱敏服务端日志，以及同时间窗口的客户端异常/资源失败
只读 ADB 方法及已解密字段查看位置见 [协作和排查方法](protocol-porting-workflow.md)

并行工作按模块认领：一方负责 `explore.go`，另一方先记录 shop 或 hero 的请求、读取字段、状态变化和 Python 定位
避免同时改 `commands.go`、`runtime.go`、运行配置和当前存档；在独立文件准备实现/证据后统一接入分发
可以提供台服 Python 能进主界面时的启动参数、引导设置和脱敏 CMD 时序作对照；不需要提供账号密钥或完整资源包

## 实际验证

**已验证**：

- `python -B scripts/explore_baseline.py`：执行固定散列原型，4 案例/21 步，合成场景、奖励、任务绑定、false case、重复交互及机关重置
- `go test ./internal/game ./internal/server -run Explore -count=1`：Python 响应/内存/存档差分、明文和加密 WebSocket、关键错误和重启顺序均通过
- `node scripts/check_explore_client.cjs`：10 条原始 B224 响应消费检查通过，依赖替身仅隔离外围；不启动完整客户端或验证资源异步加载
- `GXJY_B224_VERIFY_LOCAL` 显式指向原配置后，`TestBili224LocalRepairedInitialization` 使用真实 B224 表及当前存档临时副本，36 步通过，约 150 秒；验证原存档字节未变、隔离区保留、进场推送顺序及玩家/物件存在
- `go test ./cmd/... ./internal/... -count=1`：全部源码包通过；忽略资源目录存在枚举限制，未声称全仓 `go test ./...` 通过
- `go vet ./cmd/... ./internal/...`：通过；修改的 Go 文件 gofmt 检查无输出
- `node scripts/check_login_init_client.cjs`：已有 61 项原始客户端方法消费检查仍通过
- 维护者完成本批实机重新登录，反馈“已进入主界面”；没有附带新 CMD 明细或客户端栈，不据此补造时序

**未知/未执行**：1855/战后与全部剩余 CMD 的实机验收；Agent 本轮未单独构建启动服务或部署客户端
改动保留在工作区，未修改当前存档、运行配置或第三方原文件，未提交/推送
