package game

import (
	"sort"
	"strconv"
	"strings"
)

func (r *Runtime) maxLevel() int64 {
	n := int64(0)
	for key := range r.tables["PlayerLevel"] {
		if v, e := strconv.ParseInt(key, 10, 64); e == nil {
			n = max(n, v)
		}
	}
	if n == 0 {
		return 80
	}
	return n
}
func (r *Runtime) level() int64 { return max(1, min(integer(r.state["level"], 1), r.maxLevel())) }
func (r *Runtime) maxStamina() int64 {
	return integer(or(r.row("PlayerLevel", strconv.FormatInt(r.level(), 10))["ActionPointLimit"], 60), 60)
}
func (r *Runtime) recoverStamina() int64 {
	cur := ensure(r.state, "currency")
	have := iv(cur["IC_Stamina"])
	cap := r.maxStamina()
	now, last := r.now(), iv(r.state["stamina_at"])
	if last == 0 || have >= cap {
		r.state["stamina_at"] = now
		return have
	}
	ticks := (now - last) / 360
	if ticks > 0 {
		have += min(cap-have, ticks)
		cur["IC_Stamina"] = have
		r.state["stamina_at"] = last + ticks*360
		r.save()
	}
	return have
}
func (r *Runtime) roleInfo() record {
	now := r.now()
	p := r.state
	leader := p["leader"]
	if len(r.row("LeaderBase", leader)) == 0 {
		leader = "LGirl"
	}
	info := record{
		"result": 0, "id": p["role_id"], "uid": p["role_id"],
		"name": p["role_name"], "role_name": p["role_name"], "level": r.level(), "exp": p["exp"],
		"exterior": record{
			"sex": p["sex"], "head_image": p["head_image"], "head_rect": p["head_rect"], "back_now": p["back_now"],
			"leader": leader, "wife": or(p["wife"], "H002"), "mode": p["mode"], "signature": "", "role_name_number": 1,
			"system_opens": r.unlockIDs(), "address": record{"province": "AnHui", "city": "AnHui_1"},
			"birthday": record{"month": 1, "day": 1},
		},
		"server_timestamp": now, "server_time": now,
	}
	r.recoverStamina()
	cur := ensure(p, "currency")
	if iv(cur["IC_Stamina"]) > r.maxStamina() {
		cur["IC_Stamina"] = r.maxStamina()
		r.save()
	}
	if len(cur) > 0 {
		info["currency"] = copyValue(cur)
	}
	info["stamina_pool"] = record{"storage": r.recoverStamina(), "max_storage": r.maxStamina()}
	return info
}
func (r *Runtime) unlockIDs() record {
	ids := record{}
	level := integer(or(r.state["level"], 1), 1)
	for _, key := range r.keys["UnlockSystem"] {
		row, ok := r.tables["UnlockSystem"][key].(map[string]any)
		if !ok || strings.HasPrefix(key, "_") {
			continue
		}
		conds := arr(row["UnlockCondition"])
		unlocked := len(conds) == 0
		for _, v := range conds {
			c, ok := v.(map[string]any)
			if !ok {
				continue
			}
			ok = true
			if need, has := c["Level"]; has {
				ok = level >= integer(need, 1<<62)
			}
			if ok {
				unlocked = true
				break
			}
		}
		if unlocked {
			ids[key] = true
		}
	}
	return ids
}
func (r *Runtime) worldMap() record {
	out := record{}
	for _, pair := range [][2]string{{"ExploreWorldMap", "unlock_world_map"}, {"ExploreWorldRegion", "unlock_region_map"}, {"ExploreWorldMain", "unlock_main_map"}} {
		values := record{}
		for _, key := range r.keys[pair[0]] {
			row, ok := r.tables[pair[0]][key].(map[string]any)
			if !ok || strings.HasPrefix(key, "_") {
				continue
			}
			unlock := true
			if pair[0] == "ExploreWorldRegion" {
				if str(row["Type"]) == "Story" {
					maps, ok := row["ExploreMap"].([]any)
					unlock = ok && len(maps) > 0 && iv(row["UnlockShow"]) == 1
				}
			} else if pair[0] == "ExploreWorldMain" {
				conds := row["UnlockCondition"]
				if conds == nil {
					conds = row["OpenCondition"]
				}
				if a, ok := conds.([]any); ok && len(a) > 0 {
					unlock = false
					for _, v := range a {
						c, ok := v.(map[string]any)
						if !ok {
							continue
						}
						level := integer(or(r.state["level"], 1), 1)
						if need := int64(num(c["Level"], 0)); need == 0 || level >= need {
							unlock = true
							break
						}
					}
				}
			}
			values[key] = unlock
		}
		out[pair[1]] = values
	}
	return out
}
func (r *Runtime) summit() record {
	base := r.row("SummitBase", "Summit_1")
	s := obj(r.state["summit"])
	return record{"result": 0, "summitBaseId": str(or(base["Id"], "Summit_1")), "stage": 0, "actInfo": record{"isActOpen": false, "openTime": int64(num(base["OpenTime"], 0)), "closeTime": int64(num(base["CloseTime"], 0))}, "level": integer(or(r.state["level"], 1), 1), "rank": iv(s["rank"]), "score": iv(s["score"]), "heroes": []any{}, "records": []any{}}
}
func (r *Runtime) activities() record {
	entry := func() record {
		return record{"status": 1, "start_time": int64(1730000000), "end_time": int64(4102444800000), "is_new": 0, "unlock": true, "num": 0, "reward_info": record{}}
	}
	daily := entry()
	daily["extra_info"] = record{"cur_day": 1, "all_check_in": 1, "is_checked": 0, "cum_rewards": record{}}
	out := record{"AC_DailyCheckIn": daily}
	for _, key := range r.keys["DrawCard"] {
		aid := r.row("DrawCard", key)["ActivityID"]
		if truth(aid) && len(r.row("Activity", aid)) > 0 {
			if _, ok := out[str(aid)]; !ok {
				out[str(aid)] = entry()
			}
		}
	}
	for _, key := range r.keys["Activity"] {
		row, ok := r.tables["Activity"][key].(map[string]any)
		if ok && strings.ToUpper(str(or(row["ServerType"], or(row["Type"], "")))) == "SHOPBAG" {
			if _, ok := out[key]; !ok {
				out[key] = entry()
			}
		}
	}
	return record{"result": 0, "activities": out}
}
func (r *Runtime) teams() record {
	hid := r.canonicalLeader()
	saved := obj(r.state["teams"])
	slots := []int{1, 2, 3, 4, 5}
	for k := range saved {
		n, e := strconv.Atoi(k)
		if e != nil {
			continue
		}
		found := false
		for _, s := range slots {
			if s == n {
				found = true
			}
		}
		if !found {
			slots = append(slots, n)
		}
	}
	sort.Ints(slots)
	cap := integer(r.row("BaseConfig", "Player_DeckTotalNum")["Config"], 0)
	if cap <= 0 {
		cap = 10
	}
	slotIDs := []any{}
	for n := int64(1); n <= cap; n++ {
		slotIDs = append(slotIDs, strconv.FormatInt(n, 10))
	}
	makeTeam := func(v record, id any, members []any) record {
		lead := r.validLeader(text(v["leader"]))
		if lead == "" {
			lead = hid
		}
		if len(members) > 10 {
			members = members[:10]
		}
		return record{"team_id": id, "leader": or(lead, nil), "members": copyValue(members), "team_name": or(v["team_name"], or(r.state["role_name"], "LocalTeam")), "leader_team_skills": r.teamSkills(lead, false), "slots": copyValue(slotIDs)}
	}
	teams := record{}
	for _, n := range slots {
		key := strconv.Itoa(n)
		v := obj(saved[key])
		teams[key] = makeTeam(v, integer(or(v["team_id"], n), int64(n)), arr(v["members"]))
	}
	first := slots[0]
	if _, ok := teams["0"]; !ok {
		teams["0"] = copyValue(teams["1"])
	}
	used := record{}
	for _, v := range arr(r.row("BaseConfig", "TeamSynchro")["Config"]) {
		used[str(v)] = first
	}
	missing(used, record{"NORMAL": first, "SCENE": first})
	types := map[string]bool{}
	for _, key := range r.keys["ResourcesPoint"] {
		v := r.row("ResourcesPoint", key)["Type"]
		if truth(v) {
			types[str(v)] = true
		}
	}
	for _, key := range r.keys["ResourcesMap"] {
		row := r.row("ResourcesMap", key)
		for _, f := range []string{"Type", "Id"} {
			if truth(row[f]) {
				types[str(row[f])] = true
			}
		}
	}
	for _, v := range strings.Fields("RESOURCE RESOURCES TRIAL TOWER TOWERP INTERLUDE FRIEND_FIGHT friend_fight ARENA REALARENA REAL_ARENA FAIR FAIRPLAY ACTIVITY SUMMIT GVE CLUB BOSS MYSTERY EXPLORE ISEXPLORE NORMAL ELITE SCENE") {
		types[v] = true
	}
	for v := range types {
		if v != "" {
			missing(used, record{v: first, strings.ToUpper(v): first, strings.ToLower(v): first})
		}
	}
	expand := record{}
	for _, key := range sortedKeys(r.tables["ResourcesMap"]) {
		if _, ok := r.tables["ResourcesMap"][key].(map[string]any); !ok || strings.HasPrefix(key, "_") {
			continue
		}
		v := obj(saved[key])
		members := arr(or(v["members"], obj(teams["1"])["members"]))
		expand[key] = makeTeam(v, key, members)
	}
	return record{"result": 0, "teams": teams, "used_team": used, "expand_teams": expand}
}
