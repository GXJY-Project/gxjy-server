# 游戏命令迁移清单

本清单来自 Python `Session.dispatch` 静态提取

它只证明 Python 有分支，不证明行为正确

## 状态说明

| 状态 | 含义 |
|---|---|
| 未分析 | 尚未逐字段核对 Python 行为 |
| 已分析 | 已记录输入、输出、推送、状态和配置 |
| 代码已迁移 | Go 已有完整实现，尚未完成差分 |
| 差分通过 | 固定输入下与 Python 结果一致 |
| 查询验证通过 | 仅文档限定的初始化查询案例通过原型差分或 B224 消费验证，不代表完整系统迁移或实机通过 |
| 实机通过 | 官方客户端已完成目标流程 |
| 受阻 | 缺少需求、资源、密钥或实机条件 |

Go 当前的空对象、空数组和固定成功回包都不算迁移
2026-10-26 的初始化查询实现单独标记范围，详见 [初始化查询实现记录](login-init-implementation.md)，不得将合法空态查询误读为整个好友、公会或竞技场业务已完成
2026-10-27 当前受理 50 种请求；探索入口及交互见 [探索协议移植](explore-protocol.md)，此前查询及 B224 初态见 [补齐记录](protocol-porting-workflow.md)，静态缺口见 [扫描清单](bili224-command-coverage.md)

## 命令明细格式

每个命令进入开发前，必须补齐以下内容：

```text
编号：
名称：
请求：
回包：
推送顺序：
状态变化：
配置表：
Python 位置：
Go 位置：
差分证据：
实机证据：
状态：
待确认问题：
```

## 总表

