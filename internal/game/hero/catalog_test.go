package hero

import (
	"errors"
	"testing"
)

type fakeTables struct {
	tables map[string]map[string]any
	loads  map[string]int
	err    error
}

func (source *fakeTables) LoadTable(name string) (map[string]any, error) {
	if source.err != nil {
		return nil, source.err
	}
	if source.loads == nil {
		source.loads = make(map[string]int)
	}
	source.loads[name]++
	return source.tables[name], nil
}

func TestCatalogRequiresTableSource(t *testing.T) {
	catalog := NewCatalog(nil)
	_, err := catalog.Row("HeroBase", "H002")
	if !errors.Is(err, ErrTableSourceRequired) {
		t.Fatalf("错误 = %v", err)
	}
}

func TestCatalogRejectsMissingTableAndRow(t *testing.T) {
	source := &fakeTables{tables: map[string]map[string]any{
		"HeroBase": {
			"H002": map[string]any{"Id": "H002"},
		},
	}}
	catalog := NewCatalog(source)

	if _, err := catalog.Row("SkillBase", "HeroSkill_Missing"); !errors.Is(err, ErrTableNotFound) {
		t.Fatalf("缺表错误 = %v", err)
	}
	if _, err := catalog.Row("HeroBase", "H004"); !errors.Is(err, ErrRowNotFound) {
		t.Fatalf("缺行错误 = %v", err)
	}
	row, err := catalog.Row("HeroBase", "H002")
	if err != nil {
		t.Fatal(err)
	}
	if row["Id"] != "H002" {
		t.Fatalf("配置行 = %#v", row)
	}
}

func TestCatalogCachesLoadedTables(t *testing.T) {
	source := &fakeTables{tables: map[string]map[string]any{
		"HeroBase": {
			"H002": map[string]any{"Id": "H002"},
			"H004": map[string]any{"Id": "H004"},
		},
	}}
	catalog := NewCatalog(source)
	if _, err := catalog.Row("HeroBase", "H002"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Row("HeroBase", "H004"); err != nil {
		t.Fatal(err)
	}
	if source.loads["HeroBase"] != 1 {
		t.Fatalf("HeroBase 加载次数 = %d", source.loads["HeroBase"])
	}
}
