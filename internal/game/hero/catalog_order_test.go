package hero

import "testing"

type invalidOrderSource struct{ keys []string }

func (s invalidOrderSource) LoadTable(string) (map[string]any, error) {
	return map[string]any{"A": 1, "B": 2}, nil
}
func (s invalidOrderSource) TableKeys(string) ([]string, error) { return s.keys, nil }

func TestCatalogRejectsIncompleteDuplicateOrUnknownKeyOrder(t *testing.T) {
	for _, keys := range [][]string{{}, {"A"}, {"A", "A"}, {"A", "Missing"}} {
		if _, err := NewCatalog(invalidOrderSource{keys}).Table("Synthetic"); err == nil {
			t.Fatalf("不一致的键顺序被接受：%v", keys)
		}
	}
}