| 编号 | 命令 | Python | Go 当前情况 | 状态 |
|---:|---|---|---|---|
| 101 | MSG_TYPE_AUTH | 有分支 | 按原版生成角色、英雄、编队等完整登录推送 | 差分通过 |
| 901 | MSG_HEARTBEAT | 有分支 | 固定时钟及 clientTimestamp 回显差分通过 | 差分通过 |
| 905 | MSG_CLIENT_TRACKING | 有分支 | 原型明确的确认接收响应，不持久化统计 | 查询验证通过 |
| 1002 | MSG_STAMINA_RECOVERY | 有分支 | 原型 next_recover_time 查询，不执行体力恢复 | 查询验证通过 |
| 1003 | MSG_RANDOM_ROLE_NAME | 有分支 | 读取三段名字池并注入抽样索引 | 差分通过 |
| 1004 | MSG_UPDATE_ROLE_NAME | 有分支 | 角色快照、改名落盘及体力副作用已差分 | 差分通过 |
| 1005 | MSG_REQ_ROLE_BOARD | 有分支 | BackGroundHome 列表及当前看板，无落盘 | 差分通过 |
| 1006 | MSG_SAVE_BOARD_CHANGE | 有分支 | 保存看板，1001 先于 1006，保留原型权限校验局限 | 差分通过 |
| 1011 | MSG_REQ_FRONT_SETTINGS | 有分支 | 完整设置回包、无落盘副作用已差分 | 差分通过 |
| 1013 | MSG_SAVE_FRONT_SETTINGS | 有分支 | 分组合并、完整回包及结束存档已差分 | 差分通过 |
| 1014 | MSG_GM_CMD | 有分支 | 未接入 | 未分析 |
| 1015 | MSG_CHOOSE_INIT_LEADER | 有分支 | 队长合法性映射、落盘及后续推送已差分 | 差分通过 |
| 1016 | MSG_REQ_DOWNLOAD_REWARD | 有分支 | 原版即为空奖励，已执行原函数验证 | 差分通过 |
| 1052 | MSG_CALL_HERO | 有分支 | 未接入 | 未分析 |
| 1053 | MSG_USED_ITEM_UPGRADE | 有分支 | 未接入 | 未分析 |
| 1054 | MSG_UPGRADE_QUALITY | 有分支 | 未接入 | 未分析 |
| 1055 | MSG_UPGRADE_STAR | 有分支 | 未接入 | 未分析 |
| 1057 | MSG_HERO_AWAKEN | 有分支 | 未接入 | 未分析 |
| 1058 | MSG_USE_HERO_SURFACE | 有分支 | 未接入 | 未分析 |
| 1059 | MSG_CHANGE_DEVELOP_REMIND | 有分支 | 未接入 | 未分析 |
| 1060 | MSG_REQ_HERO_PROFILE | 有分支 | 未接入 | 未分析 |
| 1061 | MSG_GET_HERO_RECEIVED_GIFT | 有分支 | 未接入 | 未分析 |
| 1062 | MSG_SEND_HERO_GIFT | 有分支 | 未接入 | 未分析 |
| 1063 | MSG_OBTAIN_HERO_FAVOR_REWARD | 有分支 | 未接入 | 未分析 |
| 1065 | MSG_UPGRADE_ONE_HERO_FAVOR | 有分支 | 未接入 | 未分析 |
| 1066 | MSG_OBTAIN_FAVOR_LEVEL_AWARD | 有分支 | 未接入 | 未分析 |
| 1067 | MSG_REQ_COLLECT_INFO | 有分支 | 未接入 | 未分析 |
| 1068 | MSG_REQ_ACTIVATION_FETTER | 有分支 | 未接入 | 未分析 |
| 1070 | MSG_HERO_TASK_LIST | 有分支 | 未接入 | 未分析 |
| 1071 | MSG_SAVE_SURFACES_OFFSET | 有分支 | 未接入 | 未分析 |
| 1151 | MSG_REQ_RESOURCE_MAP | 有分支 | 配置地图、已有 resource_pass 和 auto_info 查询 | 查询验证通过 |
| 1152 | MSG_CHALLENGE_RESOURCE_MAP_POINT | 有分支 | 未接入 | 未分析 |
| 1202 | MSG_SAVE_TEAM | 有分支 | 未接入 | 未分析 |
| 1252 | MSG_USE_ITEM | 有分支 | 未接入 | 未分析 |
| 1254 | MSG_REQ_BAG | 有分支 | 同号回包、完整装备/道具及 bag_version 递增 | 差分通过 |
| 1265 | MSG_TRANS_FRAGS_TO_CURRENCY | 有分支 | 未接入 | 未分析 |
| 1301 | MSG_CHAPTERS | 有分支 | 地图前推送、完整章节及挂机数据已差分 | 差分通过 |
| 1302 | MSG_CHALLENGE_POINT | 有分支 | 未接入 | 未分析 |
| 1303 | MSG_SWEEP_POINT | 有分支 | 未接入 | 未分析 |
| 1304 | MSG_GAIN_CHAPTER_REWARD | 有分支 | 未接入 | 未分析 |
| 1306 | MSG_GAIN_ALL_CHAPTER_REWARDS | 有分支 | 未接入 | 未分析 |
| 1307 | MSG_GET_CHAPTER_AUTO_INFO | 有分支 | 已有巡逻信息查询，不领取奖励 | 差分通过 |
| 1308 | MSG_GAIN_AUTO_CHAPTER_REWARDS | 有分支 | 未接入 | 未分析 |
| 1309 | MSG_CHANGE_CHAPTER_AUTO_HERO | 有分支 | 未接入 | 未分析 |
| 1310 | MSG_OBTAIN_PASS_REWARDS | 有分支 | 未接入 | 未分析 |
| 1351 | MSG_BATTLE_END | 有分支 | 未接入 | 未分析 |
| 1354 | MSG_VERIFY_BATTLE | 有分支 | 未接入 | 未分析 |
| 1355 | MSG_REPORT_BATTLE | 有分支 | 未接入 | 未分析 |
| 1356 | MSG_REPORT_BATTLE_RES | 有分支 | 未接入 | 未分析 |
| 1359 | MSG_STOP_AUTO_FIGHT | 有分支 | 未接入 | 未分析 |
| 1502 | MSG_REQ_MAILS | 有分支 | 原型无邮件查询，非空状态拒绝，完整邮件未知 | 查询验证通过 |
| 1552 | MSG_REQ_TASK | 有分支 | 已有任务查询、奖励副作用及存档差分 | 差分通过 |
| 1555 | MSG_REQ_TASK_REWARD | 有分支 | 未接入 | 未分析 |
| 1559 | MSG_REQ_ONE_KEY_COLLECT_REWARD | 有分支 | 未接入 | 未分析 |
| 1601 | MSG_DRAWCARD_SYSTEM | 有分支 | 卡池及已有计数查询，保留 normal 池优先，不含抽卡 | 查询验证通过 |
| 1602 | MSG_DRAWCARD | 有分支 | 未接入 | 未分析 |
| 1603 | MSG_CONVERT_COUPON | 有分支 | 未接入 | 未分析 |
| 1651 | MSG_REQ_SHOP | 有分支 | 未接入 | 未分析 |
| 1652 | MSG_REQ_BUY_SHOP_ITEM | 有分支 | 未接入 | 未分析 |
| 1656 | MSG_REQ_RESOURCE_SHOP | 有分支 | 未接入 | 未分析 |
| 1657 | MSG_REQ_BUY_RESOURCE | 有分支 | 未接入 | 未分析 |
| 1658 | MSG_REQ_SURFACE_SHOP | 有分支 | 未接入 | 未分析 |
| 1660 | MSG_REQ_SURFACE_PRICE | 有分支 | 未接入 | 未分析 |
| 1663 | MSG_REQ_SHOP_MALL | 有分支 | 未接入 | 未分析 |
| 1668 | MSG_REQ_RECOMMEND_SHOP | 有分支 | 未接入 | 未分析 |
| 1670 | MSG_REQ_MONTH_CARD | 有分支 | 已有月卡查询及缺表回退差分 | 差分通过 |
| 1671 | MSG_GET_PAY_INFO | 有分支 | 未接入 | 未分析 |
| 1672 | MSG_PAY_SUCCESS_CONFIRM | 有分支 | 未接入 | 未分析 |
| 1676 | MSG_SELECT_BAG_REWARD | 有分支 | 未接入 | 未分析 |
| 1701 | MSG_REAL_ARENA_BASIC | 有分支 | 校验配置引用的原型本地赛季查询 | 查询验证通过 |
| 1702 | MSG_REAL_ARENA_INFO | 无独立分支 | B224 type=0 消费与 1701 相同的本地 arena_info | 查询验证通过 |
| 1752 | MSG_REQ_CHECK_BUY_MONTH_CARD | 有分支 | 读取 until 判断到期，不执行购买 | 差分通过 |
| 1802 | MSG_ARENA_INFO | 有分支 | 原型本地离线赛季查询，不含匹配和战斗 | 查询验证通过 |
| 1810 | MSG_ARENA_BASIC | 有分支 | 与 1802 共用本地查询模型 | 查询验证通过 |
| 1851 | MSG_WORLD_MAP | 有分支 | 登录和章节前的三张解锁推送已差分；独立请求未接入 | 未分析 |
| 1852 | MSG_ENTER_SCENE | 有分支 | 未接入 | 未分析 |
| 1853 | MSG_SCENE_MOVE | 有分支 | 未接入 | 未分析 |
| 1854 | MSG_SCENE_MINI | 有分支 | 未接入 | 未分析 |
| 1855 | MSG_SCENE_FIGHT | 有分支 | 未接入 | 未分析 |
| 1857 | MSG_SCENE_FINISH_CASE | 有分支 | 未接入 | 未分析 |
| 1863 | MSG_SCENE_SPEEDUP | 有分支 | 未接入 | 未分析 |
| 1864 | MSG_QUIT_SCENE | 有分支 | 未接入 | 未分析 |
| 1951 | MSG_REQ_ACTIVITIES | 有分支 | 登录活动推送及独立快照查询 | 查询验证通过 |
| 1955 | MSG_REQ_FACE_NOTICES | 有分支 | 无公告查询，非空状态拒绝 | 查询验证通过 |
| 1983 | MSG_LOTTERY | 有分支 | 未接入 | 未分析 |
| 1984 | MSG_RESET_LOTTERY | 有分支 | 未接入 | 未分析 |
| 1985 | MSG_GLOBAL_WISH | 有分支 | 未接入 | 未分析 |
| 1986 | MSG_LOTTERY_REWARD | 有分支 | 未接入 | 未分析 |
| 1991 | MSG_DICE_SOLUTION | 有分支 | 未接入 | 未分析 |
| 2016 | MSG_GET_PVP_ROOM | 有分支 | B224 无房间 room_info:null，非空状态报错 | 查询验证通过 |
| 2051 | MSG_WEAR_EQUIP_NEW | 有分支 | 未接入 | 未分析 |
| 2052 | MSG_EQUIP_UPGRADE_LEVEL_NEW | 有分支 | 未接入 | 未分析 |
| 2053 | MSG_EQUIP_UPGRADE_QUALITY_NEW | 有分支 | 未接入 | 未分析 |
| 2054 | MSG_EQUIP_UPGRADE_STAR_NEW | 有分支 | 未接入 | 未分析 |
| 2055 | MSG_LOCK_EQUIP_NEW | 有分支 | 未接入 | 未分析 |
| 2101 | MSG_REQ_TOWER_INFO | 有分支 | 原型无塔进度查询，已有塔进度报错 | 查询验证通过 |
| 2102 | MSG_CHALLENGE_TOWER_POINT | 有分支 | 未接入 | 未分析 |
| 2301 | MSG_REQ_COLLECT_LIST | 有分支 | 未接入 | 未分析 |
| 2351 | MSG_FAIR_ARENA_INFO | 有分支 | 原型本地公平赛季查询，不含英雄池生成或匹配 | 查询验证通过 |
| 2401 | MSG_GET_SUMMIT_BASE_INFO | 有分支 | 复用原型巅峰关闭查询模型，独立于 2419 推送 | 查询验证通过 |
| 2551 | MSG_REQUEST_ROGUE_INFO | 无分支，编号来自 B224 | 维护者确认的未开局初态；正式 Rogue 未实现 | 查询验证通过 |
| 2601 | MSG_GET_FT_INFO | 无分支，编号来自 B224 | 维护者确认的协战初态；正式协战未实现 | 查询验证通过 |
| 10101 | MSG_GET_FRIENDS | 有分支 | 原型无好友列表及 max_count:0，非空状态报错 | 查询验证通过 |
| 10102 | MSG_GET_RECOMMEND_FRIENDS | 有分支 | 原型空推荐列表和 max_count，非空状态拒绝 | 查询验证通过 |
| 10115 | MSG_GET_CHAT_INFO | 有分支 | 原型单频道 chat_info，初始化序列已验证 | 查询验证通过 |
| 10124 | MSG_REQ_BLACKLIST | 有分支 | 空黑名单查询，非空状态报错 | 查询验证通过 |
| 10129 | MSG_GET_CHAT_LIST | 有分支 | B224 contactList 空联系人，保留与原型的字段差异记录 | 查询验证通过 |
| 10151 | MSG_GET_CLUB | 有分支 | 无公会 member_info 与 clubs 查询，非空状态报错 | 查询验证通过 |
| 10152 | MSG_GET_RECOMMEND_CLUBS | 有分支 | 原型无公会列表和 member_info | 查询验证通过 |
| 10157 | MSG_GET_CLUB_INVITES | 有分支 | 原型无公会列表和 member_info | 查询验证通过 |
| 10167 | MSG_SEARCH_CLUB | 有分支 | 原型无公会搜索结果 | 查询验证通过 |
| 10175 | MSG_LAST_RECOMMEND_CLUBS_REQ | 有分支 | 原型无公会列表和 member_info | 查询验证通过 |
| 未映射 | MSG_GAIN_STAR_REWARD | 有分支 | 协议映射缺少编号 | 受阻 |

