package game

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/internal/assets"
	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

type goldenTables struct {
	Tables map[string]map[string]any
	Keys   map[string][]string
}

func (g goldenTables) LoadTable(name string) (map[string]any, error) { return g.Tables[name], nil }
func (g goldenTables) TableKeys(name string) ([]string, error)       { return g.Keys[name], nil }
func (g goldenTables) ObjectKeys(name string, path ...string) ([]string, error) {
	return g.Keys[name+"/"+strings.Join(path, "/")], nil
}

func TestExistingPathsMatchPython(t *testing.T) {
	path := "testdata/existing_paths_python.json"
	if override := os.Getenv("GXJY_DIFF_FIXTURE"); override != "" {
		path = override
	}
	testPathsMatchPython(t, path, nil)
}

func TestMonthCardMatchesPython(t *testing.T) {
	testPathsMatchPython(t, "testdata/month_card_python.json", nil)
}

func TestTasksMatchPython(t *testing.T) {
	path := "testdata/tasks_python.json"
	if override := os.Getenv("GXJY_TASK_FIXTURE"); override != "" {
		path = override
	}
	testPathsMatchPython(t, path, nil)
}

func TestTasksRealAssetsMatchPython(t *testing.T) {
	path, configPath := os.Getenv("GXJY_TASK_FIXTURE"), os.Getenv("GXJY_TASK_CONFIG")
	if path == "" || configPath == "" {
		t.Skip("通过 task_baseline.py 生成真实表差分后启用")
	}
	cfg, err := config.LoadRuntime(configPath)
	if err != nil {
		t.Fatal(err)
	}
	key, err := config.ReadAssetKey(cfg.AssetKeyEnv, cfg.AssetKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	testPathsMatchPython(t, path, assets.NewTableLoader(cfg.AssetRoot, cfg.BundlePath, key))
}

func TestMonthCardRealAssetsMatchPython(t *testing.T) {
	path, configPath := os.Getenv("GXJY_MONTH_CARD_FIXTURE"), os.Getenv("GXJY_MONTH_CARD_CONFIG")
	if path == "" {
		t.Skip("通过 month_card_baseline.py 生成真实资源差分后启用")
	}
	cfg, err := config.LoadRuntime(configPath)
	if err != nil {
		t.Fatal(err)
	}
	key, err := config.ReadAssetKey(cfg.AssetKeyEnv, cfg.AssetKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	// 只使用本地资源与临时合成存档，不读取配置中的真实 save_dir
	loader := assets.NewTableLoader(cfg.AssetRoot, cfg.BundlePath, key)
	testPathsMatchPython(t, path, loader)
}

func testPathsMatchPython(t *testing.T, path string, source hero.TableSource) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Tables map[string]map[string]any `json:"tables"`
		Keys   map[string][]string       `json:"table_keys"`
		Cases  []struct {
			Name           string                    `json:"name"`
			Initial        map[string]any            `json:"initial"`
			InitialJSON    string                    `json:"initial_json"`
			TablesOverride map[string]map[string]any `json:"tables_override"`
			KeysOverride   map[string][]string       `json:"table_keys_override"`
			Options        struct {
				Now       int64 `json:"now"`
				SkipGuide bool  `json:"skip_guide"`
				Replay    bool  `json:"replay_prologue"`
			} `json:"options"`
			Startup     map[string]any `json:"startup"`
			StartupDisk map[string]any `json:"startup_disk"`
			Steps       []struct {
				Command  uint16         `json:"command"`
				Request  any            `json:"request"`
				Messages []Message      `json:"messages"`
				State    map[string]any `json:"state"`
				Disk     map[string]any `json:"disk"`
			} `json:"steps"`
		} `json:"cases"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err = dec.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Cases {
		t.Run(c.Name, func(t *testing.T) {
			store := storage.NewJSONStore(t.TempDir())
			if c.Initial != nil {
				b, _ := json.Marshal(c.Initial)
				if c.InitialJSON != "" {
					b = []byte(c.InitialJSON)
				}
				if err := os.WriteFile(store.Path(), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			tables := make(map[string]map[string]any, len(fixture.Tables))
			for name, table := range fixture.Tables {
				tables[name] = table
			}
			for name, table := range c.TablesOverride {
				tables[name] = table
			}
			keys := make(map[string][]string, len(fixture.Keys))
			for name, values := range fixture.Keys {
				keys[name] = values
			}
			for name, values := range c.KeysOverride {
				keys[name] = values
			}
			caseSource := source
			if caseSource == nil {
				caseSource = goldenTables{tables, keys}
			}
			runtime, err := New(caseSource, store, Options{Now: func() time.Time { return time.Unix(c.Options.Now, 0) }, Choose: func(n int) int { return 0 }, SkipGuide: c.Options.SkipGuide, ReplayPrologue: c.Options.Replay})
			if err != nil {
				t.Fatal(err)
			}
			assertJSON(t, "startup", runtime.Snapshot(), c.Startup)
			checkDisk := func(want map[string]any) {
				t.Helper()
				var got map[string]any
				_, err := store.Load(&got)
				if err != nil {
					t.Fatal(err)
				}
				assertJSON(t, "disk", got, want)
			}
			checkDisk(c.StartupDisk)
			for _, s := range c.Steps {
				got, err := runtime.Handle(s.Command, s.Request)
				if err != nil {
					t.Fatalf("command %d: %v", s.Command, err)
				}
				assertJSON(t, "messages", got, s.Messages)
				assertJSON(t, "state", runtime.Snapshot(), s.State)
				checkDisk(s.Disk)
			}
		})
	}
}

func assertJSON(t *testing.T, label string, got, want any) {
	t.Helper()
	a, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var av, bv any
	ad := json.NewDecoder(bytes.NewReader(a))
	ad.UseNumber()
	_ = ad.Decode(&av)
	bd := json.NewDecoder(bytes.NewReader(b))
	bd.UseNumber()
	_ = bd.Decode(&bv)
	if !reflect.DeepEqual(av, bv) {
		t.Fatalf("%s differs\ngot: %s\nwant: %s", label, a, b)
	}
}
