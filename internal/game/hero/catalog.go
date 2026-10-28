package hero

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	ErrTableSourceRequired = errors.New("需要配置表来源")
	ErrTableNotFound       = errors.New("配置表不存在")
	ErrRowNotFound         = errors.New("配置行不存在")
)

// TableSource 提供配置表，空表与加载失败分别表示
type TableSource interface {
	LoadTable(string) (map[string]any, error)
}

// OrderedTableSource 保留 Python 配置字典的插入顺序
type OrderedTableSource interface {
	TableKeys(string) ([]string, error)
}

// Catalog 缓存配置，调用方只能修改返回的副本
type Catalog struct {
	mu     sync.Mutex
	source TableSource
	tables map[string]map[string]any
	keys   map[string][]string
}

func NewCatalog(source TableSource) *Catalog {
	return &Catalog{source: source, tables: map[string]map[string]any{}, keys: map[string][]string{}}
}
func (c *Catalog) Table(name string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.source == nil {
		return nil, ErrTableSourceRequired
	}
	if t, ok := c.tables[name]; ok {
		return clone(t).(map[string]any), nil
	}
	t, err := c.source.LoadTable(name)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, name)
	}
	keys := make([]string, 0, len(t))
	if ordered, ok := c.source.(OrderedTableSource); ok {
		keys, err = ordered.TableKeys(name)
		if err != nil {
			return nil, err
		}
	} else {
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	if len(keys) != len(t) {
		return nil, fmt.Errorf("配置表 %s 的键顺序与内容不一致", name)
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if _, exists := t[key]; !exists || seen[key] {
			return nil, fmt.Errorf("配置表 %s 的键顺序含缺失或重复项", name)
		}
		seen[key] = true
	}
	c.tables[name] = clone(t).(map[string]any)
	c.keys[name] = append([]string{}, keys...)
	return clone(t).(map[string]any), nil
}
func (c *Catalog) Keys(name string) ([]string, error) {
	if _, err := c.Table(name); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string{}, c.keys[name]...), nil
}
func (c *Catalog) Row(table, key string) (map[string]any, error) {
	t, err := c.Table(table)
	if err != nil {
		return nil, err
	}
	row, ok := t[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s.%s", ErrRowNotFound, table, key)
	}
	return row, nil
}
func clone(v any) any {
	switch x := v.(type) {
	case map[string]any:
		r := make(map[string]any, len(x))
		for k, v := range x {
			r[k] = clone(v)
		}
		return r
	case []any:
		r := make([]any, len(x))
		for i, v := range x {
			r[i] = clone(v)
		}
		return r
	default:
		return v
	}
}
