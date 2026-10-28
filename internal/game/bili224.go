package game

import (
	"fmt"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
)

// validateBili224SavedReferences 防止过滤构造响应时静默丢弃旧版本或损坏记录
// 存档修复是显式离线操作，运行时只返回错误，不自动清理玩家数据
func (r *Runtime) validateBili224SavedReferences() error {
	for _, entry := range [][2]string{{"hero_state", "HeroBase"}, {"equip_state", "EquipBase"}, {"tasks", "TaskBase"}, {"bag_items", "ItemConfig"}} {
		records := obj(r.state[entry[0]])
		if len(records) == 0 {
			continue
		}
		if err := r.loadQueryTable(entry[1]); err != nil {
			return err
		}
		for key, value := range records {
			id := key
			if entry[0] != "bag_items" {
				value, ok := value.(map[string]any)
				if !ok {
					return fmt.Errorf("B224 存档 %s 含非对象记录", entry[0])
				}
				if entry[0] == "equip_state" {
					id = text(value["template_id"])
				}
			} else if _, err := checkedInteger(value); err != nil {
				return fmt.Errorf("B224 背包计数无效：%w", err)
			}
			if _, ok := r.tables[entry[1]][id].(map[string]any); !ok {
				return fmt.Errorf("B224 存档 %s 引用无效：%w: %s.%s", entry[0], hero.ErrRowNotFound, entry[1], id)
			}
		}
	}
	return nil
}

// validateBili224Message 在发送前检查已确认的客户端配置引用，避免客户端断言后不发加载事件
// 依据：2026-10-26 22:06:57 的四处客户端 requireRecordById 异常及 B224 原消费函数
func (r *Runtime) validateBili224Message(message Message) error {
	ref := func(table string, value any) error {
		id, ok := value.(string)
		if !ok || id == "" {
			return fmt.Errorf("B224 响应 command=%d 的 %s 引用为空或类型错误", message.Command, table)
		}
		if err := r.loadQueryTable(table); err != nil {
			return fmt.Errorf("B224 响应 command=%d 加载 %s：%w", message.Command, table, err)
		}
		if _, ok := r.tables[table][id].(map[string]any); !ok {
			return fmt.Errorf("B224 响应 command=%d 引用无效：%w: %s.%s", message.Command, hero.ErrRowNotFound, table, id)
		}
		return nil
	}
	payload := message.Payload
	switch message.Command {
	case CommandHeroes, CommandLoginInfo:
		heroes := obj(payload["heros"])
		if message.Command == CommandLoginInfo {
			heroes = obj(obj(payload["Resource"])["heros"])
		}
		for id, value := range heroes {
			if err := ref("HeroBase", id); err != nil {
				return err
			}
			hero := obj(value)
			base := obj(hero["base_attr"])
			for table, value := range (record{"HeroQuality": base["quality_id"], "HeroStar": obj(base["star"])["star_id"]}) {
				if err := ref(table, value); err != nil {
					return err
				}
			}
			for _, value := range obj(base["hero_model"]) {
				if truth(value) {
					if err := ref("HeroData", value); err != nil {
						return err
					}
				}
			}
			for _, group := range []string{"hero_skills", "leader_skills"} {
				for _, skill := range obj(obj(hero["skills"])[group]) {
					if err := ref("SkillBase", obj(skill)["id"]); err != nil {
						return err
					}
				}
			}
		}
	case CommandBagRequest:
		for _, value := range obj(payload["items"]) {
			item := obj(value)
			switch item["item_type"] {
			case "Equip":
				if err := ref("EquipBase", item["template_id"]); err != nil {
					return err
				}
				data := obj(item["data"])
				for table, value := range (record{"EquipExp": data["level_id"], "EquipQuality": data["quality"], "EquipStar": data["star"]}) {
					if err := ref(table, value); err != nil {
						return err
					}
				}
			case "Item":
				if err := ref("ItemConfig", item["template_id"]); err != nil {
					return err
				}
			default:
				return fmt.Errorf("B224 背包含未实现物品类型")
			}
		}
	case CommandTasks:
		for _, group := range obj(payload["tasks"]) {
			for id := range obj(group) {
				if err := ref("TaskBase", id); err != nil {
					return err
				}
			}
		}
	case CommandDrawSystem:
		pools, ok := payload["card_pool"].(orderedInitializationObject)
		if !ok {
			return fmt.Errorf("B224 卡池响应不是有序配置对象")
		}
		for _, id := range pools.keys {
			if err := ref("DrawCard", id); err != nil {
				return err
			}
		}
	}
	return nil
}
