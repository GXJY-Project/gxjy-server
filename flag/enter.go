package flag

import (
	stdflag "flag"
	"fmt"

	"github.com/GXJY-Project/gxjy-server/config"
)

// Options 保存命令行选项，Runtime 中只应用显式传入的覆盖项
type Options struct {
	File    string
	DB      bool
	Version bool
	Runtime config.Runtime
}

var flagOptions Options

// Version 可通过构建参数设置，未发布构建默认为 dev
var Version = "dev"

// Parse 集中注册和解析命令行参数
func Parse() {
	bind(stdflag.CommandLine, &flagOptions)
	stdflag.Parse()
}

func bind(flags *stdflag.FlagSet, options *Options) {
	options.Runtime = config.DefaultRuntime()
	flags.StringVar(&options.File, "f", "config.json", "运行配置文件")
	flags.StringVar(&options.File, "config", "config.json", "运行配置文件，等同 -f")
	flags.BoolVar(&options.DB, "db", false, "同步数据库模型后退出")
	flags.BoolVar(&options.Version, "v", false, "显示版本后退出")
	flags.StringVar(&options.Runtime.HTTPHost, "http-host", options.Runtime.HTTPHost, "HTTP 登录监听地址")
	flags.IntVar(&options.Runtime.HTTPPort, "http-port", options.Runtime.HTTPPort, "HTTP 登录端口")
	flags.StringVar(&options.Runtime.WSHost, "ws-host", options.Runtime.WSHost, "WebSocket 监听地址")
	flags.IntVar(&options.Runtime.WSPort, "ws-port", options.Runtime.WSPort, "WebSocket 端口")
	flags.StringVar(&options.Runtime.GateHost, "gate-host", options.Runtime.GateHost, "返回给客户端的游戏服务地址")
	flags.StringVar(&options.Runtime.AssetRoot, "asset-root", options.Runtime.AssetRoot, "Cocos assets/assets 目录")
	flags.StringVar(&options.Runtime.BundlePath, "bundle-config", options.Runtime.BundlePath, "resources/config.json 路径")
	flags.StringVar(&options.Runtime.AssetKeyEnv, "asset-key-env", options.Runtime.AssetKeyEnv, "资源密钥环境变量名称")
	flags.StringVar(&options.Runtime.AssetKeyFile, "asset-key-file", options.Runtime.AssetKeyFile, "资源密钥文件路径")
	flags.StringVar(&options.Runtime.RSAPEM, "rsa-pem", options.Runtime.RSAPEM, "RSA PEM 私钥路径")
	flags.StringVar(&options.Runtime.RSAJSON, "rsa-json", options.Runtime.RSAJSON, "RSA N/D/E JSON 路径")
	flags.StringVar(&options.Runtime.ProtocolMap, "protocol-map", options.Runtime.ProtocolMap, "protocol_map.json 路径")
	flags.StringVar(&options.Runtime.ClientProfile, "client-profile", options.Runtime.ClientProfile, "目标客户端配置契约")
}

// ConfigFile 返回解析后的配置文件路径
func ConfigFile() string { return flagOptions.File }

// PrintVersion 处理版本选项，不依赖运行配置和数据库
func PrintVersion() bool {
	if !flagOptions.Version {
		return false
	}
	fmt.Println("gxjy-server", Version)
	return true
}

// Apply 将显式命令行参数覆盖到已加载的配置，保留零值和空字符串
func Apply() { apply(stdflag.CommandLine, flagOptions, &config.AppConfig) }

func apply(flags *stdflag.FlagSet, options Options, cfg *config.Runtime) {
	flags.Visit(func(option *stdflag.Flag) {
		switch option.Name {
		case "http-host":
			cfg.HTTPHost = options.Runtime.HTTPHost
		case "http-port":
			cfg.HTTPPort = options.Runtime.HTTPPort
		case "ws-host":
			cfg.WSHost = options.Runtime.WSHost
		case "ws-port":
			cfg.WSPort = options.Runtime.WSPort
		case "gate-host":
			cfg.GateHost = options.Runtime.GateHost
		case "asset-root":
			cfg.AssetRoot = options.Runtime.AssetRoot
		case "bundle-config":
			cfg.BundlePath = options.Runtime.BundlePath
		case "asset-key-env":
			cfg.AssetKeyEnv = options.Runtime.AssetKeyEnv
		case "asset-key-file":
			cfg.AssetKeyFile = options.Runtime.AssetKeyFile
		case "rsa-pem":
			cfg.RSAPEM = options.Runtime.RSAPEM
		case "rsa-json":
			cfg.RSAJSON = options.Runtime.RSAJSON
		case "protocol-map":
			cfg.ProtocolMap = options.Runtime.ProtocolMap
		case "client-profile":
			cfg.ClientProfile = options.Runtime.ClientProfile
		}
	})
}
