package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultFileName    = "save_localplayer.json"
	defaultBackupLimit = 10
)

// JSONStore 负责单个 JSON 存档的读取、原子写入和备份轮转。
type JSONStore struct {
	dir         string
	fileName    string
	backupLimit int
	now         func() time.Time
}

// NewJSONStore 创建玩家存档。目录为空时关闭持久化。
func NewJSONStore(dir string) *JSONStore {
	return &JSONStore{
		dir:         dir,
		fileName:    defaultFileName,
		backupLimit: defaultBackupLimit,
		now:         time.Now,
	}
}

// Enabled 表示存档是否启用。
func (store *JSONStore) Enabled() bool {
	return store != nil && store.dir != ""
}

// Path 返回主存档路径。
func (store *JSONStore) Path() string {
	if !store.Enabled() {
		return ""
	}
	return filepath.Join(store.dir, store.fileName)
}

// Load 读取主存档。文件不存在时返回 found=false。
func (store *JSONStore) Load(target any) (found bool, err error) {
	if !store.Enabled() {
		return false, nil
	}
	data, err := os.ReadFile(store.Path())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取玩家存档：%w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return false, fmt.Errorf("解析玩家存档：%w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false, errors.New("玩家存档只能包含一个 JSON 对象")
	}
	return true, nil
}

// Save 先写临时文件，再替换主存档，并保留上一版和十份快照。
func (store *JSONStore) Save(value any) error {
	if !store.Enabled() {
		return nil
	}
	blob, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("编码玩家存档：%w", err)
	}
	blob = append(blob, '\n')

	if err := os.MkdirAll(store.dir, 0o700); err != nil {
		return fmt.Errorf("创建存档目录：%w", err)
	}
	path := store.Path()
	current, readErr := os.ReadFile(path)
	if readErr == nil && bytes.Equal(current, blob) {
		return nil
	}
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("读取旧存档：%w", readErr)
	}

	temporary, err := os.CreateTemp(store.dir, ".player-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时存档：%w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("设置临时存档权限：%w", err)
	}
	if _, err := temporary.Write(blob); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("写入临时存档：%w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("同步临时存档：%w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭临时存档：%w", err)
	}

	if readErr == nil {
		if err := os.WriteFile(filepath.Join(store.dir, "save_localplayer.prev.json"), current, 0o600); err != nil {
			return fmt.Errorf("备份上一版存档：%w", err)
		}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("替换玩家存档：%w", err)
	}
	if err := store.writeBackup(blob); err != nil {
		return err
	}
	return nil
}

func (store *JSONStore) writeBackup(blob []byte) error {
	stamp := store.now().Format("20060102-150405.000000000")
	backupPath := filepath.Join(store.dir, "save_backup_"+stamp+".json")
	if err := os.WriteFile(backupPath, blob, 0o600); err != nil {
		return fmt.Errorf("写入存档快照：%w", err)
	}

	entries, err := os.ReadDir(store.dir)
	if err != nil {
		return fmt.Errorf("读取存档快照：%w", err)
	}
	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "save_backup_") && strings.HasSuffix(entry.Name(), ".json") {
			backups = append(backups, entry.Name())
		}
	}
	sort.Strings(backups)
	for _, name := range backups[:max(0, len(backups)-store.backupLimit)] {
		if err := os.Remove(filepath.Join(store.dir, name)); err != nil {
			return fmt.Errorf("删除旧存档快照：%w", err)
		}
	}
	return nil
}
