package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
)

// Store 隔离玩家持久化，业务保存时保留整个状态对象
type Store interface {
	Load(any) (bool, error)
	Save(any) error
}

// Options 注入 Python 差分所需的时钟、抽样索引与序章开关
type Options struct {
	ClientProfile  string
	Now            func() time.Time
	Choose         func(int) int
	SkipGuide      bool
	ReplayPrologue bool
}

// Message 是业务产生的有序回包或推送
type Message struct {
	Command uint16         `json:"command"`
	Payload map[string]any `json:"payload"`
}

// Runtime 承载现有命令业务，不依赖网络会话
type Runtime struct {
	mu         sync.Mutex
	state      record
	tables     map[string]record
	keys       map[string][]string
	store      Store
	options    Options
	err        error
	retained   record
	catalog    *hero.Catalog
	stateKeys  map[string][]string
	objectKeys func(string, ...string) ([]string, error)
}

var requiredTables = strings.Fields("HeroBase HeroData HeroQuality HeroStar HeroAwaken HeroFavor SkillBase SkillSpecial LeaderBase LeaderTeamSkill HeroCostAttr BaseConfig EquipBase EquipExp EquipQuality EquipStar SkillAttr PlayerLevel UnlockSystem ExploreWorldMap ExploreWorldRegion ExploreWorldMain ResourcesPoint ResourcesMap FightMap FightPoint RewardConfig DrawCard Activity SummitBase NameFront NameMiddle NameLast")

func defaults() record {
	return record{"role_id": int64(10001), "role_name": "LocalPlayer", "level": int64(1), "exp": int64(0), "leader": nil, "sex": int64(1), "head_image": "PHM_H001", "head_rect": "PHF_001", "back_now": "BGP_G008", "currency": record{}, "bag_items": record{}, "stamina_pool": record{"storage": 0, "max_storage": 0}, "stamina_at": 0, "system_opens": record{}, "heroes": []any{}, "points": record{}, "custom": record{"common": record{}, "need_reset": record{}}}
}

