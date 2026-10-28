package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// initializationQuery 补齐登录调用链的查询，来源和 B224 差异见 docs/login-init-implementation.md
// 社交、房间和公会目前仅覆盖原型的无数据案例，已有非空状态必须显式失败
func (r *Runtime) initializationQuery(command uint16, request record) (record, error) {
	switch command {
	case CommandFriends, CommandRecommendFriends:
		if err := r.requireEmptyInitialization("friends"); err != nil {
			return nil, err
		}
		return record{"friends": record{}, "max_count": 0}, nil
	case CommandBlacklist:
		if err := r.requireEmptyInitialization("blacklist"); err != nil {
			return nil, err
		}
		return record{"blacklist": []any{}}, nil
	case CommandChatList:
		if err := r.requireEmptyInitialization("contactList", "chat_list"); err != nil {
			return nil, err
		}
		// 维护者已确认以 B224 FriendModel.synchronizeList 的 contactList 为准
		return record{"contactList": record{}}, nil
	case CommandPVPRoom:
		if err := r.requireEmptyInitialization("room_info", "rooms"); err != nil {
			return nil, err
		}
		// B224 requestRoomInfo(false) 在 room_info 为假时调用继续登录的回调
		return record{"room_info": nil}, nil
	case CommandGuild, CommandRecommendGuilds, CommandGuildInvites, CommandSearchGuild, CommandLastGuilds:
		if err := r.requireEmptyInitialization("club_info", "clubs", "member_info"); err != nil {
			return nil, err
		}
		return record{"version": 1, "member_info": record{"invite_num": 0, "quit_cd": 0, "new_post_set": 0}, "clubs": record{}}, nil
	case CommandTowerInfo:
		if err := r.requireEmptyInitialization("tower", "tower_points", "tower_rewards"); err != nil {
			return nil, err
		}
		for id := range obj(r.state["points"]) {
			if strings.HasPrefix(id, "Tower_") {
				return nil, fmt.Errorf("尚未实现已有塔进度的查询")
			}
		}
		return record{"points": record{}, "cum_rewards": record{}, "end_time": 0}, nil
	case CommandResourceMap:
		return r.resourceMaps()
	case CommandDrawSystem:
		return r.drawSystem()
	case CommandRealArenaBasic, CommandRealArenaInfo, CommandArenaBasic, CommandArenaInfo, CommandFairArenaInfo:
		return r.initialArena(command, request)
	case CommandActivities:
		return r.activities(), nil
	case CommandStaminaRecovery:
		// Python 9509 只查询此字段，不据此执行恢复体力或扣费
		return record{"next_recover_time": 0}, nil
	case CommandChapterAutoInfo:
		if _, err := initializationObject(r.state["patrol_auto"], "patrol_auto"); err != nil {
			return nil, err
		}
		return record{"auto_info": r.patrol(), "rewards": []any{}}, nil
	case CommandMails:
		if err := r.requireEmptyInitialization("mails", "mail", "mail_list"); err != nil {
			return nil, err
		}
		return record{"mails": record{}, "max_count": 0}, nil
	case CommandFaceNotices:
		if err := r.requireEmptyInitialization("notices", "face_notices"); err != nil {
			return nil, err
		}
		return record{"notices": []any{}}, nil
	case CommandSummitRequest:
		if len(r.row("SummitBase", "Summit_1")) == 0 {
			return nil, fmt.Errorf("巅峰基础查询缺少 SummitBase.Summit_1")
		}
		return r.summit(), nil
	case CommandCheckMonthCard:
		if _, err := initializationObject(r.state["month_card"], "month_card"); err != nil {
			return nil, err
		}
		stored := ensure(r.state, "month_card")
		id := or(request["id"], or(request["month_card_id"], request["card_id"]))
		entry := record{}
		if truth(id) {
			value := stored[str(id)]
			if truth(value) {
				var ok bool
				entry, ok = value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("月卡记录不是对象")
				}
			}
		}
		// 沿用 Python 9008-9021 的非法整数回退，布尔 true 转 1
		until := integer(or(entry["until"], 0), 0)
		return record{"buy": until != 0 && until > r.now(), "inactive_time": until}, nil
	case CommandRogueInfo:
		if r.options.ClientProfile != "bili-224" {
			return nil, fmt.Errorf("2551 未开局初态适配仅支持 bili-224")
		}
		if err := r.requireEmptyInitialization("rogue", "rogue_info", "roguelike"); err != nil {
			return nil, err
		}
		// 维护者确认的临时登录适配，值来自 B224 RoguelikeModel 构造器
		// 不声明正式难度、可携带英雄、次数、图鉴和周重置已实现
		return record{
			"remain_times": 0, "reset_time": -1, "week_score": 0, "week_score_max": 0,
			"received_score": []any{}, "unlock_talent": []any{}, "cur_gold": 0,
			"cur_hp": 0, "hp_max": 0, "carry_hero": "", "hero_property": record{},
			"cur_pace":  record{"difficulty": "", "chapter": "", "node": ""},
			"cur_anger": 0, "max_anger": 0, "trophy": []any{}, "difficulty": record{},
			"carry_hero_map": record{}, "skill_hero": []any{}, "wish_hero": []any{}, "pictorial": []any{},
		}, nil
	case CommandFightTogetherInfo:
		if r.options.ClientProfile != "bili-224" {
			return nil, fmt.Errorf("2601 初始空态适配仅支持 bili-224")
		}
		if err := r.requireEmptyInitialization("fight_together", "playerFTInfo", "bossList", "actBossIds"); err != nil {
			return nil, err
		}
		// 维护者确认的临时登录适配，字段与零值来自 B224 FightTogetherModel 构造器
		// 不推断正式额度、Boss 进度、活动或重置规则，详见 docs/board-and-ft-init.md
		return record{
			"playerFTInfo": record{"maxCoin": 0, "gainCoin": 0, "maxDifficultyId": "", "assistNum": 0,
				"resetTime": record{}, "collaborator": []any{}},
			"bossList": []any{}, "actBossIds": []any{},
		}, nil
	case CommandClientTracking:
		// Python Session.dispatch 对 MSG_CLIENT_TRACKING 明确只确认接收，不保存统计
		// 这只适用于该已确认命令，未知命令仍返回错误
		return record{}, nil
	default:
		return nil, fmt.Errorf("未实现初始化查询：%d", command)
	}
}

