package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	config.InitLog()
	global.Logger.SetOutput(io.Discard)
	config.AppConfig = config.DefaultRuntime()
	os.Exit(m.Run())
}

func captureLogs(t *testing.T, output *bytes.Buffer) {
	t.Helper()
	previous := global.Logger.Out
	global.Logger.SetOutput(output)
	t.Cleanup(func() { global.Logger.SetOutput(previous) })
}

func testApp(t *testing.T) *server.App {
	t.Helper()
	app, err := server.New(server.Config{
		HTTPAddress: "127.0.0.1:0", WSAddress: "127.0.0.1:0",
		GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765,
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestLoginRoutesPreservePathDispatch(t *testing.T) {
	r := SetupRouter()
	// 来源：原 LoginHandler.response 与 existing_paths_python.json 的 HTTP 样本
	for _, test := range []struct {
		method, path, field string
		want                any
	}{
		{http.MethodGet, "/healthz", "status", "ok"},
		{http.MethodPost, "/PLAYER/LOGIN", "recommend", "1"},
		{http.MethodGet, "/assets/version.manifest", "version", "1.0.0"},
		{http.MethodGet, "/nested/project.manifest", "version", "1.0.0"},
		{http.MethodGet, "/unknown/", "url", "ws://127.0.0.1:8765/game"},
		{"PROPFIND", "/player/enter", "url", "ws://127.0.0.1:8765/game"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body[test.field] != test.want {
				t.Fatalf("%s=%v, want %v", test.field, body[test.field], test.want)
			}
		})
	}
}

func TestGameRoutesPreserveUpgradeAndNotFound(t *testing.T) {
	r := SetupGameRouter(testApp(t))
	for _, test := range []struct {
		path   string
		status int
	}{
		{"/game", http.StatusBadRequest},
		{"/game/", http.StatusNotFound},
		{"/GAME", http.StatusNotFound},
		{"/healthz", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s: status=%d, want %d", test.path, response.Code, test.status)
		}
	}
}

func TestGinWebSocketUpgradeAndFrameValidation(t *testing.T) {
	srv := httptest.NewServer(SetupGameRouter(testApp(t)))
	defer srv.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/game", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// 来源：现有 WebSocket 只接受二进制消息，Gin 升级后仍走相同的校验
	if err := connection.WriteMessage(websocket.TextMessage, []byte("synthetic-text-frame")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := connection.ReadMessage(); err == nil {
		t.Fatal("文本消息没有触发连接关闭")
	} else if timeout, ok := err.(interface{ Timeout() bool }); ok && timeout.Timeout() {
		t.Fatalf("等待连接关闭超时：%v", err)
	}
}

func TestRunClosesGinServersOnCancellation(t *testing.T) {
	app := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.Run(ctx, SetupRouter(), SetupGameRouter(app)); err != nil {
		t.Fatal(err)
	}
}

type fixtureTables struct {
	Tables map[string]map[string]any `json:"tables"`
	Keys   map[string][]string       `json:"table_keys"`
}

func (f fixtureTables) LoadTable(name string) (map[string]any, error) { return f.Tables[name], nil }
func (f fixtureTables) TableKeys(name string) ([]string, error)       { return f.Keys[name], nil }
func (f fixtureTables) ObjectKeys(name string, path ...string) ([]string, error) {
	return f.Keys[name+"/"+strings.Join(path, "/")], nil
}

func TestGinWebSocketAuthPushOrder(t *testing.T) {
	// 来源：复用既有合成 Python 样本和 TestPlainSessionAuthPushOrder 的预期顺序
	file, err := os.Open("../internal/game/testdata/existing_paths_python.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var tables fixtureTables
	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	if err := decoder.Decode(&tables); err != nil {
		t.Fatal(err)
	}
	state, err := server.NewGameStateFromSource(tables, nil, game.Options{
		Now:    func() time.Time { return time.Unix(1800000000, 0) },
		Choose: func(int) int { return 0 },
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := server.New(server.Config{
		HTTPAddress: "127.0.0.1:0", WSAddress: "127.0.0.1:0",
		GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765, State: state,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(SetupGameRouter(app))
	defer srv.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/game", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	frame, err := protocol.BuildPlainFrame(game.CommandAuth, protocol.Object(protocol.Named("token", protocol.String("test"))), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatal(err)
	}
	for index, command := range []uint16{
		game.CommandAuth, game.CommandRoleInfo, game.CommandLoginInfo, game.CommandHeroes,
		game.CommandTeams, game.CommandWorldMap, game.CommandSystemUnlock, game.CommandPushReminds,
		game.CommandRedPoints, game.CommandSummitBase, game.CommandActivities,
	} {
		kind, body, err := connection.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		decoded, _, err := protocol.ParsePlainFrame(body)
		if err != nil {
			t.Fatal(err)
		}
		if kind != websocket.BinaryMessage || decoded.Command != command {
			t.Fatalf("第 %d 个推送：type=%d command=%d, want %d", index, kind, decoded.Command, command)
		}
	}
}
