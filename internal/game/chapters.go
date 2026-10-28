package game

import (
	"fmt"
	"sort"
)

func uniqueStars(v any) []any {
	seen := map[int64]bool{}
	for _, n := range arr(v) {
		seen[iv(n)] = true
	}
	values := make([]int64, 0, len(seen))
	for n := range seen {
		values = append(values, n)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	out := []any{}
	for _, n := range values {
		out = append(out, n)
	}
	return out
}
func (r *Runtime) totalStars() int64 {
	n := int64(0)
	for _, rec := range obj(r.state["points"]) {
		n += int64(len(uniqueStars(obj(rec)["star_idxs"])))
	}
	return n
}
func (r *Runtime) condition(v any) bool {
	cond, ok := v.(map[string]any)
	if !ok {
		return false
	}
	for key, val := range cond {
		switch key {
		case "Level":
			need := int64(num(val, 0))
			if need != 0 && integer(or(r.state["level"], 1), 1) < need {
				return false
			}
		case "FightPoint":
			ids, ok := val.([]any)
			if !ok {
				ids = []any{val}
			}
			present, passed := false, false
			for _, id := range ids {
				if truth(id) {
					present = true
					if truth(obj(obj(r.state["points"])[str(id)])["pass"]) {
						passed = true
					}
				}
			}
			if present && !passed {
				return false
			}
		case "FightPointStar", "FightPointStarIdx":
			need := int64(num(val, 0))
			if need != 0 && r.totalStars() < need {
				return false
			}
		}
	}
	return true
}
func (r *Runtime) mapUnlocked(row record) bool {
	if conds, ok := row["UnlockCondition"].([]any); ok && len(conds) > 0 {
		for _, c := range conds {
			if r.condition(c) {
				return true
			}
		}
		return false
	}
	need := int64(num(row["Level"], 0))
	return need == 0 || integer(or(r.state["level"], 1), 1) >= need
}
func (r *Runtime) chapters() record {
	chapters := record{}
	pts := obj(r.state["points"])
	for _, key := range r.keys["FightMap"] {
		row, ok := r.tables["FightMap"][key].(map[string]any)
		if !ok {
			continue
		}
		points := record{}
		ids := append(append([]any{}, arr(row["FightPoint"])...), arr(row["ElitePoint"])...)
		for _, v := range ids {
			pid := str(v)
			fp, rec := r.row("FightPoint", pid), obj(pts[pid])
			stars := uniqueStars(rec["star_idxs"])
			points[pid] = record{"id": pid, "pass": truth(rec["pass"]), "num": iv(rec["num"]), "star_idxs": stars, "stars": len(stars), "type": text(fp["Type"])}
		}
		count := int64(0)
		for _, p := range points {
			count += iv(obj(p)["stars"])
		}
		chapters[key] = record{"id": key, "unlock": r.mapUnlocked(row), "points": points, "star": count}
	}
	return record{"version": 1, "chapters": chapters, "pass_rewards": record{}}
}
func (r *Runtime) patrol() record {
	today := []any{}
	cfg := obj(r.row("BaseConfig", "Point_Initail_Auto_Reward")["Config"])
	for _, v := range arr(cfg["Show"]) {
		row, ok := v.(map[string]any)
		if !ok {
			continue
		}
		item := or(row["item"], row["code"])
		amount := iv(row["amount"])
		if truth(item) && amount != 0 {
			today = append(today, record{"item": item, "amount": amount})
		}
	}
	if len(today) == 0 {
		box := obj(r.row("BaseConfig", "Point_Box_Config")["Config"])
		for _, key := range r.orderedObjectKeys("BaseConfig", "Point_Box_Config", box) {
			row := obj(box[key])
			for _, v := range arr(row["content"]) {
				entry := obj(v)
				if !truth(entry["Reward"]) {
					continue
				}
				for _, v := range arr(r.row("RewardConfig", entry["Reward"])["Content"]) {
					c := obj(v)
					code, amount := c["code"], iv(c["amount"])
					if truth(code) && amount != 0 {
						today = append(today, record{"item": code, "amount": amount})
					}
				}
				if len(today) > 0 {
					break
				}
			}
			if len(today) > 0 {
				break
			}
		}
	}
	info := ensure(r.state, "patrol_auto")
	return record{"today_rewards": today, "tomorrow_rewards": copyValue(today), "today_received": truth(info["today_received"])}
}
func (r *Runtime) orderedObjectKeys(table, row string, m record) []string {
	if keys, ok := r.keys[table+"/"+row+"/Config"]; ok {
		return keys
	}
	if len(m) > 1 {
		r.err = fmt.Errorf("配置来源未提供 %s.%s.Config 的键顺序", table, row)
		return nil
	}
	return sortedKeys(m)
}
