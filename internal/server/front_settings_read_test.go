package server

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/GXJY-Project/gxjy-server/internal/protocol"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestFrontSettingsMatchesPythonBehavior(t *testing.T) {
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
    "common": {
      "flag": true,
      "count": 3,
      "nested": {"x": [1, null, "a"]}
    },
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
	initial, err = os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	writePlainCommand(t, connection, commandFrontSettings, protocol.Object(), 2)
	response := readPlainCommand(t, connection)

	if response.Command != commandFrontSettings {
		t.Fatalf("回包命令 = %d", response.Command)
	}
	assertProtocolField(t, response.Message, "result", protocol.Uint(0))
	assertProtocolField(t, response.Message, "cmd", protocol.String("MSG_REQ_FRONT_SETTINGS"))
	assertProtocolField(t, response.Message, "version", protocol.Uint(0))
	settings, ok := valueField(response.Message, "settings")
	if !ok {
		t.Fatal("回包缺少 settings")
	}
	common, ok := valueField(settings, "common")
	if !ok {
		t.Fatal("回包缺少 settings.common")
	}
	assertProtocolField(t, common, "flag", protocol.Bool(true))
	assertProtocolField(t, common, "count", protocol.Uint(3))
	nested, ok := valueField(common, "nested")
	if !ok {
		t.Fatal("回包缺少 nested")
	}
	assertProtocolField(t, nested, "x", protocol.Array(protocol.Uint(1), protocol.Nil(), protocol.String("a")))
	needReset, ok := valueField(settings, "need_reset")
	if !ok {
		t.Fatal("回包缺少 settings.need_reset")
	}
	assertProtocolField(t, needReset, "daily", protocol.Bool(false))

	after, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, initial) {
		t.Fatal("读取设置修改了存档")
	}
}
