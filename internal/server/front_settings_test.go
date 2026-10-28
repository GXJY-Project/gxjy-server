package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestSaveFrontSettingsMatchesPythonBehavior(t *testing.T) {
	directory := t.TempDir()
	store := storage.NewJSONStore(directory)
	initial := []byte(`{
  "role_id": 10001,
  "role_name": "旧玩家",
  "level": 1,
  "exp": 0,
  "sex": 2,
  "leader": "L001",
  "custom": {
    "common": {"keep": 1, "replace": "old"},
    "need_reset": {"daily": false}
  }
}`)
	if err := os.WriteFile(store.Path(), initial, 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := newBaselineState(t, store)
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

	server := newGameTestServer(t, application)
	defer server.Close()
	connection, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/game", nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	writePlainCommand(t, connection, commandAuth, protocol.Object(), 1)
	for range 11 {
		readPlainCommand(t, connection)
	}

	request := object(field("settings", object(
		field("common", object(
			field("replace", protocol.String("new")),
			field("flag", protocol.Bool(true)),
			field("nested", object(
				field("x", protocol.Array(protocol.Uint(1), protocol.Nil(), protocol.String("a"))),
			)),
		)),
		field("need_reset", object(field("daily", protocol.Bool(true)))),
		field("ignored_group", object(field("bad", protocol.Uint(1)))),
	)))
	writePlainCommand(t, connection, commandSaveFrontSettings, request, 2)
	response := readPlainCommand(t, connection)
	if response.Command != commandSaveFrontSettings {
		t.Fatalf("回包命令 = %d", response.Command)
	}
	settings, ok := valueField(response.Message, "settings")
	if !ok {
		t.Fatal("回包缺少 settings")
	}
	common, ok := valueField(settings, "common")
	if !ok {
		t.Fatal("回包缺少 settings.common")
	}
	assertProtocolField(t, common, "keep", protocol.Uint(1))
	assertProtocolField(t, common, "replace", protocol.String("new"))
	assertProtocolField(t, common, "flag", protocol.Bool(true))
	if _, ok := valueField(settings, "ignored_group"); ok {
		t.Fatal("回包错误保留 ignored_group")
	}
	assertProtocolField(t, response.Message, "version", protocol.Uint(0))
	assertProtocolField(t, response.Message, "result", protocol.Uint(0))

	data, err := os.ReadFile(filepath.Join(directory, "save_localplayer.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	var custom map[string]map[string]json.RawMessage
	if err := json.Unmarshal(saved["custom"], &custom); err != nil {
		t.Fatal(err)
	}
	if string(custom["common"]["keep"]) != "1" || string(custom["common"]["replace"]) != `"new"` {
		t.Fatalf("存档 common = %s", saved["custom"])
	}
	if string(custom["need_reset"]["daily"]) != "true" {
		t.Fatalf("存档 need_reset = %s", saved["custom"])
	}
}

func newGameTestServer(t *testing.T, application *App) *httptest.Server {
	t.Helper()
	return httptest.NewServer(application.GameHandler())
}

func writePlainCommand(t *testing.T, connection *websocket.Conn, command uint16, message protocol.Value, sequence uint32) {
	t.Helper()
	frame, err := protocol.BuildPlainFrame(command, message, sequence)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatal(err)
	}
}

func readPlainCommand(t *testing.T, connection *websocket.Conn) protocol.DecodedBody {
	t.Helper()
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("消息类型 = %d", messageType)
	}
	decoded, _, err := protocol.ParsePlainFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func assertProtocolField(t *testing.T, value protocol.Value, name string, want protocol.Value) {
	t.Helper()
	got, ok := valueField(value, name)
	if !ok {
		t.Fatalf("缺少字段 %q", name)
	}
	if !got.Equal(want) {
		t.Fatalf("字段 %q = %#v, want %#v", name, got, want)
	}
}
