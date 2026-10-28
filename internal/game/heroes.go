package game

import "fmt"

func (r *Runtime) heroState(hid string) record {
	hs := ensure(r.state, "hero_state")
	if _, ok := hs[hid]; !ok {
		hs[hid] = record{"level": 1, "exp": 0, "quality": 0, "star": 0, "favor": 0}
		r.stateKeys["hero_state"] = append(r.stateKeys["hero_state"], hid)
	}
	return obj(hs[hid])
}
func (r *Runtime) leaderHero(lid any) string { return text(r.row("LeaderBase", lid)["HeroID"]) }
func (r *Runtime) leaderBase(hid string) string {
	lb := text(r.row("HeroBase", hid)["Leader"])
	if lb != "" && len(r.row("LeaderBase", lb)) > 0 {
		return lb
	}
	return ""
}
func (r *Runtime) validLeader(hid string) string {
	hb := r.row("HeroBase", hid)
	if hid == "" || !truth(hb) || num(hb["IsShow"], 0) != 1 {
		return ""
	}
	if r.leaderBase(hid) != "" {
		return hid
	}
	return ""
}
func (r *Runtime) canonicalLeader() string {
	hid := r.leaderHero(r.state["leader"])
	if r.validLeader(hid) != "" {
		return hid
	}
	r.ownedHeroes()
	for _, hid := range r.ownedIDs() {
		if r.validLeader(hid) != "" {
			return hid
		}
	}
	for _, hid := range []string{"H004", "H011", "H012", "H014", "H022", "H033"} {
		if r.validLeader(hid) != "" {
			return hid
		}
	}
	return r.leaderHero("LGirl")
}
func (r *Runtime) ownedIDs() []string {
	ids := []string{}
	add := func(h string) {
		for _, x := range ids {
			if x == h {
				return
			}
		}
		ids = append(ids, h)
	}
	if lead := r.leaderHero(r.state["leader"]); lead != "" {
		add(lead)
	}
	add("H002")
	for _, v := range arr(r.state["heroes"]) {
		add(str(v))
	}
	return ids
}
func (r *Runtime) ownedHeroes() record {
	out := record{}
	for _, hid := range r.ownedIDs() {
		if rec := r.heroRecord(hid); rec != nil {
			out[hid] = rec
		}
	}
	return out
}
func (r *Runtime) leaderRow(hid string) record {
	lid := text(r.row("HeroBase", hid)["Leader"])
	if row, ok := r.tables["LeaderBase"][lid].(map[string]any); ok {
		return row
	}
	return r.row("LeaderBase", "LeaderHero_"+hid)
}