## Go 主动推送

登录顺序与字段已经通过合成配置和真实配置差分；这不表示同号客户端请求也已迁移

| 编号 | 命令 | 当前情况 |
|---:|---|---|
| 1001 | MSG_ROLE_INFO_RES | 原版角色字段、货币和体力 |
| 1017 | MSG_LOGIN_INFO_RES | Role、Resource.heros、summit 完整快照 |
| 1051 | MSG_HEROS_RES | 完整英雄记录与装备加成 |
| 1201 | MSG_TEAMS_RES | teams、used_team、expand_teams |
| 1851 | MSG_WORLD_MAP | world、region、main 三份解锁映射 |
| 1012 | MSG_SYSTEM_UNLOCK_RES | 原版等级及任务门规则 |
| 906 | MSG_PUSH_REMINDS | 原版空提醒数组 |
| 903 | MSG_RED_POINTS_RES | 原版空红点对象 |
| 2419 | MSG_SYNC_BASE_INFO | 巅峰配置及玩家排行/分数 |
| 1951 | MSG_REQ_ACTIVITIES | 签到、抽卡关联和 SHOPBAG 活动 |

## 本轮证据约定

日期：2026-10-25

状态“差分通过”限定于 [已有路径对齐记录](existing-paths-alignment.md) 的样本和失败语义边界，不代表实机通过
原 Python 路径统一为 `2-SERVER/protocol/gxjy_server.py`
证据统一为 `internal/game/testdata/existing_paths_python.json`、`TestExistingPathsMatchPython` 与 `TestWebSocketMatchesPython`
真实表证据由 `scripts/check_python_diff.py` 生成到忽略目录，原函数及源文件 SHA-256 可追溯

