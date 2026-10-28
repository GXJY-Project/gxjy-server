# 2551 初始状态适配草案

状态：已验证客户端字段和触发条件；正式业务未知；维护者已明确确认接入未开局初态

编号来自 B224 的 SERVER_Protocol.MSG_REQUEST_ROGUE_INFO，不从相邻编号猜测
原脚本 `original-index.js` SHA-256 为 `89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6`
`RoguelikeModel.requestForLoading` 404865-404869 → `requestRogue` 404618-404625 → `synchronize` 403875-403918
实际 UnlockSystem.Rogue_Unlock 要求 Level >= 17
原型 SHA-256 为 `13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25`，Session.dispatch 没有此分支

草案使用客户端构造器 403801-403836 的未开局值，不分配难度、英雄、天赋、次数或积分
具体可审阅响应见 `internal/game/testdata/bili224_rogue_initial.json`
请求为空对象；响应包含显式的未开局游标、空难度/可携带英雄/图鉴、空已领奖集合和零积分，重置时间保持客户端初值 -1
仅允许没有 Rogue 记录的 B224 登录查询；已有非空或损坏 Rogue 状态必须返回错误

局限：这不能证明正式服务如何初始化难度、次数或周重置，也不支持进入 Rogue 战斗和领取奖励
不能将此适配视为 Rogue 系统移植完成
维护者在审阅草案后回复「接入未开局初态，继续打通登录」，现已在 bili-224 路径接入 Runtime.Handle
独立测试和客户端原方法验证见 `home_queries_test.go`、`board_test.go`、`check_login_init_client.cjs`
