package game

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestExploreMatchesPython(t *testing.T) {
	testPathsMatchPython(t, "testdata/explore_python.json", nil)
}

func exploreRuntime(t *testing.T) *Runtime {
	t.Helper()
	data, err := os.ReadFile("testdata/explore_python.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Tables map[string]record   `json:"tables"`
		Keys   map[string][]string `json:"table_keys"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	r, err := New(goldenTables{Tables: fixture.Tables, Keys: fixture.Keys}, storage.NewJSONStore(t.TempDir()), Options{Now: func() time.Time { return time.Unix(1800000000, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestExploreDoesNotHideCriticalFailure(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		r := exploreRuntime(t)
		want := errors.New("synthetic explore table failure")
		r.catalog = hero.NewCatalog(initFailureSource{err: want})
		if messages, err := r.Handle(CommandEnterScene, nil); !errors.Is(err, want) || len(messages) != 0 {
			t.Fatalf("配置失败仍推送或回成功：%v", err)
		}
	})
	t.Run("save", func(t *testing.T) {
		r := exploreRuntime(t)
		want := errors.New("synthetic explore save failure")
		r.store = &taskFailingStore{JSONStore: storage.NewJSONStore(t.TempDir()), err: want}
		if messages, err := r.Handle(CommandFinishSceneCase, record{"case_do_id": "OpenChest"}); !errors.Is(err, want) || len(messages) != 0 {
			t.Fatalf("保存失败仍发奖励或成功响应：%v", err)
		}
	})
	for _, request := range []struct {
		command uint16
		payload record
	}{
		{CommandEnterScene, record{"scene_id": "MissingScene"}},
		{CommandFinishSceneCase, record{"case_do_id": "MissingCase"}},
		{CommandFinishSceneCase, record{}},
	} {
		r := exploreRuntime(t)
		if messages, err := r.Handle(request.command, request.payload); err == nil || len(messages) != 0 {
			t.Fatalf("缺失场景或事件仍回成功：cmd=%d", request.command)
		}
	}
}

func TestExploreCaseOrderSurvivesRestart(t *testing.T) {
	r := exploreRuntime(t)
	for _, id := range []string{"SwitchB", "SwitchA"} {
		if _, err := r.Handle(CommandFinishSceneCase, record{"case_do_id": id}); err != nil {
			t.Fatal(err)
		}
	}
	assertState := func(runtime *Runtime) {
		t.Helper()
		messages, err := runtime.Handle(CommandEnterScene, nil)
		if err != nil {
			t.Fatal(err)
		}
		detail := obj(obj(obj(messages[1].Payload["units"])["Switch"])["unit_detail"])
		if iv(detail["state"]) != 1 {
			t.Fatal("同物件末次状态未遵循存档事件顺序")
		}
	}
	assertState(r)
	reloaded, err := New(goldenTables{Tables: r.tables, Keys: r.keys}, r.store, r.options)
	if err != nil {
		t.Fatal(err)
	}
	assertState(reloaded)
}
