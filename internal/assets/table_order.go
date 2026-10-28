package assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// TableKeys 返回原始配置的键顺序，供名字池和稳定排序使用
func (loader *TableLoader) TableKeys(name string) ([]string, error) { return loader.ObjectKeys(name) }

// ObjectKeys 返回指定嵌套对象的原始键顺序
func (loader *TableLoader) ObjectKeys(name string, path ...string) ([]string, error) {
	if _, err := loader.LoadTable(name); err != nil {
		return nil, err
	}
	loader.mu.RLock()
	defer loader.mu.RUnlock()
	return append([]string{}, loader.orders[name][strings.Join(path, "/")]...), nil
}
func cloneOrders(source map[string][]string) map[string][]string {
	out := map[string][]string{}
	for k, v := range source {
		out[k] = append([]string{}, v...)
	}
	return out
}
func mergeOrdered(dst, src map[string]any, order, srcOrder map[string][]string) {
	for _, key := range srcOrder[""] {
		if key == "_subJsonCount" {
			continue
		}
		if _, ok := dst[key]; ok {
			continue
		}
		dst[key] = cloneTableValue(src[key])
		order[""] = append(order[""], key)
		for path, keys := range srcOrder {
			if path == key || strings.HasPrefix(path, key+"/") {
				order[path] = append([]string{}, keys...)
			}
		}
	}
}
func payloadOrders(data []byte) (map[string][]string, error) {
	var payload json.RawMessage
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, err
		}
		payload = root["json"]
	} else {
		var root []json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, err
		}
		if len(root) <= 5 {
			return nil, ErrTablePayload
		}
		var levelOne []json.RawMessage
		if err := json.Unmarshal(root[5], &levelOne); err != nil || len(levelOne) == 0 {
			return nil, ErrTablePayload
		}
		var levelTwo []json.RawMessage
		if err := json.Unmarshal(levelOne[0], &levelTwo); err != nil || len(levelTwo) <= 2 {
			return nil, ErrTablePayload
		}
		payload = levelTwo[2]
	}
	orders := map[string][]string{}
	var walk func(json.RawMessage, string) error
	walk = func(raw json.RawMessage, path string) error {
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 && raw[0] == '[' {
			var children []json.RawMessage
			if err := json.Unmarshal(raw, &children); err != nil {
				return err
			}
			for i, child := range children {
				if err := walk(child, path+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
			return nil
		}
		if len(raw) == 0 || raw[0] != '{' {
			return nil
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		if _, err := d.Token(); err != nil {
			return err
		}
		keys := []string{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			key, ok := k.(string)
			if !ok {
				return fmt.Errorf("配置对象键不是字符串")
			}
			keys = append(keys, key)
			var child json.RawMessage
			if err = d.Decode(&child); err != nil {
				return err
			}
			next := key
			if path != "" {
				next = path + "/" + key
			}
			if err = walk(child, next); err != nil {
				return err
			}
		}
		orders[path] = keys
		return nil
	}
	if err := walk(payload, ""); err != nil {
		return nil, err
	}
	return orders, nil
}
