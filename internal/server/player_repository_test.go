package server

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
	"github.com/GXJY-Project/gxjy-server/models"
	playerrepo "github.com/GXJY-Project/gxjy-server/repositories/player"
)

func TestRoleUpdatePersistsThroughGORM(t *testing.T) {
	cfg := config.DefaultRuntime().Database
	cfg.DSN = filepath.Join(t.TempDir(), "players.sqlite")
	db, err := config.OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&models.Player{}); err != nil {
		t.Fatal(err)
	}
	repo := playerrepo.NewRepo(db)
	store := storage.NewPlayerStore(repo, 10001)
	if err := store.Save(json.RawMessage(`{"role_id":10001,"role_name":"初始玩家","exp":9007199254740993}`)); err != nil {
		t.Fatal(err)
	}
	state, err := newBaselineState(t, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.updateRole(commandUpdateRoleName, object(field("role_name", protocol.String("新玩家")))); err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if found, err := store.Load(&saved); err != nil || !found {
		t.Fatalf("读取数据库状态：found=%t err=%v", found, err)
	}
	if saved["role_name"] != "新玩家" || saved["exp"] != json.Number("9007199254740993") {
		t.Fatalf("保存结果错误：name=%v exp=%v", saved["role_name"], saved["exp"])
	}
	restarted, err := newBaselineState(t, storage.NewPlayerStore(playerrepo.NewRepo(db), 10001))
	if err != nil {
		t.Fatal(err)
	}
	if got := restarted.runtime.Snapshot()["role_name"]; got != "新玩家" {
		t.Fatalf("重新加载后角色名 = %v", got)
	}
}
