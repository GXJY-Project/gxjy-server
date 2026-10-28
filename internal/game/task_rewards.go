package game

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
)

// rewardDeep 对齐 reward_deep，合并同 code 的正数奖励并保留首次出现的类型
// 原型在这里展开概率节点的全部分支，不能擅自改成抽取一个分支
func (r *Runtime) rewardDeep(ref string) []any {
	merged, order := record{}, []string{}
	var walk func(string, int)
	walk = func(key string, depth int) {
		if depth > 4 || key == "" {
			return
		}
		var entries func(any, []string)
		entries = func(content any, path []string) {
			for index, value := range arr(content) {
				e, ok := value.(map[string]any)
				if !ok {
					continue
				}
				code := text(e["code"])
				if code == "" {
					nodePath := append(append([]string{}, path...), strconv.Itoa(index))
					for _, key := range r.rewardObjectKeys(e, nodePath) {
						entries(e[key], append(append([]string{}, nodePath...), key))
					}
					continue
				}
				amount, err := checkedInteger(e["amount"])
				if err != nil {
					r.err = fmt.Errorf("奖励数量无效: %w", err)
					return
				}
				kind, err := checkedInteger(e["type"])
				if err != nil {
					r.err = fmt.Errorf("奖励类型无效: %w", err)
					return
				}
				if amount <= 0 {
					continue
				}
				if kind == 999 || kind == 0 && len(r.row("RewardConfig", code)) > 0 {
					walk(code, depth+1)
					continue
				}
				if rec, exists := merged[code]; exists {
					sum, err := checkedAdd(iv(obj(rec)["amount"]), amount)
					if err != nil {
						r.err = fmt.Errorf("合并奖励数量: %w", err)
						return
					}
					obj(rec)["amount"] = sum
				} else {
					merged[code] = record{"type": kind, "code": code, "amount": amount}
					order = append(order, code)
				}
			}
		}
		entries(r.row("RewardConfig", key)["Content"], []string{key, "Content"})
	}
	walk(ref, 0)
	out := []any{}
	for _, code := range order {
		out = append(out, merged[code])
	}
	return out
}

type rewardPair struct {
	weight float64
	value  any
	path   []string
}

func (r *Runtime) rewardPairs(nodes []any, path []string) []rewardPair {
	pairs := []rewardPair{}
	for index, node := range nodes {
		for _, key := range r.rewardObjectKeys(obj(node), append(append([]string{}, path...), strconv.Itoa(index))) {
			if weight, err := strconv.ParseFloat(key, 64); err == nil {
				pairs = append(pairs, rewardPair{weight: weight, value: obj(node)[key], path: append(append([]string{}, path...), strconv.Itoa(index), key)})
			}
		}
	}
	return pairs
}

func rollTaskReward(pairs []rewardPair) rewardPair {
	if len(pairs) == 0 {
		return rewardPair{}
	}
	percent := true
	total := 0.0
	for _, pair := range pairs {
		percent = percent && pair.weight > 0 && pair.weight <= 1
		total += max(0, pair.weight)
	}
	if percent {
		sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].weight < pairs[j].weight })
		value := rand.Float64()
		for _, pair := range pairs {
			if value <= pair.weight {
				return pair
			}
		}
		return pairs[len(pairs)-1]
	}
	if total <= 0 {
		return pairs[rand.IntN(len(pairs))]
	}
	value := rand.Float64() * total
	for _, pair := range pairs {
		value -= max(0, pair.weight)
		if value < 0 {
			return pair
		}
	}
	return pairs[len(pairs)-1]
}

// unpackTaskRewardRow 对齐 _unpack_reward_row，包括宝箱行的权重和等级分档
func (r *Runtime) unpackTaskRewardRow(ref string, multiplier int64, depth int) []any {
	if depth > 8 {
		return []any{}
	}
	row := r.row("RewardConfig", ref)
	content := arr(row["Content"])
	path := []string{ref, "Content"}
	switch text(row["Type"]) {
	case "Percent", "Weight":
		picked := rollTaskReward(r.rewardPairs(content, path))
		content, path = arr(picked.value), picked.path
	case "PlayerTimes", "PlayerArray", "PlayerLevel", "PlayerCount", "PlayerCountType":
		found, best := false, 0.0
		var selected any
		var selectedPath []string
		for _, pair := range r.rewardPairs(content, path) {
			if pair.weight <= num(or(r.state["level"], 1), 1) && (!found || pair.weight > best) {
				found, best, selected = true, pair.weight, pair.value
				selectedPath = pair.path
			}
		}
		content = arr(selected)
		path = selectedPath
	}
	return r.unpackTaskRewards(content, multiplier, depth+1, path)
}

