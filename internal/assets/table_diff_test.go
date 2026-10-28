package assets

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestActualTablesMatchPython(t *testing.T) {
	path := os.Getenv("GXJY_DIFF_FIXTURE")
	if path == "" {
		t.Skip("通过 scripts/check_python_diff.py 启用真实资源差分")
	}
	root, bundle, key := os.Getenv("GXJY_DIFF_ASSET_ROOT"), os.Getenv("GXJY_DIFF_BUNDLE"), os.Getenv("GXJY_ASSET_KEY")
	if root == "" || bundle == "" || key == "" {
		t.Fatal("真实资源差分缺少目录、Bundle 或密钥环境变量")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Tables map[string]map[string]any `json:"tables"`
		Keys   map[string][]string       `json:"table_keys"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	loader := NewTableLoader(root, bundle, []byte(key))
	for name, want := range fixture.Tables {
		t.Run(name, func(t *testing.T) {
			got, err := loader.LoadTable(name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("配置表 %s 与 Python 不一致", name)
			}
			keys, err := loader.TableKeys(name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(keys, fixture.Keys[name]) {
				t.Fatalf("配置表 %s 键顺序不一致", name)
			}
		})
	}
}

func TestTableLoaderPreservesNestedOrderAndCache(t *testing.T) {
	root := t.TempDir()
	bundle := root + "/config.json"
	writeJSONFile(t, bundle, map[string]any{"paths": map[string]any{"0": []any{"config/data/Test"}}, "uuids": []string{"00AAAAAAAAAAAAAAAAAAAA"}}, false)
	uuid, _ := DecodeCocosUUID("00AAAAAAAAAAAAAAAAAAAA")
	writeImportFile(t, root, uuid, map[string]any{"json": map[string]any{"Row": map[string]any{"Config": map[string]any{"z": map[string]any{"value": 1}, "a": map[string]any{"value": 2}}}}}, false)
	// 特意手写顺序以检验 JSON 对象顺序保真
	path := root + "/resources/import/" + uuid[:2] + "/" + uuid + ".json"
	if err := os.WriteFile(path, []byte(`{"json":{"Row":{"Config":{"z":{"value":1},"a":{"value":2}}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewTableLoader(root, bundle, nil)
	got, err := loader.LoadTable("Test")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := loader.ObjectKeys("Test", "Row", "Config")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(keys, []string{"z", "a"}) {
		t.Fatalf("对象顺序=%v", keys)
	}
	got["Row"].(map[string]any)["Config"].(map[string]any)["z"].(map[string]any)["value"] = "changed"
	again, err := loader.LoadTable("Test")
	if err != nil {
		t.Fatal(err)
	}
	if again["Row"].(map[string]any)["Config"].(map[string]any)["z"].(map[string]any)["value"] == "changed" {
		t.Fatal("嵌套修改污染配置缓存")
	}
}
