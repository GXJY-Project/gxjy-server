package game

import (
	"fmt"
	"sort"
	"strings"
)

func (r *Runtime) equipAttrs(tpl string, quality, star, level int64) record {
	base := r.row("EquipBase", tpl)
	legal := obj(r.row("BaseConfig", "Equip_Attribute")["Config"])
	level = max(1, level)
	quality = max(0, quality)
	star = max(0, star)
	qr := r.row("EquipQuality", fmt.Sprintf("EQE_%s_%d", tpl, quality))
	if len(qr) == 0 {
		qr = r.row("EquipQuality", text(base["QualityEffect"]))
	}
	sr := r.row("EquipStar", fmt.Sprintf("ESE_%s_%d", tpl, star))
	if len(sr) == 0 {
		sr = r.row("EquipStar", text(base["StarEffect"]))
	}
	rarity := integer(obj(r.row("BaseConfig", "Equip_RarityFactor")["Config"])[str(base["Rarity"])], 1)
	if rarity == 0 {
		rarity = 1
	}
	factor := int64(num(base["AtkFactor"], 1))
	if num(base["AtkFactor"], 1) == 0 {
		factor = 1
	}
	out := record{}
	valid := func(k string) bool { _, ok := legal[k]; return len(legal) == 0 || ok }
	add := func(k string, v int64) {
		if valid(k) {
			out[k] = iv(out[k]) + v
		}
	}
	for name, spec := range obj(qr["QualityAttr"]) {
		if !valid(name) {
			continue
		}
		c0, c1 := 0.0, 0.0
		if a, ok := spec.([]any); ok && len(a) > 0 {
			c0 = num(a[0], 0)
			if len(a) > 1 {
				c1 = num(a[1], 0)
			}
		} else if spec != nil {
			c0 = num(spec, 0)
		}
		v, ok := base[name]
		if !ok {
			v = base["Atk"]
		}
		out[name] = int64((num(v, 0) + c0 + c1*float64(level)) * float64(factor) * float64(rarity))
	}
	for _, row := range []record{qr, sr} {
		switch b := row["AttrBonus"].(type) {
		case string:
			if b != "" {
				sa := r.row("SkillAttr", b)
				types, values := arr(sa["AttrType"]), arr(sa["Value"])
				for i, nm := range types {
					if !valid(str(nm)) {
						continue
					}
					if i >= len(values) {
						r.err = fmt.Errorf("装备属性 %s 的 AttrType 与 Value 长度不一致", b)
						break
					}
					v := values[i]
					if a, ok := v.([]any); ok {
						v = 0
						if len(a) > 0 {
							v = a[0]
						}
					}
					add(str(nm), int64(num(v, 0)))
				}
			}
		case map[string]any:
			for k, v := range b {
				add(k, int64(num(v, 0)))
			}
		case []any:
			for _, item := range b {
				if rec, ok := item.(map[string]any); ok {
					add(str(rec["attr_type"]), int64(num(rec["value"], 0)))
				}
			}
		}
	}
	if _, ok := out["Atk"]; !ok && base["Atk"] != nil && valid("Atk") {
		out["Atk"] = int64(num(base["Atk"], 0) * float64(factor) * float64(rarity))
	}
	return out
}
func (r *Runtime) equipData(inst record) record {
	tpl := text(inst["template_id"])
	level := integer(or(inst["level"], 1), 1)
	quality, star := integer(or(inst["quality_num"], 0), 0), integer(or(inst["star_num"], 0), 0)
	lid := text(inst["level_id"])
	if lid == "" || len(r.row("EquipExp", lid)) == 0 {
		lid = tpl
		for i := int64(0); i < max(0, level-1); i++ {
			next := text(r.row("EquipExp", lid)["NextId"])
			if next == "" || len(r.row("EquipExp", next)) == 0 {
				break
			}
			lid = next
		}
	}
	qid := text(inst["quality"])
	if qid == "" || len(r.row("EquipQuality", qid)) == 0 {
		qid = ""
		if tpl != "" {
			qid = fmt.Sprintf("EQE_%s_%d", tpl, quality)
		}
	}
	sid := text(inst["star"])
	if sid == "" || len(r.row("EquipStar", sid)) == 0 {
		sid = ""
		if tpl != "" {
			sid = fmt.Sprintf("ESE_%s_%d", tpl, star)
		}
	}
	return record{"user": text(inst["user"]), "attrs": r.equipAttrs(tpl, quality, star, level), "locked": truth(inst["locked"]), "exp": iv(inst["exp"]), "level_id": lid, "quality": qid, "star": sid, "enchantment": copyValue(or(inst["enchantment"], record{})), "template_id": tpl}
}
func (r *Runtime) equipTotals(hid string) record {
	out := record{}
	legal := obj(r.row("BaseConfig", "Equip_Attribute")["Config"])
	for _, v := range ensure(r.state, "equip_state") {
		inst, ok := v.(map[string]any)
		if !ok || text(inst["user"]) != hid {
			continue
		}
		for name, v := range obj(r.equipData(inst)["attrs"]) {
			if _, ok := legal[name]; len(legal) > 0 && !ok {
				continue
			}
			out[name] = iv(out[name]) + int64(num(v, 0))
		}
	}
	return out
}
func (r *Runtime) starterIDs() []string {
	ids := []string{}
	for _, tpl := range []string{"Equip_S3_1", "Equip_S3_2", "Equip_S3_3"} {
		if len(r.row("EquipBase", tpl)) > 0 {
			ids = append(ids, tpl)
		}
	}
	if len(ids) > 0 {
		return ids
	}
	for _, k := range r.keys["EquipBase"] {
		row, ok := r.tables["EquipBase"][k].(map[string]any)
		if !ok || strings.HasPrefix(k, "_") {
			continue
		}
		ht := arr(row["HeroType"])
		if len(ht) == 1 && str(ht[0]) == "All" && iv(row["IsShow"]) == 1 {
			ids = append(ids, k)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool {
		return iv(r.row("EquipBase", ids[i])["Order"]) < iv(r.row("EquipBase", ids[j])["Order"])
	})
	if len(ids) > 3 {
		ids = ids[:3]
	}
	if len(ids) == 0 {
		ids = append(ids, "Equip_S6_1")
	}
	return ids
}
func (r *Runtime) ensureStarterEquips() {
	st := ensure(r.state, "equip_state")
	tpls := r.starterIDs()
	owned := r.ownedHeroes()
	made := 0
	for _, hid := range sortedKeys(owned) {
		for slot, tpl := range tpls {
			iid := fmt.Sprintf("EQ%d_%s", slot+1, hid)
			if _, ok := st[iid]; ok {
				continue
			}
			st[iid] = record{"template_id": tpl, "user": hid, "level": 1, "quality_num": 0, "star_num": 0, "exp": 0}
			r.stateKeys["equip_state"] = append(r.stateKeys["equip_state"], iid)
			made++
		}
	}
	if made > 0 {
		r.save()
	}
}
func (r *Runtime) bagItems() record {
	out := record{}
	for iid, v := range ensure(r.state, "equip_state") {
		inst, ok := v.(map[string]any)
		if !ok {
			continue
		}
		out[iid] = record{"count": 1, "template_id": text(inst["template_id"]), "item_type": "Equip", "data": r.equipData(inst)}
	}
	for code, count := range obj(r.state["bag_items"]) {
		n := integer(or(count, 0), 0)
		if n <= 0 {
			continue
		}
		key := code
		if _, ok := out[key]; ok {
			key = "IT_" + key
		}
		out[key] = record{"count": n, "template_id": code, "item_type": "Item", "data": record{"is_new": false}}
	}
	return out
}
