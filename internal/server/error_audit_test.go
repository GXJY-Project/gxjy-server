package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtocolNamesRejectsTrailingInput(t *testing.T) {
	for _, input := range []string{`{"MSG_SYNTHETIC":1} {}`, `{"MSG_SYNTHETIC":1}garbage`} {
		path := filepath.Join(t.TempDir(), "protocol.json")
		if err := os.WriteFile(path, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadProtocolNames(path); err == nil {
			t.Fatal("损坏协议表被当成有效配置")
		}
	}
}
