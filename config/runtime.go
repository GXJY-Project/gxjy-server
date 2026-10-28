package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AppConfig 保存启动时加载的运行配置
var AppConfig Runtime

// InitConfig 加载运行配置，相对路径仍按配置文件目录解析
func InitConfig(path string) error {
	loaded, err := LoadRuntime(path)
	if err != nil {
		return err
	}
	AppConfig = loaded
	return nil
}

// Runtime 保存服务启动所需的配置。
type Runtime struct {
	HTTPHost      string   `json:"http_host"`
	HTTPPort      int      `json:"http_port"`
	WSHost        string   `json:"ws_host"`
	WSPort        int      `json:"ws_port"`
	GateHost      string   `json:"gate_host"`
	SaveDir       string   `json:"save_dir"`
	AssetRoot     string   `json:"asset_root"`
	BundlePath    string   `json:"bundle_config"`
	AssetKeyEnv   string   `json:"asset_key_env"`
	AssetKeyFile  string   `json:"asset_key_file"`
	RSAPEM        string   `json:"rsa_pem"`
	RSAJSON       string   `json:"rsa_json"`
	ProtocolMap   string   `json:"protocol_map"`
	ClientProfile string   `json:"client_profile"`
	Database      Database `json:"database"`
}

// DefaultRuntime 返回不依赖原版资源的默认配置。
func DefaultRuntime() Runtime {
	return Runtime{
		HTTPHost:     "127.0.0.1",
		HTTPPort:     8080,
		WSHost:       "127.0.0.1",
		WSPort:       8765,
		GateHost:     "127.0.0.1",
		SaveDir:      "data/save",
		AssetKeyEnv:  "GXJY_ASSET_KEY",
		AssetKeyFile: "data/keys/asset.key",
		Database: Database{
			Driver:       "sqlite",
			DSN:          "data/db/gxjy.sqlite",
			MaxIdleConns: 1,
			MaxOpenConns: 1,
		},
	}
}

// LoadRuntime 读取 JSON 配置，并按配置文件目录解析相对路径。
func LoadRuntime(path string) (Runtime, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Runtime{}, fmt.Errorf("读取运行配置：%w", err)
	}
	baseDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return Runtime{}, fmt.Errorf("解析配置目录：%w", err)
	}
	return decodeRuntime(data, baseDir)
}

func decodeRuntime(data []byte, baseDir string) (Runtime, error) {
	result := DefaultRuntime()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Runtime{}, fmt.Errorf("解析运行配置：%w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Runtime{}, errors.New("运行配置只能包含一个 JSON 对象")
	}

	resolveRuntimePaths(&result, baseDir)
	return result, nil
}

func resolveRuntimePaths(config *Runtime, baseDir string) {
	paths := []*string{
		&config.SaveDir,
		&config.AssetRoot,
		&config.BundlePath,
		&config.AssetKeyFile,
		&config.RSAPEM,
		&config.RSAJSON,
		&config.ProtocolMap,
	}
	if config.Database.Driver == "sqlite" {
		config.Database.DSN = os.ExpandEnv(config.Database.DSN)
		if config.Database.DSN != ":memory:" {
			paths = append(paths, &config.Database.DSN)
		}
	}
	for _, current := range paths {
		if *current == "" || filepath.IsAbs(*current) {
			continue
		}
		*current = filepath.Clean(filepath.Join(baseDir, *current))
	}
}
