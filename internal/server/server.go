package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GXJY-Project/gxjy-server/global"
	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
)

// Config 是 Go 服务端的运行参数。
type Config struct {
	HTTPAddress string
	WSAddress   string
	GateHost    string
	HTTPPort    int
	WSPort      int

	RSAKey       *gxjycrypto.RSAKey
	State        *GameState
	ProtocolName map[uint16]string
}

// App 同时承载 HTTP 登录和 WebSocket 游戏连接。
type App struct {
	config Config
}

// New 创建可运行的服务实例。
func New(config Config) (*App, error) {
	if config.HTTPAddress == "" || config.WSAddress == "" || config.GateHost == "" {
		return nil, errors.New("监听地址和 gate-host 不能为空")
	}
	if config.HTTPPort < 1 || config.HTTPPort > 65535 || config.WSPort < 1 || config.WSPort > 65535 {
		return nil, errors.New("端口必须在 1 到 65535 之间")
	}
	if config.State == nil {
		config.State = &GameState{}
	}
	if global.Logger == nil {
		return nil, errors.New("日志尚未初始化，请先调用 config.InitLog")
	}
	if config.ProtocolName == nil {
		config.ProtocolName = make(map[uint16]string)
	}
	return &App{config: config}, nil
}

// Run 启动两个服务，并在上下文结束时关闭它们。
func (app *App) Run(ctx context.Context, loginHandler, gameHandler http.Handler) error {
	httpListener, err := net.Listen("tcp", app.config.HTTPAddress)
	if err != nil {
		return fmt.Errorf("监听 HTTP：%w", err)
	}
	wsListener, err := net.Listen("tcp", app.config.WSAddress)
	if err != nil {
		return errors.Join(fmt.Errorf("监听 WebSocket：%w", err), httpListener.Close())
	}

	httpServer := &http.Server{
		Handler:           loginHandler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	wsServer := &http.Server{
		Handler:           gameHandler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	errorsChannel := make(chan error, 2)
	go func() {
		errorsChannel <- httpServer.Serve(httpListener)
	}()
	go func() {
		errorsChannel <- wsServer.Serve(wsListener)
	}()
	global.Logger.Infof("HTTP 登录服务已启动：address:%s", httpListener.Addr())
	global.Logger.Infof("WebSocket 游戏服务已启动：address:%s", wsListener.Addr())

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpErr := httpServer.Shutdown(shutdownContext)
		wsErr := wsServer.Shutdown(shutdownContext)
		var closeErr error
		if httpErr != nil || wsErr != nil {
			// 超时或关闭失败时释放监听器，保留失败原因供入口记录
			closeErr = errors.Join(httpServer.Close(), wsServer.Close())
		}
		return errors.Join(httpErr, wsErr, closeErr, serveFailure(<-errorsChannel), serveFailure(<-errorsChannel))
	case serveErr := <-errorsChannel:
		closeErr := errors.Join(httpServer.Close(), wsServer.Close())
		return errors.Join(serveFailure(serveErr), closeErr, serveFailure(<-errorsChannel))
	}
}

func serveFailure(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// GameHandler 返回游戏 WebSocket 处理器
func (app *App) GameHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		global.Logger.Infof("收到 WebSocket 请求：method:%s path:%q", request.Method, request.URL.Path)
		if request.URL.Path != "/game" {
			http.NotFound(writer, request)
			return
		}
		connection, err := upgradeWebSocket(writer, request)
		if err != nil {
			global.Logger.Warnf("WebSocket 升级失败：%v", err)
			return
		}
		defer connection.close()
		global.Logger.Info("WebSocket 连接已建立")
		session := &gameSession{
			connection: connection,
			rsaKey:     app.config.RSAKey,
			state:      app.config.State,
			names:      app.config.ProtocolName,
		}
		if err := session.run(); err != nil && !errors.Is(err, net.ErrClosed) {
			global.Logger.Warnf("游戏会话结束：%v", err)
		}
	})
}

// LoadProtocolNames 读取 Python 使用的 protocol_map.json。
func LoadProtocolNames(path string) (map[uint16]string, error) {
	if path == "" {
		return map[uint16]string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, errors.New("协议表不是 JSON 对象")
	}
	type entry struct {
		name    string
		command uint16
	}
	entries := make([]entry, 0, 512)
	for decoder.More() {
		nameToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := nameToken.(string)
		if !ok {
			return nil, errors.New("协议名称不是字符串")
		}
		var command uint64
		if err := decoder.Decode(&command); err != nil {
			return nil, err
		}
		if command > 0xffff {
			return nil, fmt.Errorf("协议编号超出 uint16：%d", command)
		}
		entries = append(entries, entry{name: name, command: uint16(command)})
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("协议表只能包含一个 JSON 对象")
	}

	result := make(map[uint16]string, len(entries))
	for _, item := range entries {
		if isInterServerName(item.name) {
			continue
		}
		if _, exists := result[item.command]; !exists {
			result[item.command] = item.name
		}
	}
	for _, item := range entries {
		if _, exists := result[item.command]; !exists {
			result[item.command] = item.name
		}
	}
	return result, nil
}

func isInterServerName(name string) bool {
	prefixes := []string{
		"MSG_GAME2", "MSG_GLOBAL2", "MSG_SOCIAL2", "MSG_WORLD2", "MSG_WEB2",
		"MSG_2GAME", "MSG_2GLOBAL", "MSG_2SOCIAL", "MSG_2WORLD",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