共享类型：

- `Role`：result:int、id/uid:int|string、name/role_name:string、level/exp:int、exterior:object、server_timestamp/server_time:int（秒）、stamina_pool:{storage:int,max_storage:int}；非空时包含 currency:map<string,int>
- `exterior`：sex:int、head_image/head_rect/back_now/leader/wife:string、mode:any|null、signature:string、role_name_number:int、system_opens:map<string,bool>、address:{province:string,city:string}、birthday:{month:int,day:int}
- `HeroRecord`：level/exp/cost:int、base_attr:{quality_id:string,rarity:string,star:{star_id:string,star_level:int},hero_model:{hero_surface:string,leader_surface:string},awaken_id?:string}、hero_surfaces:map<string,string>、favor_id:string、favor_exp:int、leader_attrs:map<string,int>、skills:{hero_skills:object,leader_skills:object,leader_team_skills:object}
- `Team`：team_id:int|string、leader:string、members:array、team_name:string、leader_team_skills:object、slots:array<string>；英雄侧羁绊 id 指向 SkillBase，编队侧指向 LeaderTeamSkill
- `BagItem`：count:int、template_id:string、item_type:string、data:object；装备 data 包含 user:string、attrs:map<string,int>、locked:bool、exp:int、level_id/quality/star/template_id:string、enchantment:object；普通道具 data 为 is_new:bool
- `Summit`：result:int、summitBaseId:string、stage/level/rank/score:int、actInfo:{isActOpen:bool,openTime:int,closeTime:int}、heroes/records:array
- `Activity`：status/start_time/end_time/is_new/num:int、unlock:bool、reward_info:object；签到另含 extra_info:object
- 设置值支持 JSON 值；非法配置和超出编码范围的输入不计入当前差分覆盖