func (r *Runtime) unpackTaskRewards(content []any, multiplier int64, depth int, path []string) []any {
	out := []any{}
	if depth > 8 {
		return out
	}
	for index, value := range content {
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		code := text(entry["code"])
		if code == "" {
			nodePath := append(append([]string{}, path...), strconv.Itoa(index))
			keys := r.rewardObjectKeys(entry, nodePath)
			pairs := []rewardPair{}
			for _, key := range keys {
				if weight, err := strconv.ParseFloat(key, 64); err == nil {
					pairs = append(pairs, rewardPair{weight: weight, value: entry[key], path: append(append([]string{}, nodePath...), key)})
				}
			}
			picked := rollTaskReward(pairs)
			out = append(out, r.unpackTaskRewards(arr(picked.value), multiplier, depth+1, picked.path)...)
			continue
		}
		kind, err := checkedInteger(entry["type"])
		if err != nil {
			r.err = fmt.Errorf("奖励类型无效: %w", err)
			return nil
		}
		amount, err := checkedInteger(entry["amount"])
		if err != nil {
			r.err = fmt.Errorf("奖励数量无效: %w", err)
			return nil
		}
		mul := multiplier
		if amount > 0 {
			mul, err = checkedMultiply(multiplier, amount)
			if err != nil {
				r.err = fmt.Errorf("展开奖励数量: %w", err)
				return nil
			}
		}
		if kind == 999 {
			out = append(out, r.unpackTaskRewardRow(code, mul, depth+1)...)
			continue
		}
		row := r.row("ItemConfig", code)
		if str(row["Type"]) == "Box" || truth(row["Reward"]) {
			for _, ref := range arr(row["Reward"]) {
				out = append(out, r.unpackTaskRewardRow(str(ref), mul, depth+1)...)
			}
			continue
		}
		copy := copyValue(entry).(record)
		copy["amount"], err = checkedMultiply(amount, multiplier)
		if err != nil {
			r.err = fmt.Errorf("展开奖励数量: %w", err)
			return nil
		}
		out = append(out, copy)
	}
	return out
}

// expRewardRoute 来自 EXP_CODE_ROUTE，类型常量以真实赋值 KEXP_TYPE=1 为准
func expRewardRoute(code string) (string, string) {
	switch code {
	case "0", "IC_PlayerExp":
		return "player", ""
	case "1", "IC_RoleExp", "IC_HeroExp":
		return "hero", ""
	case "2", "3", "IC_EquipExp":
		return "equip", ""
	case "4", "IC_PassCardExp":
		return "currency", "IC_PassCardExp"
	case "5", "IC_HeroCollectExp":
		return "currency", "IC_HeroCollectExp"
	case "IC_Active", "IC_WeekActive", "IC_ClubCoin":
		return "currency", code
	case "IC_ClubContri", "IC_ClubExp":
		return "currency", "IC_ClubCoin"
	}
	return "", ""
}

func (r *Runtime) applyTaskRewards(content []any) error {
	if err := r.loadQueryTable("ItemConfig"); err != nil {
		return err
	}
	content = r.unpackTaskRewards(content, 1, 0, nil)
	if r.err != nil {
		return r.err
	}
	cur := ensure(r.state, "currency")
	bag := []any{}
	for _, value := range content {
		e := obj(value)
		code, amount, kind := text(e["code"]), iv(e["amount"]), iv(e["type"])
		if code == "" || amount == 0 {
			continue
		}
		route, target := "", ""
		if kind == 1 {
			route, target = expRewardRoute(code)
		}
		if kind == 10 {
			target = map[string]string{"0": "IC_Active", "1": "IC_ClubCoin", "2": "IC_ClubCoin", "3": "IC_WeekActive"}[code]
			if target != "" {
				route = "currency"
			} else {
				route, target = expRewardRoute(code)
			}
		}
		switch route {
		case "player":
			exp, err := checkedMultiply(amount, 100)
			if err != nil {
				return fmt.Errorf("奖励主角经验: %w", err)
			}
			r.grantTaskPlayerExp(exp)
		case "hero":
			if err := r.loadQueryTable("HeroLevel"); err != nil {
				return err
			}
			r.grantTaskHeroExp(amount)
		case "equip":
			r.grantTaskEquipExp(amount)
		case "currency":
			r.addTaskAmount(cur, target, amount)
		default:
			if _, exists := cur[code]; exists && kind == 2 {
				r.addTaskAmount(cur, code, amount)
				if r.err != nil {
					return r.err
				}
				continue
			}
			if kind == 2 || kind == 4 || len(r.row("ItemConfig", code)) > 0 || len(r.row("EquipBase", code)) > 0 {
				bag = append(bag, e)
			}
		}
		if r.err != nil {
			return r.err
		}
	}
	if len(bag) > 0 {
		r.grantTaskBagItems(bag)
	}
	if r.err != nil {
		return r.err
	}
	r.save()
	return r.err
}

func (r *Runtime) taskAwardContent(value any) []any {
	if list, ok := value.([]any); ok {
		out := []any{}
		for _, ref := range list {
			out = append(out, r.taskAwardContent(ref)...)
		}
		return out
	}
	return r.unpackTaskRewards(arr(r.row("RewardConfig", value)["Content"]), 1, 0, []string{str(value), "Content"})
}

