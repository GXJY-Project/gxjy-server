# B224 看板和协战初始化

日期：2026-10-27；目标：B 服 1.1.224.1 本地补丁

## 证据与状态

**已验证**：1005/1006 来自原型协议表及 `Session.dispatch` 9406-9497
原型 SHA-256：`13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25`
**已验证**：B224 `PlayerModel.requestBoardInfo` 382331-382345、`requestSaveBoardChange` 382312-382330 消费相同字段
B224 原脚本 SHA-256：`89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6`

**已验证**：2601 在实际 B224 SERVER_Protocol 中命名为 MSG_GET_FT_INFO，旧 Python 没有分支
`FightTogetherModel.requestForLoading` 261649-261653 → `requestGetFTInfo` 261411-261419 → `synchronize` 261242-261258
实际 UnlockSystem.FightTogether_Unlock 要求 Level >= 20；80 级角色会触发这个条件
维护者明确确认「先支持初始空态，继续打通登录」，此实现限定为临时登录适配
正式额度、活动、Boss、重置和奖励规则仍为 **未知**

## 请求与响应

响应字段均在顶层，包含 `result:0` 和对应 `cmd` 名称

| CMD | 请求 | 响应 | 状态与变化 |
|---:|---|---|---|
| 1005 | {} | board:{wife,back_now,mode}，board_info:{back_all,back_new:{}} | 已验证原型差分；back_all 收集 BackGroundHome 的非元数据对象行，值为 1；不保存 |
| 1006 | hero_id?、back_id?、mode? | 先 1001 精简角色推送，再同号 board 和 board_info:{back_all:{},back_new:{}} | 已验证原型差分和发送顺序；保存外观 |
| 2601 | {} | playerFTInfo、bossList:[]、actBossIds:[] | 已验证客户端初态消费；不修改或保存玩家状态；不是正式协战实现 |

1006 按原型仅保存 truthy 的 back_id、HeroBase 中 IsShow==1 的 hero_id，以及不为 null 的 mode
mode=0/false 可以清零，缺失/null 保留原值
原型不检查英雄/背景所有权，并在 1005 列出全部背景；本次保留这些已知原型局限，没有扩展为正式发放系统
先保存后生成 1001；精简 exterior 包含 wife/back_now/mode/leader/system_opens，system_opens 空时沿用原型完整 exterior 回退
配置读取失败、保存失败或生成角色失败直接返回错误，不向网络层交付成功消息

2601 的 playerFTInfo 明确为：

```json
{"maxCoin":0,"gainCoin":0,"maxDifficultyId":"","assistNum":0,"resetTime":{},"collaborator":[]}
```

这些值来自 B224 构造器 261209-261222，不是正式服额度或重置时间的证明
仅 bili-224 可用；fight_together/playerFTInfo/bossList/actBossIds 任一非空或类型损坏时明确失败
已有协战进度的存档识别字段仍可能不完整，新增持久化模型前须重新核对，不将此临时响应作为迁移工具

## 验证

**已验证，实际执行**：

- `python -B scripts/board_baseline.py`：3 个合成案例、23 步，执行散列固定的原 Python 函数
- Go 看板响应、推送顺序、内存状态和持久化结果差分通过；包含零/false/null、无效英雄和保存失败
- 明文及官方加密 WS 测试通过；2601 仅 B224、只读和非空状态拒绝测试通过
- `node scripts/check_login_init_client.cjs` 在 B224 原方法上检查看板回调、协战初态及其他查询；当前共 61 项通过

2551 的独立初态适配、其他新增请求及总体状态见 [主界面查询补齐与协作方法](protocol-porting-workflow.md)
**未知 / 未验收**：本次补丁在真实客户端进入主界面的结果；离线回调通过不能代替实机验证
