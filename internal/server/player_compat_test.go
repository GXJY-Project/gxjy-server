package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestRoleUpdatePreservesUnmigratedSaveFields(t *testing.T) {
	directory := t.TempDir()
	store := storage.NewJSONStore(directory)
	original := []byte(`{
  "role_id": 10001,
  "role_name": "旧玩家",
  "level": 80,
  "exp": 12,
  "sex": 2,
  "leader": "L001",
  "hero_state": {"H002": {"level": 70}},
  "bag_items": {"IC_HeroExp": 99},
  "unknown_null": null,
  "custom": {"common": {"StoryA": true}, "need_reset": {}, "future_group": {"value": 7}}
}`)
	if err := os.WriteFile(store.Path(), original, 0o600); err != nil {
		t.Fatal(err)
	}

	state, err := newBaselineState(t, store)
	if err != nil {
		t.Fatal(err)
	}
	message := object(field("role_name", protocol.String("新玩家")))
	if err := state.updateRole(commandUpdateRoleName, message); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(directory, "save_localplayer.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"hero_state", "bag_items", "custom", "unknown_null"} {
		if _, ok := saved[key]; !ok {
			t.Fatalf("存档字段 %q 丢失", key)
		}
	}
	var custom map[string]json.RawMessage
	if err := json.Unmarshal(saved["custom"], &custom); err != nil {
		t.Fatal(err)
	}
	var future map[string]json.RawMessage
	if err := json.Unmarshal(custom["future_group"], &future); err != nil {
		t.Fatal(err)
	}
	if string(future["value"]) != "7" {
		t.Fatalf("未知设置组丢失: %s", custom["future_group"])
	}
	if string(saved["role_id"]) != "10001" {
		t.Fatalf("role_id 类型或值被改写：%s", saved["role_id"])
	}
	var roleName string
	if err := json.Unmarshal(saved["role_name"], &roleName); err != nil {
		t.Fatal(err)
	}
	if roleName != "新玩家" {
		t.Fatalf("role_name = %q", roleName)
	}
}

func TestDispatchRejectsUnsupportedCommand(t *testing.T) {
	session := &gameSession{
		state: &GameState{},
		names: map[uint16]string{7777: "MSG_NOT_MIGRATED"},
	}
	err := session.dispatch(7777, protocol.Nil())
	if err == nil {
		t.Fatal("未迁移命令被当成成功")
	}
	if got := err.Error(); got != "不支持游戏命令：command=7777 name=MSG_NOT_MIGRATED" {
		t.Fatalf("错误 = %q", got)
	}
}