func (r *Runtime) grantTaskPlayerExp(amount int64) {
	if amount <= 0 {
		return
	}
	r.addTaskAmount(r.state, "exp", amount)
	if r.err != nil {
		return
	}
	ups := 0
	for step := 0; step < 200; step++ {
		level := integer(or(r.state["level"], 1), 1)
		row := r.row("PlayerLevel", str(level))
		need := iv(row["PlayerExp"])
		if need == 0 || level >= r.maxLevel() || iv(r.state["exp"]) < need {
			break
		}
		r.state["exp"] = iv(r.state["exp"]) - need
		r.state["level"] = level + 1
		ups++
		if rewards := r.taskAwardContent(row["LevelReward"]); len(rewards) > 0 {
			if err := r.applyTaskRewards(rewards); err != nil {
				r.err = err
				return
			}
		}
		if gained := iv(row["GetActionPoint"]); gained != 0 {
			cur := ensure(r.state, "currency")
			cur["IC_Stamina"] = min(r.maxStamina(), iv(cur["IC_Stamina"])+gained)
		}
	}
	if ups > 0 && iv(obj(r.state["currency"])["IC_Stamina"]) > r.maxStamina() {
		ensure(r.state, "currency")["IC_Stamina"] = r.maxStamina()
		r.save()
	}
}

func (r *Runtime) grantTaskHeroExp(amount int64) {
	ensure(r.state, "hero_state")
	owned := taskIDs(r.state["heroes"])
	hid := ""
	leader := r.leaderHero(r.state["leader"])
	for _, id := range owned {
		if id == leader {
			hid = leader
			break
		}
	}
	if hid == "" && len(owned) > 0 {
		hid = owned[0]
	}
	if hid == "" {
		hid = "H002"
		if ids := r.stateObjectKeys("hero_state"); len(ids) > 0 {
			hid = ids[0]
		}
	}
	st := r.heroState(hid)
	r.addTaskAmount(st, "exp", amount)
	if r.err != nil {
		return
	}
	for step := 0; step < 200; step++ {
		level := integer(or(st["level"], 1), 1)
		value := r.row("HeroLevel", str(level))["Exp"]
		if values, ok := value.(map[string]any); ok {
			value = values[str(level)]
		}
		need := iv(value)
		if need == 0 || iv(st["exp"]) < need {
			break
		}
		st["exp"], st["level"] = iv(st["exp"])-need, level+1
	}
}

func (r *Runtime) grantTaskEquipExp(amount int64) {
	state := obj(r.state["equip_state"])
	id := ""
	for _, key := range r.stateObjectKeys("equip_state") {
		entry, best := obj(state[key]), obj(state[id])
		level, bestLevel := integer(or(entry["level"], 1), 1), integer(or(best["level"], 1), 1)
		if id == "" || level < bestLevel || level == bestLevel && iv(entry["exp"]) < iv(best["exp"]) {
			id = key
		}
	}
	if id != "" {
		r.addTaskAmount(obj(state[id]), "exp", amount)
	}
}

func (r *Runtime) grantTaskBagItems(entries []any) {
	bag, state := ensure(r.state, "bag_items"), ensure(r.state, "equip_state")
	for _, value := range entries {
		e := obj(value)
		code, amount, kind := text(e["code"]), iv(e["amount"]), iv(e["type"])
		if code == "" || amount <= 0 {
			continue
		}
		if kind == 4 || kind == 0 && len(r.row("EquipBase", code)) > 0 {
			for count := int64(0); count < amount; count++ {
				id := fmt.Sprintf("EQN_%s_%d", code, len(state)+1)
				for {
					if _, exists := state[id]; !exists {
						break
					}
					id += "x"
				}
				state[id] = record{"template_id": code, "user": "", "level": 1, "quality_num": 0, "star_num": 0, "exp": 0}
				r.stateKeys["equip_state"] = append(r.stateKeys["equip_state"], id)
			}
		} else {
			r.addTaskAmount(bag, code, amount)
			if r.err != nil {
				return
			}
		}
	}
}

func (r *Runtime) addTaskAmount(values record, key string, amount int64) {
	if r.err != nil {
		return
	}
	have, err := checkedInteger(values[key])
	if err != nil {
		r.err = fmt.Errorf("已有数量字段 %s 无效: %w", key, err)
		return
	}
	sum, err := checkedAdd(have, amount)
	if err != nil {
		r.err = fmt.Errorf("数量字段 %s 溢出: %w", key, err)
		return
	}
	values[key] = sum
}

func (r *Runtime) rewardObjectKeys(value record, path []string) []string {
	if len(value) < 2 {
		return sortedKeys(value)
	}
	if r.objectKeys == nil || len(path) == 0 {
		r.err = fmt.Errorf("奖励节点缺少原始对象顺序")
		return nil
	}
	keys, err := r.objectKeys("RewardConfig", path...)
	if err != nil {
		r.err = fmt.Errorf("读取奖励对象顺序: %w", err)
		return nil
	}
	if len(keys) != len(value) {
		r.err = fmt.Errorf("奖励对象顺序与内容不一致")
		return nil
	}
	return keys
}
