package player_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/models"
	playerrepo "github.com/GXJY-Project/gxjy-server/repositories/player"
	"gorm.io/gorm"
)

func openPlayerDB(t *testing.T, path string) *gorm.DB {
	t.Helper()
	cfg := config.DefaultRuntime().Database
	cfg.DSN = path
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
	return db
}

func TestPlayerRepoPersistsAndIsolatesRoles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "players.sqlite")
	db := openPlayerDB(t, path)
	repo := playerrepo.NewRepo(db)
	if _, err := repo.GetByRoleID(10001); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("不存在的角色：%v", err)
	}
	for _, player := range []models.Player{
		{RoleID: 10001, State: `{"level":80,"heroes":["H002"]}`},
		{RoleID: 10002, State: `{"role_name":"乙"}`},
		{RoleID: 10001, State: `{"level":0,"heroes":[],"custom":{}}`},
	} {
		if err := repo.Save(&player); err != nil {
			t.Fatal(err)
		}
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	repo = playerrepo.NewRepo(openPlayerDB(t, path))
	for roleID, want := range map[int64]string{
		10001: `{"level":0,"heroes":[],"custom":{}}`,
		10002: `{"role_name":"乙"}`,
	} {
		got, err := repo.GetByRoleID(roleID)
		if err != nil || got.RoleID != roleID || got.State != want {
			t.Fatalf("重启后角色 %d：%+v, err=%v", roleID, got, err)
		}
	}
}

func TestPlayerRepoUsesInjectedTransactionAndReportsFailure(t *testing.T) {
	db := openPlayerDB(t, filepath.Join(t.TempDir(), "players.sqlite"))
	want := errors.New("模拟业务失败")
	err := db.Transaction(func(tx *gorm.DB) error {
		repo := playerrepo.NewRepo(tx)
		if err := repo.Save(&models.Player{RoleID: 10001, State: `{}`}); err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatal(err)
	}
	repo := playerrepo.NewRepo(db)
	if _, err := repo.GetByRoleID(10001); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("失败事务未回滚：%v", err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByRoleID(10001); err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("连接失败被视为角色不存在：%v", err)
	}
	if err := repo.Save(&models.Player{RoleID: 10001, State: `{}`}); err == nil {
		t.Fatal("连接关闭后保存仍成功")
	}
}