// New 按 load_player、ensure_starter_equips 的真实启动顺序构建状态
func New(source hero.TableSource, store Store, options Options) (*Runtime, error) {
	if options.ClientProfile != "" && options.ClientProfile != "bili-224" {
		return nil, fmt.Errorf("不支持的客户端配置契约：%s", options.ClientProfile)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Choose == nil {
		options.Choose = rand.IntN
	}
	r := &Runtime{state: defaults(), tables: map[string]record{}, keys: map[string][]string{}, store: store, options: options, retained: record{}, stateKeys: map[string][]string{}}
	catalog := hero.NewCatalog(source)
	r.catalog = catalog
	for _, name := range requiredTables {
		t, err := catalog.Table(name)
		if err != nil {
			return nil, fmt.Errorf("加载业务配置 %s: %w", name, err)
		}
		r.tables[name] = t
		r.keys[name], err = catalog.Keys(name)
		if err != nil {
			return nil, err
		}
	}
	if source, ok := source.(interface {
		ObjectKeys(string, ...string) ([]string, error)
	}); ok {
		r.objectKeys = source.ObjectKeys
		keys, err := source.ObjectKeys("BaseConfig", "Point_Box_Config", "Config")
		if err != nil {
			return nil, err
		}
		r.keys["BaseConfig/Point_Box_Config/Config"] = keys
	}
	if store != nil {
		var loaded orderedPlayer
		found, err := store.Load(&loaded)
		if err != nil {
			return nil, err
		}
		if found {
			disk := loaded.values
			if err := validatePlayerShapes(disk); err != nil {
				return nil, err
			}
			r.stateKeys = loaded.keys
			// 原版忽略未知 null；按迁移规则仍在落盘时保留，不混入业务快照
			for k, v := range disk {
				if _, known := r.state[k]; !known && v == nil {
					r.retained[k] = nil
				}
			}
			customExtra := record{}
			for k, v := range obj(disk["custom"]) {
				if k != "common" && k != "need_reset" {
					customExtra[k] = copyValue(v)
				}
			}
			if len(customExtra) > 0 {
				r.retained["custom"] = customExtra
			}
			for k, v := range disk {
				if v != nil {
					r.state[k] = copyValue(v)
				}
			}
			r.state["currency"] = obj(r.state["currency"])
			if _, ok := disk["stamina_pool"]; !ok {
				r.state["stamina_pool"] = copyValue(defaults()["stamina_pool"])
			} else if _, ok := r.state["stamina_pool"].(map[string]any); !ok {
				r.state["stamina_pool"] = copyValue(defaults()["stamina_pool"])
			}
			if !truth(r.state["stamina_at"]) {
				r.state["stamina_at"] = r.now()
			}
			if c, ok := disk["custom"].(map[string]any); ok {
				r.state["custom"] = record{"common": copyValue(obj(c["common"])), "need_reset": copyValue(obj(c["need_reset"]))}
			}
		}
	}
	if options.ClientProfile == "bili-224" {
		if err := r.validateBili224SavedReferences(); err != nil {
			return nil, err
		}
	}
	common := obj(obj(r.state["custom"])["common"])
	for _, flag := range []string{"guide_draw", "IsSkinGuide"} {
		if options.SkipGuide {
			if _, ok := common[flag]; !ok {
				common[flag] = true
			}
		} else {
			delete(common, flag)
		}
	}
	if !options.SkipGuide && options.ReplayPrologue && r.clearPrologue() > 0 {
		r.save()
	}
	if r.validLeader(r.leaderHero(r.state["leader"])) == "" {
		lead := r.leaderBase(r.canonicalLeader())
		if lead == "" {
			lead = "LGirl"
		}
		r.state["leader"] = lead
		delete(r.state, "teams")
	}
	// B224 表中没有原型起手装备，也没有确认的等价发放规则，不能重建无效模板
	if options.ClientProfile != "bili-224" {
		r.ensureStarterEquips()
	}
	if r.err != nil {
		return nil, r.err
	}
	return r, nil
}

func (r *Runtime) now() int64 { return r.options.Now().Unix() }
func (r *Runtime) row(table string, key any) record {
	if key == nil {
		return record{}
	}
	return obj(r.tables[table][str(key)])
}
func (r *Runtime) save() {
	if r.store == nil || r.err != nil {
		return
	}
	var disk record
	_, err := r.store.Load(&disk)
	if err != nil {
		r.err = err
		return
	}
	if !hasProgress(r.state) && hasProgress(disk) {
		r.err = errors.New("拒绝用无进度状态覆盖已有进度存档")
		return
	}
	persisted := copyValue(r.state).(record)
	if cases, ok := persisted["cases"].(map[string]any); ok {
		// 多个 case 改写同一物件时按完成顺序覆盖，保存和重启不能打乱该顺序
		persisted["cases"] = orderedInitializationObject{keys: r.stateObjectKeys("cases"), values: cases}
	}
	for k, v := range r.retained {
		if k == "custom" {
			missing(ensure(persisted, "custom"), obj(v))
		} else if _, exists := persisted[k]; !exists {
			persisted[k] = copyValue(v)
		}
	}
	r.err = r.store.Save(persisted)
}
func hasProgress(s record) bool {
	return integer(or(s["level"], 1), 1) > 1 || len(obj(s["hero_state"])) > 2 || truth(obj(s["teams"])["1"]) || truth(s["points"])
}
func (r *Runtime) Snapshot() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return copyValue(r.state).(record)
}
func (r *Runtime) clearPrologue() int {
	common := obj(obj(r.state["custom"])["common"])
	story := text(r.row("BaseConfig", "Initial_Storey")["Config"])
	n := 0
	for k := range common {
		if k == "Story"+story || strings.HasPrefix(k, "StorySMSP_T00_0100_") {
			delete(common, k)
			n++
		}
	}
	return n
}