// teamSkills 的两种 id 含义分别保留，英雄侧指向 SkillBase，编队侧指向 LeaderTeamSkill
func (r *Runtime) teamSkills(hid string, heroSide bool) record {
	out := record{}
	lb := r.leaderBase(hid)
	if lb == "" {
		return out
	}
	for _, tid := range arr(r.row("LeaderBase", lb)["LeaderTeamSkill"]) {
		row := r.row("LeaderTeamSkill", tid)
		if len(row) == 0 {
			continue
		}
		ts := row["TeamSkill"]
		valid := truth(ts) && len(r.row("SkillBase", ts)) > 0
		if heroSide {
			if valid {
				out[str(tid)] = record{"id": ts}
			}
		} else {
			entry := record{"id": str(tid)}
			if valid {
				entry["skill"] = ts
			}
			out[str(tid)] = entry
		}
	}
	return out
}
func (r *Runtime) leaderSkills(hid string) record {
	out := record{}
	lb := r.leaderBase(hid)
	if lb == "" {
		return out
	}
	add := func(sid any) {
		if truth(sid) && len(r.row("SkillBase", sid)) > 0 {
			out[str(sid)] = record{"id": sid, "level": 1, "lock": false}
		}
	}
	row := r.row("LeaderBase", lb)
	for _, col := range []string{"NormalSkill", "UltraSkill", "PassiveSkill", "EnemyUltraSkill"} {
		add(row[col])
	}
	star := iv(obj(obj(r.state["hero_state"])[hid])["star"])
	for n := int64(1); n <= 6; n++ {
		if n <= star {
			add(fmt.Sprintf("LeaderSkill_PassiveEx_%s_Star%d", hid, n))
		}
	}
	add(r.row("HeroStar", fmt.Sprintf("HeroStar_%s_%d", hid, star))["LeaderSpecialBonus"])
	for tid := range r.teamSkills(hid, true) {
		add(r.row("LeaderTeamSkill", tid)["TeamSkill"])
	}
	return out
}
func (r *Runtime) awakenSkills(hid string, st record) map[string]bool {
	out := map[string]bool{}
	n := iv(st["awaken"])
	if n <= 0 {
		return out
	}
	row := r.row("HeroAwaken", fmt.Sprintf("HeroAwaken_%s_%d", hid, n))
	for _, bonus := range arr(row["SpecialBonus"]) {
		for _, spe := range arr(r.row("SkillBase", str(bonus))["Special"]) {
			sr := r.row("SkillSpecial", str(spe))
			v := obj(sr["Value"])
			switch text(sr["EffectType"]) {
			case "UnlockSlot":
				if truth(v["slot"]) {
					out[str(v["slot"])] = true
				}
			case "UnlockSkill":
				if truth(v["skill"]) {
					out[str(v["skill"])] = true
				}
			}
		}
	}
	return out
}
func (r *Runtime) skillPayload(sid any, lock bool) record {
	key := text(sid)
	row, ok := r.tables["SkillBase"][key].(map[string]any)
	if !ok {
		return nil
	}
	out := record{"id": key}
	for _, f := range []string{"Name", "Icon", "Desc", "ShortDesc", "Anima", "Type", "Script", "Special", "Attr", "Args", "SuperArmor", "SkillAI", "Tag", "SpecialTag", "CardType"} {
		v := row[f]
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		if a, ok := v.([]any); ok && len(a) == 0 {
			continue
		}
		out[f] = copyValue(v)
	}
	out["lock"] = lock
	return out
}
func (r *Runtime) favor(hid string, st record) (string, int64) {
	cur := text(r.row("HeroBase", hid)["Favor"])
	chain := []string{}
	seen := map[string]bool{}
	for cur != "" && !seen[cur] {
		row, ok := r.tables["HeroFavor"][cur].(map[string]any)
		if !ok {
			break
		}
		seen[cur] = true
		chain = append(chain, cur)
		cur = text(row["NextId"])
	}
	if len(chain) == 0 {
		return "", 0
	}
	n := integer(or(st["favor"], 0), 0)
	n = max(0, min(n, int64(len(chain)-1)))
	return chain[n], iv(st["favor_exp"])
}
func (r *Runtime) heroRecord(hid string) record {
	hb := r.row("HeroBase", hid)
	st := r.heroState(hid)
	if !truth(hb) {
		return nil
	}
	data := arr(hb["Data"])
	surface := any("")
	if len(data) > 0 {
		surface = data[0]
	}
	awake := r.awakenSkills(hid, st)
	skills := record{}
	for _, slot := range []string{"NormalSkill", "ProudSkill", "UltraSkill", "OnstageSkill", "PassiveSkill"} {
		sid := hb[slot]
		if truth(sid) {
			locked := slot == "PassiveSkill" && !awake[str(sid)]
			sp := r.skillPayload(sid, locked)
			if sp != nil {
				sp["level"] = 1
				skills[str(sid)] = sp
			}
		}
	}
	resolve := func(table, prefix, stateKey, fallback string) any {
		id := fmt.Sprintf("%s_%s_%d", prefix, hid, iv(st[stateKey]))
		if len(r.row(table, id)) == 0 {
			return hb[fallback]
		}
		return id
	}
	qid := resolve("HeroQuality", "HeroQuality", "quality", "Quality")
	sid := resolve("HeroStar", "HeroStar", "star", "Star")
	aid := resolve("HeroAwaken", "HeroAwaken", "awaken", "Awaken")
	chain := []any{}
	if truth(st["surface"]) {
		chain = append(chain, st["surface"])
	}
	if iv(st["awaken"]) > 0 {
		if len(data) > 1 {
			chain = append(chain, data[1])
		} else if len(data) == 1 {
			r.err = fmt.Errorf("英雄 %s 的觉醒外观配置缺少第二项", hid)
		}
	}
	chain = append(chain, surface)
	chosen := surface
	for _, cand := range chain {
		if truth(cand) && len(r.row("HeroData", cand)) > 0 {
			chosen = cand
			break
		}
	}
	surfaces := record{}
	for _, d := range data {
		if truth(d) && len(r.row("HeroData", d)) > 0 {
			surfaces[str(d)] = str(d)
		}
	}
	lb := r.leaderRow(hid)
	ls := any(nil)
	ld := arr(lb["Data"])
	if len(ld) > 0 {
		ls = ld[0]
	}
	if !truth(ls) || len(r.row("HeroData", ls)) == 0 {
		ls = chosen
	}
	base := record{"quality_id": qid, "rarity": hb["Rarity"], "star": record{"star_id": sid, "star_level": iv(st["star"]) + 1}, "hero_model": record{"hero_surface": chosen, "leader_surface": ls}}
	if truth(aid) {
		base["awaken_id"] = aid
	}
	missing(skills, r.leaderSkills(hid))
	fid, fexp := r.favor(hid, st)
	return record{"level": integer(or(st["level"], 1), 1), "exp": iv(st["exp"]), "cost": or(hb["FirstCost"], 0), "base_attr": base, "hero_surfaces": surfaces, "favor_id": fid, "favor_exp": fexp, "leader_attrs": r.leaderAttrs(hid), "skills": record{"hero_skills": skills, "leader_skills": r.leaderSkills(hid), "leader_team_skills": r.teamSkills(hid, true)}}
}
func (r *Runtime) heroAttr(hid, att string) int64 {
	hb := r.row("HeroBase", hid)
	st := r.heroState(hid)
	level := integer(or(st["level"], 1), 1)
	quality, star := iv(st["quality"]), iv(st["star"])
	v := 0.0
	for q := int64(0); q <= quality; q++ {
		qr := r.row("HeroQuality", fmt.Sprintf("HeroQuality_%s_%d", hid, q))
		if len(qr) == 0 {
			break
		}
		sr := r.row("HeroStar", fmt.Sprintf("HeroStar_%s_%d", hid, star))
		v += num(qr[att+"QualityBase"], 0) + num(sr[att+"StarBase"], 0) + (num(qr[att+"QualityStep"], 0)+num(sr[att+"StarStep"], 0))*float64(level)
	}
	rf := obj(r.row("BaseConfig", "Hero_RarityFactor")["Config"])
	rarity := num(obj(rf[text(hb["Rarity"])])[att], 1)
	cost := num(r.row("HeroCostAttr", str(hb["FirstCost"]))["Base"+att], 1)
	return int64(num(hb[att], 0) + v*num(hb[att+"Factor"], 1)*rarity*cost)
}
func (r *Runtime) leaderAttrs(hid string) record {
	row := r.leaderRow(hid)
	out := record{}
	if len(row) == 0 {
		return out
	}
	eq := r.equipTotals(hid)
	for _, att := range []string{"Atk", "PhyDef", "MagDef", "Hp"} {
		base := float64(r.heroAttr(hid, att)) + num(eq[att], 0)
		out[att] = int64(base * num(row["Leader"+att+"Factor"], 1))
	}
	return out
}