## 101 MSG_TYPE_AUTH

| 项目 | 记录 |
|---|---|
| 请求 | 对象，可带 msg/seq 信封；无业务必填字段 |
| 回包 | result:int、cmd:string |
| 主动推送顺序 | 101→1001→1017→1051→1201→1851→1012→906→903→2419→1951 |
| 玩家状态变化 | 重播标记清理；英雄默认进度；体力恢复/封顶可落盘 |
| 配置表 | 登录依赖的 33 张表见 game.requiredTables |
| Python 位置 | Session.dispatch、send_login_info、send_login_pushes |
| Go 位置 | internal/game/runtime.go、login_state.go、heroes.go、equipment.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 901 MSG_HEARTBEAT

| 项目 | 记录 |
|---|---|
| 请求 | clientTimestamp?:JSON值，可带信封 |
| 回包 | result:int、cmd:string、server_timestamp:int（毫秒）、clientTimestamp?:原值 |
| 主动推送顺序 | 仅901 |
| 玩家状态变化 | 无 |
| 配置表 | 无 |
| Python 位置 | Session.dispatch MSG_HEARTBEAT |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1003 MSG_RANDOM_ROLE_NAME

| 项目 | 记录 |
|---|---|
| 请求 | 无业务字段 |
| 回包 | result:int、cmd:string、role_name:string |
| 主动推送顺序 | 仅1003 |
| 玩家状态变化 | 消耗三次名字池抽样，不写存档 |
| 配置表 | NameFront、NameMiddle、NameLast |
| Python 位置 | random_role_name、Session.dispatch |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1004 MSG_UPDATE_ROLE_NAME

