package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/internal/assets"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

// TestBiliRuntimeMatchesPython 实际读取 B 服 33 张表并复核合成存档的业务变化
// 来源与边界见 docs/bili-runtime-validation.md，默认测试不读取本地资源或真实存档
func TestBiliRuntimeMatchesPython(t *testing.T) {
	configPath := os.Getenv("GXJY_BILI_RUNTIME_CONFIG")
	if configPath == "" {
		t.Skip("通过 scripts/check_bili_runtime.py 启用独立 B 服验证")
	}
	cfg, err := config.LoadRuntime(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configDir, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SaveDir != filepath.Join(configDir, "go-saves") || cfg.AssetRoot == "" || cfg.BundlePath == "" || cfg.AssetKeyEnv != "GXJY_BILI_TEST_ASSET_KEY" {
		t.Fatal("B 服验证必须使用独立资源、存档与测试密钥入口")
	}
	if os.Getenv("GXJY_BILI_RUNTIME_REPORT") != filepath.Join(configDir, "go-runtime-report.json") {
		t.Fatal("B 服报告必须写入此次独立配置所在目录")
	}
	key := os.Getenv(cfg.AssetKeyEnv)
	if key == "" {
		t.Fatal("缺少 B 服测试资源密钥")
	}
	var fixture struct {
		Kind  string `json:"tables_kind"`
		Cases []struct {
			Name    string         `json:"name"`
			Initial map[string]any `json:"initial"`
			Options struct {
				Now       int64 `json:"now"`
				SkipGuide bool  `json:"skip_guide"`
				Replay    bool  `json:"replay_prologue"`
			} `json:"options"`
			Startup     any `json:"startup"`
			StartupDisk any `json:"startup_disk"`
			Restart     any `json:"restart"`
			RestartDisk any `json:"restart_disk"`
			Steps       []struct {
				Command  uint16    `json:"command"`
				Request  any       `json:"request"`
				Messages []Message `json:"messages"`
				State    any       `json:"state"`
				Disk     any       `json:"disk"`
			} `json:"steps"`
		} `json:"cases"`
	}
	readBiliJSON(t, os.Getenv("GXJY_BILI_RUNTIME_FIXTURE"), &fixture)
	if fixture.Kind != "bili-281-authorized" || len(requiredTables) != 33 || len(fixture.Cases) < 9 {
		t.Fatal("B 服验证样本来源、表数或用例不完整")
	}
	loader := assets.NewTableLoader(cfg.AssetRoot, cfg.BundlePath, []byte(key))
	report := record{"tables": record{}, "cases": []any{}, "status": "verified", "issues": []any{}}
	for _, name := range requiredTables {
		table, err := loader.LoadTable(name)
		if err != nil {
			t.Fatalf("真实 B 服配置 %s：%v", name, err)
		}
		obj(report["tables"])[name] = len(table)
	}
	if err := os.MkdirAll(cfg.SaveDir, 0700); err != nil {
		t.Fatal(err)
	}
	runDir, err := os.MkdirTemp(cfg.SaveDir, "run-")
	if err != nil {
		t.Fatal(err)
	}
	var auditRuntime *Runtime
	var clientRoutes struct {
		Request  uint16 `json:"bag_request_command"`
		Response uint16 `json:"bag_response_command"`
	}
	readBiliJSON(t, os.Getenv("GXJY_BILI_CLIENT_EXPECTATIONS"), &clientRoutes)
	if clientRoutes.Request != CommandBagRequest || clientRoutes.Response == 0 {
		t.Fatal("B 服背包消息表缺少必要路由")
	}
	for _, sample := range fixture.Cases {
		counts := record{"name": sample.Name, "steps": len(sample.Steps), "prototype_status": "failed"}
		counts["routing_issues"] = []any{}
		passed := t.Run(sample.Name, func(t *testing.T) {
			// 每个用例写入此次运行的新目录，保留测试存档供复核
			store := storage.NewJSONStore(filepath.Join(runDir, sample.Name))
			if sample.Initial != nil {
				if err := store.Save(sample.Initial); err != nil {
					t.Fatal(err)
				}
			}
			r, err := New(loader, store, Options{Now: func() time.Time { return time.Unix(sample.Options.Now, 0) },
				Choose: func(n int) int { return 0 }, SkipGuide: sample.Options.SkipGuide, ReplayPrologue: sample.Options.Replay})
			if err != nil {
				t.Fatal(err)
			}
			if len(r.tables) != 33 {
				t.Fatal("Runtime 没有载入全部 33 张配置表")
			}
			auditRuntime = r
			if sample.Initial == nil {
				counts["initial_values"] = record{"heroes": r.ownedIDs(), "head_image": r.state["head_image"],
					"wife": or(r.state["wife"], "H002"), "currency": copyValue(r.state["currency"]), "leader": r.state["leader"]}
			}
			biliEqual(t, "启动内存", r.Snapshot(), sample.Startup)
			checkDisk := func(want any) {
				t.Helper()
				var disk any
				if _, err := store.Load(&disk); err != nil {
					t.Fatal(err)
				}
				biliEqual(t, "存档", disk, want)
			}
			checkDisk(sample.StartupDisk)
			counts["save"] = store.Path()
			for _, step := range sample.Steps {
				messages, err := r.Handle(step.Command, step.Request)
				if err != nil {
					t.Fatalf("command %d：%v", step.Command, err)
				}
				biliEqual(t, fmt.Sprintf("命令 %d 回包", step.Command), messages, step.Messages)
				if step.Command == clientRoutes.Request {
					if len(messages) != 1 || messages[0].Command != clientRoutes.Response {
						commands := []uint16{}
						for _, message := range messages {
							commands = append(commands, message.Command)
						}
						counts["routing_issues"] = append(arr(counts["routing_issues"]), record{"request": step.Command,
							"go": commands, "bili": clientRoutes.Response})
					}
				}
				biliEqual(t, fmt.Sprintf("命令 %d 内存", step.Command), r.Snapshot(), step.State)
				checkDisk(step.Disk)
			}
			counts["prototype_status"] = "verified"
			issues := validateBiliReferences(r)
			counts["issues"] = issues
			counts["config_status"] = "verified"
			if len(issues) > 0 {
				counts["config_status"] = "incompatible"
			}
			counts["heroes"] = len(r.ownedHeroes())
			counts["equips"] = len(obj(r.state["equip_state"]))
			counts["bag_items"] = len(r.bagItems())
			counts["hero_data"] = r.ownedHeroes()
			counts["bag_data"] = r.bagItems()
			counts["hero_state"] = copyValue(r.state["hero_state"])
			if r.err != nil {
				t.Fatal(r.err)
			}
			reloaded, err := New(loader, store, r.options)
			if err != nil {
				t.Fatal(err)
			}
			biliEqual(t, "重载内存", reloaded.Snapshot(), sample.Restart)
			checkDisk(sample.RestartDisk)
			counts["restart_status"] = "verified"
		})
		report["cases"] = append(arr(report["cases"]), counts)
		if !passed {
			report["status"] = "failed"
		}
	}
	if auditRuntime != nil {
		report["client_attributes"] = checkBiliClientAttributes(t, auditRuntime)
	}
	// Python 相等与 B 服验收分别记录，旧原型也可能构造无效的配置引用
	for _, value := range arr(report["cases"]) {
		counts := obj(value)
		issues := arr(counts["issues"])
		if len(issues) > 0 {
			report["status"] = "incompatible"
			t.Errorf("%s：%d 个 B 配置引用无效，详情见独立报告", counts["name"], len(issues))
		}
		if issues := arr(counts["routing_issues"]); len(issues) > 0 {
			report["status"] = "incompatible"
			t.Errorf("%s：%d 次背包回包未使用 B 服响应编号", counts["name"], len(issues))
		}
	}
	if len(arr(obj(report["client_attributes"])["differences"])) > 0 {
		report["status"] = "incompatible"
	}
	if path := os.Getenv("GXJY_BILI_RUNTIME_REPORT"); path != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		// 此处只写显式设置的测试报告，实际资源和正常存档不会被替换
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func readBiliJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func biliEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	normalize := func(value any) any {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var result any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	if !reflect.DeepEqual(normalize(got), normalize(want)) {
		t.Fatalf("%s 与 Python 不一致", label)
	}
}

func validateBiliReferences(r *Runtime) []any {
	issues := []any{}
	check := func(table string, id any, owner string) {
		if !truth(id) || len(r.row(table, id)) == 0 {
			issues = append(issues, record{"owner": owner, "table": table, "id": copyValue(id)})
		}
	}
	heroes := r.ownedHeroes()
	for _, hid := range sortedKeys(heroes) {
		hero := obj(heroes[hid])
		base := obj(hero["base_attr"])
		check("HeroQuality", base["quality_id"], hid)
		check("HeroStar", obj(base["star"])["star_id"], hid)
		// B 服不读取旧 awaken_id，Awaken 数组是配置阶段列表，由原客户端另行复核
		check("HeroData", obj(base["hero_model"])["hero_surface"], hid)
		check("HeroData", obj(base["hero_model"])["leader_surface"], hid)
		if truth(hero["favor_id"]) {
			check("HeroFavor", hero["favor_id"], hid)
		}
		for _, name := range sortedKeys(obj(obj(hero["skills"])["hero_skills"])) {
			check("SkillBase", name, hid)
		}
	}
	bag := r.bagItems()
	for _, iid := range sortedKeys(bag) {
		item := obj(bag[iid])
		if text(item["item_type"]) != "Equip" {
			continue
		}
		check("EquipBase", item["template_id"], iid)
		data := obj(item["data"])
		check("EquipExp", data["level_id"], iid)
		check("EquipQuality", data["quality"], iid)
		check("EquipStar", data["star"], iid)
	}
	return issues
}

func checkBiliClientAttributes(t *testing.T, r *Runtime) record {
	t.Helper()
	var expected struct {
		Source string `json:"source_sha256"`
		Heroes []struct {
			ID      string `json:"id"`
			Level   int64  `json:"level"`
			Quality int64  `json:"quality"`
			Star    int64  `json:"star"`
			Attrs   record `json:"attrs"`
		} `json:"heroes"`
		Equipment []struct {
			ID        string `json:"id"`
			Level     int64  `json:"level"`
			Quality   int64  `json:"quality"`
			Star      int64  `json:"star"`
			LevelID   string `json:"level_id"`
			QualityID string `json:"quality_id"`
			StarID    string `json:"star_id"`
			Attrs     record `json:"attrs"`
		} `json:"equipment"`
	}
	readBiliJSON(t, os.Getenv("GXJY_BILI_CLIENT_EXPECTATIONS"), &expected)
	if expected.Source != "a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0" || len(expected.Heroes) == 0 || len(expected.Equipment) != 290 {
		t.Fatal("缺少更新版 B 服客户端原函数验证输入")
	}
	report := record{"hero_vectors": len(expected.Heroes), "equipment_vectors": len(expected.Equipment), "differences": []any{}}
	differences := []any{}
	for _, sample := range expected.Heroes {
		r.state["hero_state"] = record{sample.ID: record{"level": sample.Level, "quality": sample.Quality, "star": sample.Star}}
		for _, attr := range []string{"Atk", "PhyDef", "MagDef", "Hp"} {
			got, want := r.heroAttr(sample.ID, attr), iv(sample.Attrs[attr])
			if got != want {
				differences = append(differences, record{"kind": "hero_attr", "id": sample.ID, "level": sample.Level,
					"quality": sample.Quality, "star": sample.Star, "field": attr, "go": got, "bili": want})
			}
		}
	}
	for _, sample := range expected.Equipment {
		got := r.equipData(record{"template_id": sample.ID, "level": sample.Level, "quality_num": sample.Quality, "star_num": sample.Star})
		for field, want := range (record{"level_id": sample.LevelID, "quality": sample.QualityID, "star": sample.StarID}) {
			if got[field] != want {
				differences = append(differences, record{"kind": "equip_id", "id": sample.ID, "level": sample.Level,
					"quality": sample.Quality, "field": field, "go": got[field], "bili": want})
			}
		}
		for attr, want := range sample.Attrs {
			if iv(obj(got["attrs"])[attr]) != iv(want) {
				differences = append(differences, record{"kind": "equip_attr", "id": sample.ID, "level": sample.Level,
					"quality": sample.Quality, "field": attr, "go": obj(got["attrs"])[attr], "bili": want})
			}
		}
		if sample.Level == 1 && sample.Quality == 0 {
			initial := r.equipData(record{"template_id": sample.ID})
			if initial["star"] != sample.StarID {
				differences = append(differences, record{"kind": "equip_initial_star", "id": sample.ID,
					"go": initial["star"], "bili": sample.StarID})
			}
		}
	}
	sort.Slice(differences, func(i, j int) bool {
		a, _ := json.Marshal(differences[i])
		b, _ := json.Marshal(differences[j])
		return bytes.Compare(a, b) < 0
	})
	report["differences"] = differences
	if len(differences) > 0 {
		t.Errorf("与 B 服原属性/装备方法存在 %d 项差异，详情见独立报告", len(differences))
	}
	return report
}
