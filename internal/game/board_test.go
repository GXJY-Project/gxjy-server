package game

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestBoardMatchesPython(t *testing.T) {
	testPathsMatchPython(t, "testdata/board_python.json", nil)
}

func TestBoardDoesNotHideTableOrSaveFailure(t *testing.T) {
	r := initQueryRuntime(t)
	want := errors.New("synthetic board table failure")
	r.catalog = hero.NewCatalog(initFailureSource{err: want})
	if messages, err := r.Handle(CommandRoleBoard, nil); !errors.Is(err, want) || len(messages) != 0 {
		t.Fatalf("看板配置失败仍回成功：messages=%v err=%v", messages, err)
	}
	store := &taskFailingStore{JSONStore: storage.NewJSONStore(t.TempDir())}
	r.store = store
	want = errors.New("synthetic board save failure")
	store.err = want
	if messages, err := r.Handle(CommandSaveBoardChange, record{"mode": 0}); !errors.Is(err, want) || len(messages) != 0 {
		t.Fatalf("保存失败仍推送角色或回成功：messages=%v err=%v", messages, err)
	}
}

func TestFightTogetherInitialStateIsScopedAndReadOnly(t *testing.T) {
	r := initQueryRuntime(t)
	if messages, err := r.Handle(CommandFightTogetherInfo, nil); err == nil || len(messages) != 0 {
		t.Fatal("初始空态适配泄露到非 B224 路径")
	}
	r.options.ClientProfile = "bili-224"
	before := r.Snapshot()
	r.store = &taskFailingStore{JSONStore: storage.NewJSONStore(t.TempDir()), err: errors.New("只读查询不应保存")}
	messages, err := r.Handle(CommandFightTogetherInfo, record{})
	if err != nil || len(messages) != 1 {
		t.Fatalf("B224 初始查询失败：messages=%v err=%v", messages, err)
	}
	var expected struct{ Payload record }
	data, err := os.ReadFile("testdata/bili224_ft_initial.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	assertJSON(t, "B224 协战初始响应", messages[0].Payload, expected.Payload)
	if !reflect.DeepEqual(before, r.Snapshot()) {
		t.Fatal("初始查询改变了玩家状态")
	}
	for _, key := range []string{"fight_together", "playerFTInfo", "bossList", "actBossIds"} {
		for _, value := range []any{true, "corrupt", record{"existing": 1}, []any{"existing"}} {
			r.state[key] = value
			before := r.Snapshot()
			messages, err := r.Handle(CommandFightTogetherInfo, record{})
			if err == nil || len(messages) != 0 || !reflect.DeepEqual(before, r.Snapshot()) {
				t.Fatalf("已有或损坏协战状态被隐藏：key=%s err=%v", key, err)
			}
		}
		delete(r.state, key)
	}
}

func TestRogueInitialStateIsScopedAndReadOnly(t *testing.T) {
	r := initQueryRuntime(t)
	if messages, err := r.Handle(CommandRogueInfo, nil); err == nil || len(messages) != 0 {
		t.Fatal("Rogue 初态适配泄露到非 B224 路径")
	}
	r.options.ClientProfile = "bili-224"
	r.store = &taskFailingStore{JSONStore: storage.NewJSONStore(t.TempDir()), err: errors.New("只读查询不应保存")}
	before := r.Snapshot()
	messages, err := r.Handle(CommandRogueInfo, record{})
	if err != nil || len(messages) != 1 {
		t.Fatalf("Rogue 初态查询失败：%v", err)
	}
	var expected struct{ Payload record }
	data, err := os.ReadFile("testdata/bili224_rogue_initial.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	assertJSON(t, "Rogue 初始响应", messages[0].Payload, expected.Payload)
	if !reflect.DeepEqual(before, r.Snapshot()) {
		t.Fatal("Rogue 初态查询修改了玩家状态")
	}
	for _, key := range []string{"rogue", "rogue_info", "roguelike"} {
		for _, value := range []any{true, "corrupt", record{"existing": 1}, []any{"existing"}} {
			r.state[key] = value
			before := r.Snapshot()
			messages, err := r.Handle(CommandRogueInfo, record{})
			if err == nil || len(messages) != 0 || !reflect.DeepEqual(before, r.Snapshot()) {
				t.Fatalf("已有 Rogue 进度被隐藏：key=%s err=%v", key, err)
			}
		}
		delete(r.state, key)
	}
}
