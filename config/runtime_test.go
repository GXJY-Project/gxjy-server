package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeRuntime(t *testing.T) {
	baseDir := filepath.Clean("C:/gxjy/gxjy-server")
	config, err := decodeRuntime([]byte(`{
		"http_port": 9090,
		"client_profile": "bili-224",
		"save_dir": "data/save",
		"rsa_pem": "data/keys/gate.pem",
		"rsa_json": "data/keys/server_rsa.json"
	}`), baseDir)
	if err != nil {
		t.Fatal(err)
	}

	if config.HTTPHost != "127.0.0.1" || config.HTTPPort != 9090 {
		t.Fatalf("HTTP 配置错误：%s:%d", config.HTTPHost, config.HTTPPort)
	}
	if config.ClientProfile != "bili-224" {
		t.Fatal("客户端配置契约未传递")
	}
	wantPEM := filepath.Clean(filepath.Join(baseDir, "data/keys/gate.pem"))
	if config.RSAPEM != wantPEM {
		t.Fatalf("RSA PEM 路径 = %q, want %q", config.RSAPEM, wantPEM)
	}
	wantSaveDir := filepath.Clean(filepath.Join(baseDir, "data/save"))
	if config.SaveDir != wantSaveDir {
		t.Fatalf("存档目录 = %q, want %q", config.SaveDir, wantSaveDir)
	}
	if config.AssetKeyEnv != "GXJY_ASSET_KEY" {
		t.Fatalf("资源密钥环境变量 = %q", config.AssetKeyEnv)
	}
}

func TestDecodeRuntimeRejectsUnknownField(t *testing.T) {
	_, err := decodeRuntime([]byte(`{"unknown": true}`), ".")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("未知字段错误 = %v", err)
	}
}

func TestDatabaseConfigPaths(t *testing.T) {
	baseDir := t.TempDir()
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"默认 SQLite", `{}`, filepath.Join(baseDir, "data/db/gxjy.sqlite")},
		{"SQLite 路径", `{"database":{"driver":"sqlite","dsn":"test/player.sqlite"}}`, filepath.Join(baseDir, "test/player.sqlite")},
		{"SQLite 内存", `{"database":{"driver":"sqlite","dsn":":memory:"}}`, ":memory:"},
		{"PostgreSQL DSN", `{"database":{"driver":"postgres","dsn":"${GXJY_DATABASE_DSN}"}}`, "${GXJY_DATABASE_DSN}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := decodeRuntime([]byte(tc.input), baseDir)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Database.DSN != tc.want {
				t.Fatalf("DSN = %q, want %q", cfg.Database.DSN, tc.want)
			}
		})
	}
}