func (r *Runtime) requireEmptyInitialization(keys ...string) error {
	for _, key := range keys {
		value := r.state[key]
		switch v := value.(type) {
		case nil:
			continue
		case map[string]any:
			if len(v) == 0 {
				continue
			}
		case []any:
			if len(v) == 0 {
				continue
			}
		}
		return fmt.Errorf("尚未支持的非空或损坏初始化状态：%s", key)
	}
	return nil
}

func initializationObject(value any, field string) (record, error) {
	if value == nil {
		return record{}, nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("初始化字段 %s 不是对象", field)
	}
	return m, nil
}

func initializationInteger(value any, field string) (int64, error) {
	// 缺失/空值沿用原型的零值，非法内容和溢出不能回退为零
	if value == nil {
		return 0, nil
	}
	switch value.(type) {
	case []any, map[string]any:
		return 0, fmt.Errorf("初始化字段 %s 不是数值", field)
	}
	n, err := checkedInteger(value)
	if err != nil {
		return 0, fmt.Errorf("初始化字段 %s: %w", field, err)
	}
	return n, nil
}

// resourceMaps 对齐 Python resource_map_payload 与 _resource_challenge_num
func (r *Runtime) resourceMaps() (record, error) {
	passed, err := initializationObject(r.state["resource_pass"], "resource_pass")
	if err != nil {
		return nil, err
	}
	challengeNum := int64(2) // 原型无次数配置时的显式默认值，不代表正式每日次数规则
	for _, key := range []string{"Resources_DailyNum", "Resources_ChallengeNum", "ResourcesMap_DailyNum", "Resources_HighRewardNum"} {
		value := r.row("BaseConfig", key)["Config"]
		if value == nil {
			continue
		}
		n, err := initializationInteger(value, key)
		if err != nil {
			return nil, err
		}
		if n != 0 {
			challengeNum = n
			break
		}
	}
	maps := record{}
	for _, id := range r.keys["ResourcesMap"] {
		if strings.HasPrefix(id, "_") {
			continue
		}
		row, ok := r.tables["ResourcesMap"][id].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("资源地图配置不是对象：%s", id)
		}
		if value := row["ResourcesPoint"]; value != nil {
			if _, ok := value.([]any); !ok {
				return nil, fmt.Errorf("资源地图 %s ResourcesPoint 不是数组", id)
			}
		}
		points := record{}
		for _, value := range arr(row["ResourcesPoint"]) {
			pid := text(value)
			if len(r.row("ResourcesPoint", pid)) == 0 {
				return nil, fmt.Errorf("资源地图 %s 引用了不存在的 ResourcesPoint.%s", id, pid)
			}
			rec, err := initializationObject(passed[pid], "resource_pass/"+pid)
			if err != nil {
				return nil, err
			}
			n, err := initializationInteger(rec["pass_num"], "resource_pass/"+pid+"/pass_num")
			if err != nil {
				return nil, err
			}
			achievements := []any{}
			if value := rec["achievement"]; value != nil {
				var ok bool
				achievements, ok = value.([]any)
				if !ok {
					return nil, fmt.Errorf("资源关卡 %s achievement 不是数组", pid)
				}
			}
			points[pid] = record{"achievement": copyValue(achievements), "pass_num": n}
		}
		maps[id] = record{"points": points, "extra_reward_num": challengeNum, "challenge_num": challengeNum}
	}
	return record{"version": 1, "resource_maps": maps, "auto_info": r.patrol()}, nil
}

