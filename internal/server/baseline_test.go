package server

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/internal/game"
)

type baselineTables struct {
	Tables map[string]map[string]any `json:"tables"`
	Keys   map[string][]string       `json:"table_keys"`
}

func (b baselineTables) LoadTable(name string) (map[string]any, error) { return b.Tables[name], nil }
func (b baselineTables) TableKeys(name string) ([]string, error)       { return b.Keys[name], nil }
func (b baselineTables) ObjectKeys(name string, path ...string) ([]string, error) {
	return b.Keys[name+"/"+strings.Join(path, "/")], nil
}
func newBaselineState(t *testing.T, store game.Store) (*GameState, error) {
	t.Helper()
	data, err := os.ReadFile("../game/testdata/existing_paths_python.json")
	if err != nil {
		t.Fatal(err)
	}
	var tables baselineTables
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err = d.Decode(&tables); err != nil {
		t.Fatal(err)
	}
	return NewGameStateFromSource(tables, store, game.Options{Now: func() time.Time { return time.Unix(1800000000, 0) }, Choose: func(n int) int { return 0 }})
}