| 项目 | 记录 |
|---|---|
| 请求 | role_name?:string |
| 回包 | Role 加 cmd:string |
| 主动推送顺序 | 仅1004 |
| 玩家状态变化 | 非空名字写盘；构造 Role 可恢复体力 |
| 配置表 | PlayerLevel、LeaderBase、UnlockSystem |
| Python 位置 | Session.dispatch、make_role_info |
| Go 位置 | internal/game/runtime.go、login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1011 MSG_REQ_FRONT_SETTINGS

| 项目 | 记录 |
|---|---|
| 请求 | 无业务字段 |
| 回包 | result:int、cmd:string、settings:{common:object,need_reset:object}、version:int=0 |
| 主动推送顺序 | 仅1011 |
| 玩家状态变化 | 无，不写盘 |
| 配置表 | 无 |
| Python 位置 | Session.dispatch MSG_REQ_FRONT_SETTINGS |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1013 MSG_SAVE_FRONT_SETTINGS

| 项目 | 记录 |
|---|---|
| 请求 | settings?:object，common/need_reset?:object |
| 回包 | 同1011，cmd 为当前名称 |
| 主动推送顺序 | 仅1013 |
| 玩家状态变化 | 两个组按键合并，其他组不应用；settings 为对象时写盘 |
| 配置表 | 无 |
| Python 位置 | Session.dispatch MSG_SAVE_FRONT_SETTINGS |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1015 MSG_CHOOSE_INIT_LEADER

| 项目 | 记录 |
|---|---|
| 请求 | leader_id?:string、role_name?:string |
| 回包 | Role 加 cmd:string |
| 主动推送顺序 | 1015→1051→1201→1851→1012→906→903→2419→1951 |
| 玩家状态变化 | 按原函数映射并校验队长、性别与名称，保存后推送；快照含体力和英雄默认状态副作用 |
| 配置表 | HeroBase、LeaderBase 及登录所需表 |
| Python 位置 | Session.dispatch、leader_hero_id、canonical_leader_id |
| Go 位置 | internal/game/runtime.go、heroes.go、login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1016 MSG_REQ_DOWNLOAD_REWARD

| 项目 | 记录 |
|---|---|
| 请求 | 无业务字段 |
| 回包 | result:int、cmd:string、rewards:array=[] |
| 主动推送顺序 | 仅1016 |
| 玩家状态变化 | 无 |
| 配置表 | 无 |
| Python 位置 | Session.dispatch MSG_REQ_DOWNLOAD_REWARD |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1254 MSG_REQ_BAG

| 项目 | 记录 |
|---|---|
| 请求 | 无业务字段 |
| 回包 | result:int、cmd:string、items:map<string,BagItem>、max_count:int=0、version:int |
| 主动推送顺序 | 仅1254；不回1251 |
| 玩家状态变化 | bag_version 每次递增；此请求不主动保存 |
| 配置表 | EquipBase、EquipExp、EquipQuality、EquipStar、SkillAttr、BaseConfig |
| Python 位置 | Session.dispatch、bag_payload、bag_items_map、equip_instance_data |
| Go 位置 | internal/game/runtime.go、equipment.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1301 MSG_CHAPTERS

