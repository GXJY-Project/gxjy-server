package game

import (
	"bytes"
	"os"
	"testing"

	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

func TestNewRejectsMalformedPlayerContainersWithoutSaving(t *testing.T) {
	for _, input := range []string{`{"equip_state":true}`, `{"hero_state":1}`, `{"bag_items":[1]}`, `{"currency":true}`, `{"heroes":"invalid"}`, `{"custom":true}`} {
		store := storage.NewJSONStore(t.TempDir())
		if err := os.WriteFile(store.Path(), []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := New(&taskQuerySource{}, store, Options{}); err == nil {
			t.Fatalf("损坏状态被接受：%s", input)
		}
		after, err := os.ReadFile(store.Path())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(after, []byte(input)) {
			t.Fatal("加载失败时覆盖了原存档")
		}
	}
}
