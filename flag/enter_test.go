package flag

import (
	stdflag "flag"
	"io"
	"path/filepath"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/GXJY-Project/gxjy-server/models"
)

func TestParseAndApplyOnlyExplicitOverrides(t *testing.T) {
	flags := stdflag.NewFlagSet("test", stdflag.ContinueOnError)
	var options Options
	bind(flags, &options)
	err := flags.Parse([]string{"-f", "config with spaces.json", "-db", "--http-port=0", "--asset-root=", "--client-profile=bili-224"})
	if err != nil {
		t.Fatal(err)
	}
	if options.File != "config with spaces.json" || !options.DB || options.Version {
		t.Fatalf("参数结果：%+v", options)
	}
	cfg := config.DefaultRuntime()
	cfg.HTTPPort, cfg.WSPort = 9090, 9091
	cfg.AssetRoot, cfg.RSAPEM = "loaded-assets", "loaded-key"
	apply(flags, options, &cfg)
	if cfg.HTTPPort != 0 || cfg.AssetRoot != "" || cfg.ClientProfile != "bili-224" {
		t.Fatalf("显式参数未覆盖配置：%+v", cfg)
	}
	if cfg.WSPort != 9091 || cfg.RSAPEM != "loaded-key" {
		t.Fatal("未指定的参数覆盖了文件配置")
	}
}

func TestConfigAliasAndVersion(t *testing.T) {
	flags := stdflag.NewFlagSet("test", stdflag.ContinueOnError)
	var options Options
	bind(flags, &options)
	if err := flags.Parse([]string{"--config=alternate.json", "-v"}); err != nil {
		t.Fatal(err)
	}
	if options.File != "alternate.json" || !options.Version {
		t.Fatalf("参数结果：%+v", options)
	}
}

func TestRunMigratesOnlyWhenRequested(t *testing.T) {
	previousOptions, previousDB, previousLogger := flagOptions, global.DB, global.Logger
	t.Cleanup(func() {
		flagOptions, global.DB, global.Logger = previousOptions, previousDB, previousLogger
	})
	flagOptions = Options{}
	global.DB = nil
	if done, err := Run(); done || err != nil {
		t.Fatalf("普通启动执行了迁移：done=%t err=%v", done, err)
	}
	cfg := config.DefaultRuntime().Database
	cfg.DSN = filepath.Join(t.TempDir(), "migrate.sqlite")
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
	global.DB = db
	config.InitLog()
	global.Logger.SetOutput(io.Discard)
	flagOptions.DB = true
	if done, err := Run(); !done || err != nil {
		t.Fatalf("迁移结果：done=%t err=%v", done, err)
	}
	if !db.Migrator().HasTable(&models.Player{}) {
		t.Fatal("迁移未创建玩家表")
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	if done, err := Run(); !done || err == nil {
		t.Fatalf("迁移失败被当成成功：done=%t err=%v", done, err)
	}
}
