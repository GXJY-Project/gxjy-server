package game

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// orderedPlayer 保留英雄、装备和探索事件键顺序，分别用于经验选择和物件状态覆盖
// 保存事件时沿用该顺序，避免重启后改变同一物件最后生效的状态
type orderedPlayer struct {
	values record
	keys   map[string][]string
}

func (p *orderedPlayer) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&p.values); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.keys = map[string][]string{}
	for _, group := range []string{"hero_state", "equip_state", "cases"} {
		if _, ok := p.values[group].(map[string]any); !ok {
			continue
		}
		decoder := json.NewDecoder(bytes.NewReader(raw[group]))
		if _, err := decoder.Token(); err != nil {
			return err
		}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return err
			}
			var value json.RawMessage
			if err := decoder.Decode(&value); err != nil {
				return err
			}
			p.keys[group] = append(p.keys[group], token.(string))
		}
	}
	return nil
}

func (r *Runtime) stateObjectKeys(group string) []string {
	values := obj(r.state[group])
	seen := map[string]bool{}
	keys := []string{}
	for _, key := range r.stateKeys[group] {
		if _, exists := values[key]; exists && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	for _, key := range sortedKeys(values) {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	return keys
}

func validatePlayerShapes(values record) error {
	for _, field := range []string{"currency", "bag_items", "hero_state", "equip_state", "teams", "points", "summit", "tasks", "month_card", "system_opens", "custom", "cases"} {
		if value := values[field]; value != nil {
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("玩家存档字段 %s 不是对象", field)
			}
		}
	}
	if value := values["heroes"]; value != nil {
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("玩家存档字段 heroes 不是数组")
		}
	}
	return nil
}
