package assets

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeCocosUUID(t *testing.T) {
	got, err := DecodeCocosUUID("00AAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("DecodeCocosUUID() error = %v", err)
	}
	want := "00000000-0000-0000-0000-000000000000"
	if got != want {
		t.Fatalf("DecodeCocosUUID() = %q, want %q", got, want)
	}

	unchanged, err := DecodeCocosUUID("plain-uuid")
	if err != nil || unchanged != "plain-uuid" {
		t.Fatalf("短 UUID 应保持原值，got %q, err %v", unchanged, err)
	}
}

func TestTableLoaderMergesAllSources(t *testing.T) {
	root := t.TempDir()
	bundlePath := filepath.Join(root, "resources", "config.json")
	compressedUUIDs := []string{
		"00AAAAAAAAAAAAAAAAAAAA",
		"11AAAAAAAAAAAAAAAAAAAA",
		"22AAAAAAAAAAAAAAAAAAAA",
	}
	bundle := map[string]any{
		"paths": map[string]any{
			"0": []any{"config/data/Test"},
			"1": []any{"config/data/Test_sub1"},
			"2": []any{"config/data/Extra"},
		},
		"uuids": compressedUUIDs,
	}
	writeJSONFile(t, bundlePath, bundle, false)

	uuids := make([]string, len(compressedUUIDs))
	for index, compressed := range compressedUUIDs {
		var err error
		uuids[index], err = DecodeCocosUUID(compressed)
		if err != nil {
			t.Fatal(err)
		}
	}
	writeImportFile(t, root, uuids[0], map[string]any{
		"json": map[string]any{
			"main":          "kept",
			"_subJsonCount": 1,
			"_subFiles":     map[string]any{"Extra": true},
		},
	}, true)
	writeImportFile(t, root, uuids[1], []any{
		1, 0, 0, []any{}, []any{}, []any{[]any{0, "Test_sub1", map[string]any{
			"main": "ignored",
			"sub":  json.Number("2"),
		}}},
	}, false)
	writeImportFile(t, root, uuids[2], map[string]any{
		"json": map[string]any{"extra": true},
	}, false)

	loader := NewTableLoader(root, bundlePath, nil)
	table, err := loader.LoadTable("Test")
	if err != nil {
		t.Fatalf("LoadTable() error = %v", err)
	}
	if table["main"] != "kept" || table["extra"] != true {
		t.Fatalf("合并结果错误：%#v", table)
	}
	if number, ok := table["sub"].(json.Number); !ok || number.String() != "2" {
		t.Fatalf("子表数值错误：%#v", table["sub"])
	}
	if _, exists := table["_subJsonCount"]; exists {
		t.Fatal("_subJsonCount 不应保留")
	}
	if _, exists := table["_subFiles"]; exists {
		t.Fatal("_subFiles 不应保留")
	}

	// 返回值可修改，但不能污染缓存。
	table["main"] = "changed"
	second, err := loader.LoadTable("Test")
	if err != nil || second["main"] != "kept" {
		t.Fatalf("缓存被调用方修改：%#v, err %v", second, err)
	}
}

func TestTableLoaderRequiresKeyForREFBundle(t *testing.T) {
	root := t.TempDir()
	bundlePath := filepath.Join(root, "config.json")
	if err := os.WriteFile(bundlePath, []byte("REF:\x00\x00\x00\x02"), 0o600); err != nil {
		t.Fatal(err)
	}
	loader := NewTableLoader(root, bundlePath, nil)
	_, err := loader.TablePath("Test")
	if !errors.Is(err, ErrAssetKeyRequired) {
		t.Fatalf("TablePath() error = %v, want ErrAssetKeyRequired", err)
	}
}

func writeImportFile(t *testing.T, root, uuid string, value any, zipped bool) {
	t.Helper()
	path := filepath.Join(root, "resources", "import", uuid[:2], uuid+".json")
	writeJSONFile(t, path, value, zipped)
}

func writeJSONFile(t *testing.T, path string, value any, zipped bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if zipped {
		var output bytes.Buffer
		writer := gzip.NewWriter(&output)
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		data = output.Bytes()
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
