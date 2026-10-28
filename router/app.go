package router

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/GXJY-Project/gxjy-server/internal/assets"
	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/server"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
	"github.com/GXJY-Project/gxjy-server/repositories/player"
)

// initApp 加载游戏资源并创建共享游戏服务
func initApp() (*server.App, error) {
	cfg := config.AppConfig
	if cfg.AssetRoot == "" || cfg.BundlePath == "" {
		return nil, errors.New("登录业务需要配置 Cocos 资源目录和 Bundle")
	}
	key, err := config.ReadAssetKey(cfg.AssetKeyEnv, cfg.AssetKeyFile)
	if err != nil {
		return nil, fmt.Errorf("加载资源密钥：%w", err)
	}
	loader := assets.NewTableLoader(cfg.AssetRoot, cfg.BundlePath, key)
	// 当前服务共享一个 Runtime，角色编号沿用 game.defaults 的 10001
	store := storage.NewPlayerStore(player.NewRepo(global.DB), 10001)
	state, err := server.NewGameStateFromSource(loader, store, game.Options{
		ClientProfile:  cfg.ClientProfile,
		SkipGuide:      pythonEnvBool("GXJY_SKIP_GUIDE"),
		ReplayPrologue: pythonEnvBool("GXJY_REPLAY_PROLOGUE"),
	})
	if err != nil {
		return nil, fmt.Errorf("加载游戏状态：%w", err)
	}
	global.Logger.Infof("登录业务配置加载完成：client_profile:%s", cfg.ClientProfile)
	var rsaKey *gxjycrypto.RSAKey
	if cfg.RSAPEM != "" || cfg.RSAJSON != "" {
		var source gxjycrypto.RSAKeySource
		rsaKey, source, err = gxjycrypto.LoadRSAKeyFiles(cfg.RSAPEM, cfg.RSAJSON)
		if err != nil {
			return nil, fmt.Errorf("加载 RSA 私钥：%w", err)
		}
		global.Logger.Infof("RSA 私钥加载完成：source:%s", source)
	} else {
		global.Logger.Warn("未配置 RSA 私钥；只支持社区明文协议")
	}
	names, err := server.LoadProtocolNames(cfg.ProtocolMap)
	if err != nil {
		return nil, fmt.Errorf("读取协议表：%w", err)
	}
	return server.New(server.Config{
		HTTPAddress:  fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort),
		WSAddress:    fmt.Sprintf("%s:%d", cfg.WSHost, cfg.WSPort),
		GateHost:     cfg.GateHost,
		HTTPPort:     cfg.HTTPPort,
		WSPort:       cfg.WSPort,
		RSAKey:       rsaKey,
		State:        state,
		ProtocolName: names,
	})
}

// Run 启动服务并响应退出信号，返回前完成服务关闭
func Run() error {
	app, err := initApp()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return app.Run(ctx, SetupRouter(), SetupGameRouter(app))
}

func pythonEnvBool(name string) bool {
	value := os.Getenv(name)
	return value != "" && value != "0" && value != "false" && value != "False"
}
