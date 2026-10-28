package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/internal/assets"
	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
)

func TestLoginInitializationMatchesPythonAndApprovedB224Fields(t *testing.T) {
	testPathsMatchPython(t, "testdata/login_init_python.json", nil)
}

func initQueryRuntime(t *testing.T) *Runtime {
	t.Helper()
	source := goldenTables{Tables: map[string]record{}, Keys: map[string][]string{}}
	for _, name := range requiredTables {
		source.Tables[name] = record{}
	}
	for _, entry := range []struct{ table, id string }{
		{"ArenaBase", "Arena_Season1"}, {"ArenaRealTimeBase", "ArenaRealTime_1"}, {"ArenaFairPlayBase", "ArenaFairPlay_Week1"},
	} {
		source.Tables[entry.table] = record{entry.id: record{"Id": entry.id}}
		source.Keys[entry.table] = []string{entry.id}
	}
	source.Tables["DrawCard"] = record{
		"AAA_Limited": record{"Type": "limited", "CountType": "Limit", "Coupon": []any{"Ticket"}, "CouponCost": []any{1}, "Number": []any{1}},
		"ZZZ_Normal":  record{"Type": "NORMAL", "CountType": "New", "Coupon": []any{"Ticket"}, "CouponCost": []any{1}, "Number": []any{1}},
	}
	source.Keys["DrawCard"] = []string{"AAA_Limited", "ZZZ_Normal"}
	r, err := New(source, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestInitializationDoesNotHideNonemptyOrCorruptState(t *testing.T) {
	for _, test := range []struct {
		command uint16
		key     string
	}{
		{CommandFriends, "friends"}, {CommandBlacklist, "blacklist"}, {CommandChatList, "contactList"},
		{CommandPVPRoom, "room_info"}, {CommandGuild, "club_info"}, {CommandTowerInfo, "tower_points"},
		{CommandArenaBasic, "arena"}, {CommandRealArenaInfo, "real_arena"}, {CommandFairArenaInfo, "fair_arena"},
	} {
		t.Run(test.key, func(t *testing.T) {
			for _, value := range []any{true, record{"existing": 1}, "broken"} {
				r := initQueryRuntime(t)
				r.state[test.key] = value
				before := r.Snapshot()
				messages, err := r.Handle(test.command, record{})
				if err == nil || len(messages) != 0 || !strings.Contains(err.Error(), test.key) {
					t.Fatalf("已有或损坏状态被掩盖：command=%d messages=%v err=%v", test.command, messages, err)
				}
				if !reflect.DeepEqual(before, r.Snapshot()) {
					t.Fatal("失败查询改写了存档状态")
				}
			}
		})
	}
}

func TestRealArenaLoadingInfoUsesSameStateAsBasic(t *testing.T) {
	r := initQueryRuntime(t)
	basic, err := r.Handle(CommandRealArenaBasic, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := r.Handle(CommandRealArenaInfo, record{"type": 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(info) != 1 || info[0].Command != 1702 || !reflect.DeepEqual(basic[0].Payload["arena"], info[0].Payload["arena"]) {
		t.Fatalf("1702 缺少与 1701 一致的 arena_info：%v", info)
	}
	for _, value := range []any{1, "invalid", record{}} {
		messages, err := r.Handle(CommandRealArenaInfo, record{"type": value})
		if err == nil || len(messages) != 0 {
			t.Fatalf("未实现的实时竞技场请求仍成功：messages=%v err=%v", messages, err)
		}
	}
}

func TestInitializationPreservesArenaConfigErrors(t *testing.T) {
	want := errors.New("synthetic arena table failure")
	for _, command := range []uint16{CommandArenaBasic, CommandArenaInfo, CommandRealArenaBasic, CommandRealArenaInfo, CommandFairArenaInfo} {
		for _, source := range []hero.TableSource{
			initFailureSource{err: want}, goldenTables{Tables: map[string]record{}},
			goldenTables{Tables: map[string]record{"ArenaBase": {}, "ArenaRealTimeBase": {}, "ArenaFairPlayBase": {}}},
		} {
			r := initQueryRuntime(t)
			r.catalog = hero.NewCatalog(source)
			messages, err := r.Handle(command, nil)
			if err == nil || len(messages) != 0 || !(errors.Is(err, want) || errors.Is(err, hero.ErrTableNotFound) || errors.Is(err, hero.ErrRowNotFound)) {
				t.Fatalf("赛季配置错误被当成成功：command=%d messages=%v err=%v", command, messages, err)
			}
		}
	}
}

type initFailureSource struct{ err error }

func (s initFailureSource) LoadTable(string) (map[string]any, error) { return nil, s.err }

func TestInitializationRejectsMissingReferencesAndCorruptCounters(t *testing.T) {
	for _, run := range []func(*Runtime) uint16{
		func(r *Runtime) uint16 {
			r.state["gacha"] = "broken"
			return CommandDrawSystem
		},
		func(r *Runtime) uint16 {
			r.state["gacha"] = record{"total": "9223372036854775808"}
			return CommandDrawSystem
		},
		func(r *Runtime) uint16 {
			r.tables["DrawCard"]["AAA_Limited"].(record)["Number"] = []any{record{}}
			return CommandDrawSystem
		},
		func(r *Runtime) uint16 {
			r.tables["DrawCard"] = record{}
			r.keys["DrawCard"] = nil
			return CommandDrawSystem
		},
		func(r *Runtime) uint16 {
			r.state["resource_pass"] = false
			return CommandResourceMap
		},
		func(r *Runtime) uint16 {
			r.tables["ResourcesMap"] = record{"Trial": record{"ResourcesPoint": "not-an-array"}}
			r.keys["ResourcesMap"] = []string{"Trial"}
			return CommandResourceMap
		},
		func(r *Runtime) uint16 {
			r.tables["ResourcesMap"] = record{"Trial": record{"ResourcesPoint": []any{"MissingPoint"}}}
			r.keys["ResourcesMap"] = []string{"Trial"}
			return CommandResourceMap
		},
		func(r *Runtime) uint16 {
			r.state["points"] = record{"Tower_01": record{"pass": true}}
			return CommandTowerInfo
		},
	} {
		r := initQueryRuntime(t)
		command := run(r)
		messages, err := r.Handle(command, nil)
		if err == nil || len(messages) != 0 {
			t.Fatalf("关键输入错误仍成功：command=%d messages=%v err=%v", command, messages, err)
		}
	}
}

func TestDrawPoolOrderSurvivesJSONBridge(t *testing.T) {
	r := initQueryRuntime(t)
	messages, err := r.Handle(CommandDrawSystem, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(messages[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(data), `"id":"ZZZ_Normal"`) > strings.Index(string(data), `"id":"AAA_Limited"`) {
		t.Fatalf("卡池顺序丢失：%s", data)
	}
	var decoded record
	if err := json.Unmarshal(data, &decoded); err != nil || len(obj(decoded["card_pool"])) != 2 {
		t.Fatalf("卡池没有编码为对象：%s err=%v", data, err)
	}
}

// TestInitializationLocalAssets 只读本地配置并使用新内存状态，不读取运行配置的 save_dir
func TestInitializationLocalAssets(t *testing.T) {
	configPath := os.Getenv("GXJY_INIT_ASSET_CONFIG")
	if configPath == "" {
		t.Skip("显式指定本地配置后验证查询引用")
	}
	cfg, err := config.LoadRuntime(configPath)
	if err != nil {
		t.Fatal(err)
	}
	key, err := config.ReadAssetKey(cfg.AssetKeyEnv, cfg.AssetKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	variants := map[string][2]string{"configured": {cfg.AssetRoot, cfg.BundlePath}}
	if root := os.Getenv("GXJY_INIT_B224_ROOT"); root != "" {
		variants["bili-224"] = [2]string{root, filepath.Join(root, "resources", "config.json")}
	}
	for name, paths := range variants {
		t.Run(name, func(t *testing.T) {
			r, err := New(assets.NewTableLoader(paths[0], paths[1], key), nil, Options{})
			if err != nil {
				t.Fatal(err)
			}
			for _, command := range []uint16{
				CommandFriends, CommandBlacklist, CommandChatList, CommandPVPRoom, CommandGuild, CommandTowerInfo,
				CommandResourceMap, CommandDrawSystem, CommandRealArenaBasic, CommandRealArenaInfo,
				CommandArenaBasic, CommandArenaInfo, CommandFairArenaInfo, CommandActivities, CommandClientTracking,
			} {
				t.Run(str(command), func(t *testing.T) {
					messages, err := r.Handle(command, record{})
					if err != nil || len(messages) != 1 {
						t.Fatalf("本地配置查询失败：command=%d messages=%d err=%v", command, len(messages), err)
					}
				})
			}
		})
	}
}