// drawSystem 只迁移原型的卡池查询，不实现抽卡、扣费或奖励发放
func (r *Runtime) drawSystem() (record, error) {
	state, err := initializationObject(r.state["gacha"], "gacha")
	if err != nil {
		return nil, err
	}
	times, err := initializationObject(state["pool_times"], "gacha/pool_times")
	if err != nil {
		return nil, err
	}
	total, err := initializationInteger(state["total"], "gacha/total")
	if err != nil {
		return nil, err
	}
	pity, err := initializationInteger(state["pity"], "gacha/pity")
	if err != nil {
		return nil, err
	}
	pools, probable := record{}, record{}
	order := []string{}
	for _, id := range r.keys["DrawCard"] {
		if strings.HasPrefix(id, "_") {
			continue
		}
		row, ok := r.tables["DrawCard"][id].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("抽卡配置不是对象：%s", id)
		}
		cost, err := initializationIntegerList(row["CouponCost"], "DrawCard/"+id+"/CouponCost")
		if err != nil {
			return nil, err
		}
		number, err := initializationIntegerList(row["Number"], "DrawCard/"+id+"/Number")
		if err != nil {
			return nil, err
		}
		n, err := initializationInteger(times[id], "gacha/pool_times/"+id)
		if err != nil {
			return nil, err
		}
		coupon := "IC_DrawNormal"
		if values, ok := row["Coupon"].([]any); ok && len(values) > 0 {
			coupon = str(values[0])
		} else if truth(row["Coupon"]) {
			coupon = str(row["Coupon"])
		}
		pools[id] = record{"id": id, "draw_times": n, "records": []any{}, "preselection_value": 0, "fresh": true,
			"pity": pity, "pity_limit": 80, "cost": cost, "number": number, "coupon": coupon}
		order = append(order, id)
		// 0.02 和 80 分别来自原型 GACHA_BASE_SIX / GACHA_PITY_LIMIT
		probable[id] = 0.02
		if countType := text(row["CountType"]); countType != "" {
			probable[countType] = 0.02
		}
	}
	if len(pools) == 0 {
		return nil, fmt.Errorf("DrawCard 没有可用于初始化的卡池")
	}
	// B224 默认选第一个池，保留 gacha_system 的 normal 优先、同组按 ID 排序
	sort.Slice(order, func(i, j int) bool {
		a := strings.ToLower(text(r.row("DrawCard", order[i])["Type"])) == "normal"
		b := strings.ToLower(text(r.row("DrawCard", order[j])["Type"])) == "normal"
		if a != b {
			return a
		}
		return order[i] < order[j]
	})
	return record{"total": total, "ssr_probable": probable, "card_pool": orderedInitializationObject{keys: order, values: pools}}, nil
}

// orderedInitializationObject 在 JSON 和网络编码时保持卡池键顺序，不改变 Message 或分发接口
type orderedInitializationObject struct {
	keys   []string
	values record
}

func (o orderedInitializationObject) OrderedKeys() []string       { return append([]string{}, o.keys...) }
func (o orderedInitializationObject) OrderedValue(key string) any { return o.values[key] }

func (o orderedInitializationObject) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			out.WriteByte(',')
		}
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(o.values[key])
		if err != nil {
			return nil, err
		}
		out.Write(name)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func initializationIntegerList(value any, field string) ([]any, error) {
	if value == nil {
		return []any{}, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("初始化字段 %s 不是数组", field)
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		n, err := initializationInteger(value, field)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// initialArena 保留原型的本地赛季模型，配置引用必须存在
// 这不是正式赛季、匹配、排名或奖励业务，已有非空竞技场存档不能被该模型覆盖
func (r *Runtime) initialArena(command uint16, request record) (record, error) {
	if err := r.requireEmptyInitialization("arena", "real_arena", "fair_arena"); err != nil {
		return nil, err
	}
	table, id := "ArenaBase", "Arena_Season1"
	switch command {
	case CommandRealArenaBasic, CommandRealArenaInfo:
		table, id = "ArenaRealTimeBase", "ArenaRealTime_1"
	case CommandFairArenaInfo:
		table, id = "ArenaFairPlayBase", "ArenaFairPlay_Week1"
	}
	if _, err := r.catalog.Row(table, id); err != nil {
		return nil, fmt.Errorf("本地原型赛季引用无效：%w", err)
	}
	if command == CommandRealArenaInfo {
		kind, err := initializationInteger(request["type"], "type")
		if err != nil {
			return nil, err
		}
		if kind != 0 {
			return nil, fmt.Errorf("实时竞技场仅实现登录查询 type=0")
		}
	}
	if command == CommandFairArenaInfo {
		return record{"arena": record{"season_config_id": id, "status": 1, "win_time": 0, "max_win_num": 0,
			"lose_time": 0, "reset_time": 0, "rewards": []any{}, "heroes": []any{}}}, nil
	}
	season := record{"season_id": 1, "season_config_id": id, "end_time": int64(4102444800000)}
	if command == CommandRealArenaBasic || command == CommandRealArenaInfo {
		season["status"] = 1
		// 1702 消费与 1701 相同的 arena_info；B224:116540 直接设置其中 is_match
		return record{"arena": record{"season": season, "arena_info": record{"rank": 1, "score": 1000, "rewards": []any{}}}}, nil
	}
	return record{"arena": record{"season": season, "rank": 1, "score": 1000, "opponents": []any{}}}, nil
}
