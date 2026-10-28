package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/internal/assets"
	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestBili224DoesNotRegeneratePrototypeEquipment(t *testing.T) {
	r, err := New(&taskQuerySource{}, nil, Options{ClientProfile: "bili-224"})
	if err != nil {
		t.Fatal(err)
	}
	if len(obj(r.Snapshot()["equip_state"])) != 0 {
		t.Fatal("B224 重新生成了没有发放依据的旧模板装备")
	}
	if _, err := New(&taskQuerySource{}, nil, Options{ClientProfile: "unknown"}); err == nil {
		t.Fatal("接受了未知客户端契约")
	}
}

func TestBili224RejectsOldOrMalformedSavedRecordsWithoutSaving(t *testing.T) {
	for _, input := range []string{
		`{"equip_state":{"instance":true}}`,
		`{"equip_state":{"instance":{"template_id":"Legacy"}}}`,
		`{"tasks":{"Legacy":{"status":2}}}`,
		`{"hero_state":{"MissingHero":{"level":1}}}`,
		`{"bag_items":{"MissingItem":1}}`,
		`{"bag_items":{"MissingItem":"overflow"}}`,
	} {
		store := storage.NewJSONStore(t.TempDir())
		if err := os.WriteFile(store.Path(), []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := New(&taskQuerySource{}, store, Options{ClientProfile: "bili-224"}); err == nil {
			t.Fatal("旧版或损坏记录被接受")
		}
		after, err := os.ReadFile(store.Path())
		if err != nil || !bytes.Equal(after, []byte(input)) {
			t.Fatal("校验失败改写了原存档")
		}
	}
}

func TestBili224RejectsInvalidOutgoingConfigReferences(t *testing.T) {
	r := initQueryRuntime(t)
	for _, message := range []Message{
		{CommandHeroes, record{"heros": record{"MissingHero": record{}}}},
		{CommandBagRequest, record{"items": record{"instance": record{"item_type": "Equip", "template_id": "Equip_S3_1"}}}},
		{CommandTasks, record{"tasks": record{"Daily": record{"MissingTask": record{}}}}},
		{CommandDrawSystem, record{"card_pool": orderedInitializationObject{keys: []string{"MissingPool"}}}},
	} {
		err := r.validateBili224Message(message)
		if !errors.Is(err, hero.ErrRowNotFound) && !errors.Is(err, hero.ErrTableNotFound) {
			t.Fatalf("command=%d 的无效引用未保留配置错误：%v", message.Command, err)
		}
	}
	// 客户端实测 H002 的旧版被动技能在 B224 SkillBase 不存在
	r.tables["HeroBase"]["H002"] = record{}
	r.tables["HeroQuality"]["Quality"] = record{}
	r.tables["HeroStar"]["Star"] = record{}
	message := Message{CommandHeroes, record{"heros": record{"H002": record{
		"base_attr": record{"quality_id": "Quality", "star": record{"star_id": "Star"}},
		"skills":    record{"hero_skills": record{"HeroSkill_Passive_H002": record{"id": "HeroSkill_Passive_H002"}}},
	}}}}
	if err := r.validateBili224Message(message); !errors.Is(err, hero.ErrRowNotFound) {
		t.Fatalf("无效技能未阻止发送：%v", err)
	}
	obj(obj(obj(message.Payload["heros"])["H002"])["skills"])["hero_skills"] = record{}
	if err := r.validateBili224Message(message); err != nil {
		t.Fatalf("无技能的有效引用被拒绝：%v", err)
	}
}

// TestBili224LocalRepairedInitialization 显式启用后只读本地存档，在临时副本执行真实初始化
// 来源：22:06:57 实测请求序列和四处 B224 配置断言；不把原始玩家回包写入测试输出
func TestBili224LocalRepairedInitialization(t *testing.T) {
	configPath := os.Getenv("GXJY_B224_VERIFY_LOCAL")
	if configPath == "" {
		t.Skip("显式指定本地 B224 配置后验证修复存档的临时副本")
	}
	cfg, err := config.LoadRuntime(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientProfile != "bili-224" {
		t.Fatal("本地配置没有选择 B224")
	}
	key, err := config.ReadAssetKey(cfg.AssetKeyEnv, cfg.AssetKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	originalPath := storage.NewJSONStore(cfg.SaveDir).Path()
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		after, err := os.ReadFile(originalPath)
		if err != nil || !bytes.Equal(original, after) {
			t.Error("验证期间正常存档变化")
		}
	}()
	store := storage.NewJSONStore(filepath.Join(t.TempDir(), "player"))
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), original, 0600); err != nil {
		t.Fatal(err)
	}
	loader := assets.NewTableLoader(cfg.AssetRoot, cfg.BundlePath, key)
	r, err := New(loader, store, Options{ClientProfile: cfg.ClientProfile})
	if err != nil {
		t.Fatal(err)
	}
	before := copyValue(r.state["_bili224_quarantine"])
	for _, command := range []uint16{101, 1852, 1851, 1853, 1854, 1863, 1864, 1016, 1011, 1670, 1552, 1254, 10115, 10101, 10124, 10129, 1601, 1301, 10115, 2601, 2551, 1005, 1006, 1002, 1307, 1502, 1955, 2401, 1752, 10102, 10152, 10157, 10167, 10175, 901, 2016} {
		t.Run(str(command), func(t *testing.T) {
			messages, err := r.Handle(command, record{})
			if err != nil {
				t.Fatal(err)
			}
			if len(messages) == 0 {
				t.Fatal("初始化缺少响应")
			}
			if command == CommandEnterScene {
				if len(messages) != 2 || messages[0].Command != CommandExploreDP {
					t.Fatal("进场前缺少探索度推送")
				}
				units := obj(messages[1].Payload["units"])
				if len(units) < 2 || obj(units["1"])["type"] != "PLAYER" {
					t.Fatal("真实 B224 初始场景缺少玩家或探索物件")
				}
			}
		})
	}
	if len(obj(r.state["equip_state"])) != 0 {
		t.Fatal("隔离后旧模板装备重新生成")
	}
	if err := store.Save(r.Snapshot()); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(loader, store, r.options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, reloaded.state["_bili224_quarantine"]) {
		t.Fatal("重启后隔离区丢失")
	}
}
