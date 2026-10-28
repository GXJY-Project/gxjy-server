package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestPlainSessionAuthPushOrder(t *testing.T) {
	state, err := newBaselineState(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	application, err := New(Config{
		HTTPAddress: "127.0.0.1:8080",
		WSAddress:   "127.0.0.1:8765",
		GateHost:    "127.0.0.1",
		HTTPPort:    8080,
		WSPort:      8765,
		State:       state,
	})
	if err != nil {
		t.Fatal(err)
	}

	httpServer := httptest.NewServer(application.GameHandler())
	defer httpServer.Close()
	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/game"
	connection, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	authFrame, err := protocol.BuildPlainFrame(commandAuth, object(field("token", protocol.String("test"))), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, authFrame); err != nil {
		t.Fatal(err)
	}

	want := []uint16{
		commandAuth, commandRoleInfo, commandLoginInfo, commandHeroes,
		commandTeams, commandWorldMap, commandSystemUnlock, commandPushReminds,
		commandRedPoints, commandSummitBase, commandActivities,
	}
	for index, command := range want {
		messageType, body, err := connection.ReadMessage()
		if err != nil {
			t.Fatalf("读取第 %d 个推送：%v", index, err)
		}
		if messageType != websocket.BinaryMessage {
			t.Fatalf("第 %d 个推送不是二进制消息", index)
		}
		decoded, _, err := protocol.ParsePlainFrame(body)
		if err != nil {
			t.Fatalf("解析第 %d 个推送：%v", index, err)
		}
		if decoded.Command != command {
			t.Fatalf("第 %d 个命令 = %d, want %d", index, decoded.Command, command)
		}
	}
}

func TestGameHandlerUpgradeFailureDoesNotWriteHeaderTwice(t *testing.T) {
	application, err := New(Config{
		HTTPAddress: "127.0.0.1:8080",
		WSAddress:   "127.0.0.1:8765",
		GateHost:    "127.0.0.1",
		HTTPPort:    8080,
		WSPort:      8765,
	})
	if err != nil {
		t.Fatal(err)
	}

	var serverLog bytes.Buffer
	httpServer := httptest.NewUnstartedServer(application.GameHandler())
	httpServer.Config.ErrorLog = log.New(&serverLog, "", 0)
	httpServer.Start()
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/game")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
	if strings.Contains(serverLog.String(), "superfluous response.WriteHeader") {
		t.Fatalf("发生重复状态码写入：%s", serverLog.String())
	}
}

func TestRoleUpdatePersistsAcrossStateReload(t *testing.T) {
	store := storage.NewJSONStore(filepath.Join(t.TempDir(), "save"))
	state, err := newBaselineState(t, store)
	if err != nil {
		t.Fatal(err)
	}
	message := object(field("role_name", protocol.String("独立玩家")))
	if err := state.updateRole(commandUpdateRoleName, message); err != nil {
		t.Fatal(err)
	}

	reloaded, err := newBaselineState(t, store)
	if err != nil {
		t.Fatal(err)
	}
	player := reloaded.runtime.Snapshot()
	if player["role_name"] != "独立玩家" || fmt.Sprint(player["role_id"]) != defaultRoleID {
		t.Fatalf("恢复玩家 = %#v", player)
	}
}
