package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProtocolNamesPrefersClientMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "protocol_map.json")
	data := []byte(`{"MSG_GAME2WORLD_DUP":1019,"MSG_CHAT_INFO_RES":1019,"MSG_TYPE_AUTH":101}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	names, err := LoadProtocolNames(path)
	if err != nil {
		t.Fatal(err)
	}
	if names[1019] != "MSG_CHAT_INFO_RES" || names[101] != "MSG_TYPE_AUTH" {
		t.Fatalf("names = %#v", names)
	}
}