// Handle 保留 Python 的请求编号、推送顺序和命令执行期间的状态副作用
func (r *Runtime) Handle(command uint16, request any) ([]Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = nil
	payload := obj(request)
	if _, ok := payload["seq"]; ok {
		if inner, ok := payload["msg"]; ok {
			payload = obj(inner)
		}
	}
	name, ok := CommandName(command)
	if !ok {
		return nil, fmt.Errorf("不支持游戏命令：command=%d", command)
	}
	base := record{"result": 0, "cmd": name}
	out := []Message{}
	send := func(cmd uint16, p record) { out = append(out, Message{cmd, copyValue(p).(record)}) }
	pushes := func() {
		send(CommandHeroes, record{"result": 0, "heros": r.ownedHeroes()})
		send(CommandTeams, r.teams())
		w := r.worldMap()
		w["result"] = 0
		send(CommandWorldMap, w)
		send(CommandSystemUnlock, record{"result": 0, "unlock_ids": r.unlockIDs()})
		send(CommandPushReminds, record{"result": 0, "reminds": []any{}})
		send(CommandRedPoints, record{"result": 0, "red_points": record{}})
		send(CommandSummitBase, r.summit())
		send(CommandActivities, r.activities())
	}
	switch command {
	case CommandAuth:
		if r.options.ReplayPrologue {
			r.clearPrologue()
		}
		send(CommandAuth, base)
		send(CommandRoleInfo, r.roleInfo())
		send(CommandLoginInfo, record{"result": 0, "Role": r.roleInfo(), "Resource": record{"heros": r.ownedHeroes()}, "summit": r.summit()})
		pushes()
	case CommandHeartbeat:
		base["server_timestamp"] = r.options.Now().UnixMilli()
		if v, ok := payload["clientTimestamp"]; ok {
			base["clientTimestamp"] = v
		}
		send(command, base)
	case CommandRandomRoleName:
		base["role_name"] = r.randomName()
		send(command, base)
	case CommandUpdateRoleName, CommandChooseLeader:
		if command == CommandChooseLeader {
			if truth(payload["leader_id"]) {
				lid := payload["leader_id"]
				lead := r.validLeader(r.leaderHero(lid))
				if lead == "" {
					lead = r.canonicalLeader()
				}
				lb := r.leaderBase(lead)
				if lb == "" {
					lb = "LGirl"
				}
				r.state["leader"] = lb
				r.state["sex"] = 2
				if str(lid) == "LBoy" {
					r.state["sex"] = 1
				}
			}
		}
		if truth(payload["role_name"]) {
			r.state["role_name"] = copyValue(payload["role_name"])
		}
		if command == CommandChooseLeader || truth(payload["role_name"]) {
			r.save()
		}
		resp := r.roleInfo()
		resp["cmd"] = name
		send(command, resp)
		if command == CommandChooseLeader {
			pushes()
		}
	case CommandRoleBoard:
		board, err := r.roleBoard()
		if err != nil {
			return nil, err
		}
		merge(base, board)
		send(command, base)
	case CommandSaveBoardChange:
		r.saveBoardChange(payload)
		if r.err != nil {
			return nil, r.err
		}
		// 原型先推精简 1001 再回复 1006，system_opens 必须完整保留
		full := r.roleInfo()
		if r.err != nil {
			return nil, r.err
		}
		exterior := obj(full["exterior"])
		keep := record{}
		for _, key := range []string{"wife", "back_now", "mode", "leader", "system_opens"} {
			if value, ok := exterior[key]; ok {
				keep[key] = copyValue(value)
			}
		}
		if !truth(keep["system_opens"]) {
			keep = exterior
		}
		send(CommandRoleInfo, record{"result": 0, "cmd": "MSG_ROLE_INFO_RES", "exterior": keep, "level": full["level"]})
		base["board"] = r.board()
		base["board_info"] = record{"back_all": record{}, "back_new": record{}}
		send(command, base)
	case CommandFrontSettings, CommandSaveFrontSettings:
		custom := obj(r.state["custom"])
		if command == CommandSaveFrontSettings {
			if incoming, ok := payload["settings"].(map[string]any); ok {
				for _, group := range []string{"common", "need_reset"} {
					if vals, ok := incoming[group].(map[string]any); ok {
						merge(obj(custom[group]), vals)
					}
				}
				r.save()
			}
		}
		base["settings"] = copyValue(custom)
		base["version"] = 0
		send(command, base)
	case CommandDownloadReward:
		base["rewards"] = []any{}
		send(command, base)
	case CommandBagRequest:
		r.state["bag_version"] = iv(r.state["bag_version"]) + 1
		base["items"] = r.bagItems()
		base["max_count"] = 0
		base["version"] = r.state["bag_version"]
		send(command, base)
	case CommandChapters:
		w := r.worldMap()
		w["result"] = 0
		send(CommandWorldMap, w)
		merge(base, r.chapters())
		base["auto_info"] = r.patrol()
		send(command, base)
	case CommandMonthCard:
		cards, err := r.monthCards()
		if err != nil {
			return nil, err
		}
		base["month_card"] = cards
		send(command, base)
	case CommandTasks:
		tasks, err := r.taskPayload()
		if err != nil {
			return nil, err
		}
		base["tasks"] = tasks
		send(command, base)
	case CommandChatInfo:
		// Python Session.dispatch：ChatModel 直接读取 chat_info 中的两个频道字段
		base["chat_info"] = record{"world_channel_line": 1, "max_world_channel_line": 1}
		send(command, base)
	case CommandWorldMap, CommandEnterScene, CommandFinishSceneCase:
		if err := r.loadExploreTables(); err != nil {
			return nil, err
		}
		switch command {
		case CommandWorldMap:
			body, err := r.exploreWorld()
			if err != nil {
				return nil, err
			}
			merge(base, body)
		case CommandEnterScene:
			id := str(or(payload["scene_id"], r.initialSceneID()))
			units, err := r.sceneUnits(id, payload["pos"])
			if err != nil {
				return nil, err
			}
			merge(base, record{"scene_id": id, "grid_speed": 0.3, "units": units, "all_light": true, "fog_size": 0, "fog": []any{}})
			dp, err := r.exploreDP()
			if err != nil {
				return nil, err
			}
			send(CommandExploreDP, dp)
		case CommandFinishSceneCase:
			body, dp, err := r.finishSceneCase(payload)
			if err != nil {
				return nil, err
			}
			merge(base, body)
			if dp != nil {
				send(CommandExploreDP, dp)
			}
		}
		send(command, base)
	case CommandSceneMini:
		merge(base, record{"scene_id": copyValue(payload["scene_id"]), "all_light": true, "fog_size": 0, "fog": []any{}, "mini_map": record{}})
		send(command, base)
	case CommandSceneMove, CommandSceneSpeedup, CommandQuitScene:
		// Python 8946/8957/9498 明确只确认；1853 不回 pos，避免客户端将每次移动视为位置校正
		send(command, base)
	case CommandFriends, CommandRecommendFriends, CommandBlacklist, CommandChatList, CommandPVPRoom,
		CommandGuild, CommandTowerInfo, CommandResourceMap, CommandDrawSystem,
		CommandRealArenaBasic, CommandRealArenaInfo, CommandArenaBasic, CommandArenaInfo,
		CommandFairArenaInfo, CommandActivities, CommandClientTracking, CommandFightTogetherInfo, CommandRogueInfo,
		CommandStaminaRecovery, CommandChapterAutoInfo, CommandMails, CommandCheckMonthCard,
		CommandFaceNotices, CommandSummitRequest, CommandRecommendGuilds, CommandGuildInvites,
		CommandSearchGuild, CommandLastGuilds:
		body, err := r.initializationQuery(command, payload)
		if err != nil {
			return nil, fmt.Errorf("初始化查询 command=%d: %w", command, err)
		}
		merge(base, body)
		send(command, base)
	default:
		return nil, fmt.Errorf("不支持游戏命令：command=%d", command)
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.options.ClientProfile == "bili-224" {
		for _, message := range out {
			if err := r.validateBili224Message(message); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (r *Runtime) randomName() string {
	out := ""
	for _, table := range []string{"NameFront", "NameMiddle", "NameLast"} {
		pool := []string{}
		for _, k := range r.keys[table] {
			row := r.row(table, k)
			if k != "_subJsonCount" && truth(row["Content"]) {
				pool = append(pool, str(row["Content"]))
			}
		}
		if len(pool) == 0 && table == "NameMiddle" {
			pool = append(pool, "")
		}
		if len(pool) == 0 {
			r.err = fmt.Errorf("名字配置表 %s 没有可抽取内容", table)
			return ""
		}
		i := r.options.Choose(len(pool))
		if i < 0 || i >= len(pool) {
			r.err = fmt.Errorf("随机索引超出名字池范围: %d", i)
			return ""
		}
		out += pool[i]
	}
	return out
}
func sortedKeys(m record) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