| 项目 | 记录 |
|---|---|
| 请求 | 无业务字段 |
| 回包 | result:int、cmd:string、version:int、chapters:map<id,{id:string,unlock:bool,points:object,star:int}>、pass_rewards:object、auto_info:{today_rewards:array,tomorrow_rewards:array,today_received:bool}；point 含 id:string、pass:bool、num:int、star_idxs:array<int>、stars:int、type:string |
| 主动推送顺序 | 1851→1301 |
| 玩家状态变化 | 缺失时创建 patrol_auto，不主动保存 |
| 配置表 | FightMap、FightPoint、BaseConfig、RewardConfig、ExploreWorldMap/Region/Main |
| Python 位置 | Session.dispatch、chapter_payload、map_unlocked、patrol_auto_info |
| Go 位置 | internal/game/runtime.go、chapters.go、login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1001 MSG_ROLE_INFO_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | Role |
| 主动推送顺序 | 101 后、1017 前 |
| 玩家状态变化 | 体力恢复/封顶及 stamina_at；必要时保存 |
| 配置表 | PlayerLevel、LeaderBase、UnlockSystem |
| Python 位置 | make_role_info、send_role_info |
| Go 位置 | internal/game/login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1017 MSG_LOGIN_INFO_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、Role:Role、Resource:{heros:map<string,HeroRecord>}、summit:Summit |
| 主动推送顺序 | 1001 后、1051 前 |
| 玩家状态变化 | 角色体力与英雄默认状态副作用 |
| 配置表 | PlayerLevel、Hero*、Skill*、Leader*、Equip*、BaseConfig、SummitBase、UnlockSystem |
| Python 位置 | send_login_info |
| Go 位置 | internal/game/runtime.go、heroes.go、login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1051 MSG_HEROS_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、heros:map<string,HeroRecord> |
| 主动推送顺序 | 1017 后、1201 前；1015 后第一条推送 |
| 玩家状态变化 | 缺失英雄进度补默认值；启动先补起手装备并保存 |
| 配置表 | HeroBase/Data/Quality/Star/Awaken/Favor/CostAttr、SkillBase/Special、LeaderBase/TeamSkill、Equip*、SkillAttr、BaseConfig |
| Python 位置 | owned_heroes、hero_record、ensure_starter_equips |
| Go 位置 | internal/game/heroes.go、equipment.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1201 MSG_TEAMS_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、teams:map<string,Team>、used_team:map<string,int>、expand_teams:map<string,Team> |
| 主动推送顺序 | 1051 后、1851 前 |
| 玩家状态变化 | canonical_leader_id 可能补英雄默认状态；不保存 |
| 配置表 | HeroBase、LeaderBase、LeaderTeamSkill、SkillBase、BaseConfig、ResourcesPoint、ResourcesMap |
| Python 位置 | team_payload |
| Go 位置 | internal/game/login_state.go、heroes.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1851 MSG_WORLD_MAP（推送）

| 项目 | 记录 |
|---|---|
| 请求 | 登录或1301触发；独立1851请求未接入 |
| 回包 | result:int、unlock_world_map/unlock_region_map/unlock_main_map:map<string,bool> |
| 主动推送顺序 | 登录1201后1012前；1301前 |
| 玩家状态变化 | 无 |
| 配置表 | ExploreWorldMap、ExploreWorldRegion、ExploreWorldMain |
| Python 位置 | world_map_unlock_payload |
| Go 位置 | internal/game/login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1012 MSG_SYSTEM_UNLOCK_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、unlock_ids:map<string,bool> |
| 主动推送顺序 | 1851 后、906 前 |
| 玩家状态变化 | 无 |
| 配置表 | UnlockSystem |
| Python 位置 | loader_unlock_ids、login_unlock_ids、_unlock_satisfied |
| Go 位置 | internal/game/login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 906 MSG_PUSH_REMINDS

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、reminds:array=[] |
| 主动推送顺序 | 1012 后、903 前 |
| 玩家状态变化 | 无 |
| 配置表 | 无 |
| Python 位置 | send_login_pushes |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 903 MSG_RED_POINTS_RES

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送 |
| 回包 | result:int、red_points:object={}  |
| 主动推送顺序 | 906 后、2419 前 |
| 玩家状态变化 | 无 |
| 配置表 | 无 |
| Python 位置 | send_login_pushes |
| Go 位置 | internal/game/runtime.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 2419 MSG_SYNC_BASE_INFO

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送；2401是不同请求，本轮未接入 |
| 回包 | Summit |
| 主动推送顺序 | 903 后、1951 前 |
| 玩家状态变化 | 无 |
| 配置表 | SummitBase |
| Python 位置 | summit_base_info、send_login_pushes |
| Go 位置 | internal/game/login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

## 1951 MSG_REQ_ACTIVITIES（推送）

| 项目 | 记录 |
|---|---|
| 请求 | 主动推送；独立1951请求未接入 |
| 回包 | result:int、activities:map<string,Activity> |
| 主动推送顺序 | 2419 后 |
| 玩家状态变化 | 无 |
| 配置表 | DrawCard、Activity |
| Python 位置 | draw_activity_payload、send_login_pushes |
| Go 位置 | internal/game/login_state.go |
| 差分证据 | 本轮共享样本及业务/双协议差分测试 |
| 当前状态 | 差分通过 |
| 实机证据 | 本轮未进行客户端实机验证 |

