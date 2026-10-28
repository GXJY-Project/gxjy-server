package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testPlayer struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

func TestJSONStoreSaveLoadAndPrevious(t *testing.T) {
	store := NewJSONStore(t.TempDir())
	if err := store.Save(testPlayer{Name: "甲", Level: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(testPlayer{Name: "乙", Level: 2}); err != nil {
		t.Fatal(err)
	}

	var loaded testPlayer
	found, err := store.Load(&loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !found || loaded.Name != "乙" || loaded.Level != 2 {
		t.Fatalf("存档结果 = %#v, found=%t", loaded, found)
	}
	previous, err := os.ReadFile(filepath.Join(store.dir, "save_localplayer.prev.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(previous) != "{\n  \"name\": \"甲\",\n  \"level\": 1\n}\n" {
		t.Fatalf("上一版存档 = %s", previous)
	}
}

func TestJSONStoreRotatesBackupsAndSkipsSameValue(t *testing.T) {
	store := NewJSONStore(t.TempDir())
	store.backupLimit = 2
	current := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		current = current.Add(time.Nanosecond)
		return current
	}

	for level := 1; level <= 3; level++ {
		if err := store.Save(testPlayer{Name: "玩家", Level: level}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Save(testPlayer{Name: "玩家", Level: 3}); err != nil {
		t.Fatal(err)
	}

	backups, err := filepath.Glob(filepath.Join(store.dir, "save_backup_*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 {
		t.Fatalf("快照数量 = %d, want 2", len(backups))
	}
}

func TestJSONStoreMissingAndInvalidFiles(t *testing.T) {
	store := NewJSONStore(t.TempDir())
	var loaded testPlayer
	found, err := store.Load(&loaded)
	if err != nil || found {
		t.Fatalf("空目录读取 = found:%t err:%v", found, err)
	}
	if err := os.WriteFile(store.Path(), []byte("{}{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(&loaded); err == nil {
		t.Fatal("连续 JSON 对象应被拒绝")
	}
}
