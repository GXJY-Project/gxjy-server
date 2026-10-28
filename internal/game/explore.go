package game

import (
	"fmt"
	"strings"
)

// 场景查询与交互字段来自 Python scene_units / Session.dispatch，详见 docs/explore-protocol.md
// 这里保留原型已确认的地图可见性与序章机关重置，不推断正式探索规则
const (
	menMechanism1 = "SP_0_MenA_KaiGun1_Do1"
	menMechanism2 = "SP_0_MenA_KaiGun2_Do1"
	menOpenCase   = "SP_0_MenA_KaiGun_Do2"
	menBossCase   = "SP_0_Master_ShiLaiMu_Do1"
)

func (r *Runtime) initialSceneID() string {
	return str(or(r.row("BaseConfig", "Initial_ExploreScene")["Config"], "Scene_SP_0"))
}

func (r *Runtime) loadExploreTables() error {
	for _, name := range []string{"ExploreScene", "ExploreObject", "ExploreCaseDo", "ExploreCaseEnd", "TaskBase", "ExploreDPTask"} {
		if err := r.loadQueryTable(name); err != nil {
			return err
		}
	}
	for _, field := range []string{"cases", "tasks"} {
		if _, err := initializationObject(r.state[field], field); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) guideCases() record {
	out := record{}
	for id, value := range obj(r.state["cases"]) {
		if truth(value) {
			out[id] = true
		}
	}
	return out
}

func (r *Runtime) caseDone(id string) bool {
	_, exists := obj(r.state["cases"])[id]
	return id != "" && exists
}

func sceneCaseEnds(row record) []any {
	ends := obj(row["CaseEnd"])
	out := []any{}
	for _, branch := range []string{"End", "Out", "Fight", "Win", "Lose"} {
		out = append(out, arr(ends[branch])...)
	}
	return out
}

func (r *Runtime) exploreObjectStates() (record, error) {
	out, owners := record{}, map[string]string{}
	for _, id := range r.keys["ExploreObject"] {
		for _, cid := range taskIDs(r.row("ExploreObject", id)["Cases"]) {
			if _, exists := owners[cid]; !exists {
				owners[cid] = id
			}
		}
	}
	// 原型遍历存档 case 的插入顺序，同一物件的后一个状态覆盖前一个
	for _, cid := range r.stateObjectKeys("cases") {
		for _, value := range sceneCaseEnds(r.row("ExploreCaseDo", cid)) {
			end := r.row("ExploreCaseEnd", text(value))
			if text(end["Order"]) != "ChangeSpecialState1" {
				continue
			}
			args := obj(end["Value2"])
			id := text(or(args["ObjectID"], owners[cid]))
			if id == "" || args["SpecialState"] == nil {
				continue
			}
			state, err := checkedInteger(args["SpecialState"])
			if err != nil {
				return nil, fmt.Errorf("物件 %s SpecialState 无效: %w", id, err)
			}
			out[id] = state
		}
	}
	return out, nil
}

func (r *Runtime) activeObjectCases() (map[string][]string, error) {
	out := map[string][]string{}
	done := obj(r.state["tasks"])
	for _, tid := range r.keys["TaskBase"] {
		row := r.row("TaskBase", tid)
		if strings.HasPrefix(tid, "_") || len(row) == 0 {
			continue
		}
		entry, err := initializationObject(done[tid], "tasks entry")
		if err != nil {
			return nil, err
		}
		status, err := checkedInteger(entry["status"])
		if err != nil {
			return nil, err
		}
		if status >= 2 {
			continue
		}
		for _, value := range arr(row["ExploreCaseDo"]) {
			binding := obj(value)
			id := text(binding["ObjectID"])
			if id == "" {
				continue
			}
			for _, cid := range taskIDs(binding["CaseDoID"]) {
				if !r.caseDone(cid) && !containsString(out[id], cid) {
					out[id] = append(out[id], cid)
				}
			}
		}
	}
	return out, nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (r *Runtime) exploreCaseFields(row record, extra []string) (record, bool) {
	ids := taskIDs(row["Cases"])
	for _, id := range extra {
		if !containsString(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return record{}, false
	}
	going, status := []any{}, record{}
	for _, id := range ids {
		if !r.caseDone(id) {
			going = append(going, id)
			status[id] = record{"status": true}
		}
	}
	if len(going) == 0 {
		return nil, true
	}
	out := record{"going_case_list": going, "case_do_map": status}
	if truth(row["CaseDoPassword"]) {
		out["case_do_password"] = copyValue(row["CaseDoPassword"])
	}
	return out, false
}

func (r *Runtime) ensureMenDoorState() error {
	cases := obj(r.state["cases"])
	if !truth(cases[menBossCase]) && truth(cases[menMechanism1]) && truth(cases[menMechanism2]) {
		// 原型明确包含的自愈行为：未打 Boss 且两机关已完成时撤销三个标记并保存
		delete(cases, menMechanism1)
		delete(cases, menMechanism2)
		delete(cases, menOpenCase)
		r.stateKeys["cases"] = r.stateObjectKeys("cases")
		r.save()
	}
	return r.err
}

func objectXY(value any) (record, bool) {
	positions := arr(value)
	if len(positions) == 0 {
		return nil, false
	}
	first := positions[0]
	if groups, ok := first.(map[string]any); ok {
		keys := sortedKeys(groups)
		if len(keys) == 0 {
			return nil, false
		}
		first = groups[keys[0]]
	}
	xy := arr(first)
	if len(xy) < 2 {
		return nil, false
	}
	x, ex := sceneCoordinate(xy[0])
	y, ey := sceneCoordinate(xy[1])
	// 与原型 _object_xy 相同：无法解析坐标的配置物件不作为场景单位
	if ex != nil || ey != nil {
		return nil, false
	}
	return record{"x": x, "y": y}, true
}

func sceneCoordinate(value any) (int64, error) {
	switch v := value.(type) {
	case nil, []any, map[string]any:
		return 0, fmt.Errorf("坐标不是数值")
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, fmt.Errorf("坐标字符串为空")
		}
	}
	return checkedInteger(value)
}

func (r *Runtime) scenePlayerPosition(scene record, requested any) (record, any) {
	face := any("Right")
	if position, ok := requested.(map[string]any); ok {
		_, x := position["x"]
		_, y := position["y"]
		if x && y {
			return record{"x": copyValue(position["x"]), "y": copyValue(position["y"])}, face
		}
	}
	if positions := arr(scene["PlayerPos"]); len(positions) > 0 {
		groups := obj(positions[0])
		keys := sortedKeys(groups)
		if len(keys) > 0 {
			entry := obj(groups[keys[0]])
			xy := arr(entry["Pos"])
			if len(xy) >= 2 {
				return record{"x": copyValue(xy[0]), "y": copyValue(xy[1])}, or(entry["Face"], "Right")
			}
		}
	}
	transfer := obj(r.row("ExploreObject", scene["InitialPos"])["Transfer"])
	xy := arr(transfer["Pos"])
	if len(xy) >= 2 {
		return record{"x": copyValue(xy[0]), "y": copyValue(xy[1])}, or(transfer["Face"], "Right")
	}
	return nil, face
}

func (r *Runtime) sceneUnits(id string, requested any) (record, error) {
	scene := r.row("ExploreScene", id)
	if len(scene) == 0 {
		return nil, fmt.Errorf("场景不存在：%s", id)
	}
	units := record{}
	hero := r.leaderHero(r.state["leader"])
	if hero != "" {
		if len(r.row("HeroBase", hero)) == 0 {
			return nil, fmt.Errorf("场景玩家英雄不存在：%s", hero)
		}
		position, face := r.scenePlayerPosition(scene, requested)
		detail := record{"model_id": hero, "face": face}
		if position != nil {
			detail["pos"] = position
		}
		units["1"] = record{"type": "PLAYER", "unit_detail": detail}
	}
	if id == "Scene_SP_0" {
		if err := r.ensureMenDoorState(); err != nil {
			return nil, err
		}
	}
	states, err := r.exploreObjectStates()
	if err != nil {
		return nil, err
	}
	extra, err := r.activeObjectCases()
	if err != nil {
		return nil, err
	}
	for _, oid := range r.keys["ExploreObject"] {
		row := r.row("ExploreObject", oid)
		if text(row["Scene"]) != id || truth(row["IsDel"]) {
			continue
		}
		position, ok := objectXY(row["Position"])
		if !ok {
			continue
		}
		fields, allDone := r.exploreCaseFields(row, extra[oid])
		if allDone {
			continue
		}
		detail := record{"thing_id": oid, "pos": position}
		if state, ok := states[oid]; ok {
			detail["state"] = state
		}
		merge(detail, fields)
		units[oid] = record{"type": "THING", "unit_detail": detail}
	}
	objects := arr(scene["NormalObject"])
	if len(objects) > 6 {
		objects = objects[:6]
	}
	for _, value := range objects {
		oid := str(value)
		if _, exists := units[oid]; exists {
			continue
		}
		if position, ok := objectXY(r.row("ExploreObject", oid)["Position"]); ok {
			units[oid] = record{"type": "THING", "unit_detail": record{"thing_id": oid, "pos": position}}
		}
	}
	return units, nil
}

func (r *Runtime) exploreDP() (record, error) {
	out, done := record{}, obj(r.state["cases"])
	for _, tid := range r.keys["ExploreDPTask"] {
		row := r.row("ExploreDPTask", tid)
		if strings.HasPrefix(tid, "_") || len(row) == 0 {
			continue
		}
		count := int64(0)
		for _, value := range arr(row["CaseDoID"]) {
			if _, exists := done[str(value)]; exists {
				count++
			}
		}
		limit, err := checkedInteger(row["CaseDoNum"])
		if err != nil {
			return nil, fmt.Errorf("探索度 %s 次数无效: %w", tid, err)
		}
		if limit > 0 && count > limit {
			count = limit
		}
		region := text(row["WorldRegion"])
		if out[region] == nil {
			out[region] = record{}
		}
		obj(out[region])[tid] = count
	}
	return record{"explore_dp_map": out}, nil
}

func (r *Runtime) exploreWorld() (record, error) {
	id := str(or(r.state["scene_id"], r.initialSceneID()))
	// 原型先取引导事件快照，scene_units 随后的序章重置不回改本条快照
	guide := r.guideCases()
	transfer := obj(r.row("ExploreObject", r.row("ExploreScene", id)["InitialPos"])["Transfer"])
	var position any
	xy := arr(transfer["Pos"])
	if len(xy) >= 2 {
		position = record{"x": copyValue(xy[0]), "y": copyValue(xy[1])}
	}
	units, err := r.sceneUnits(id, position)
	if err != nil {
		return nil, err
	}
	out := r.worldMap()
	merge(out, record{"scene_id": id, "teleporte_map": record{}, "guide_case_do": guide, "units": units})
	if position != nil {
		out["pos"] = position
	}
	return out, nil
}

func (r *Runtime) finishSceneCase(request record) (record, record, error) {
	id := text(request["case_do_id"])
	row := r.row("ExploreCaseDo", id)
	if id == "" || len(row) == 0 {
		return nil, nil, fmt.Errorf("场景交互事件不存在：%s", id)
	}
	first := !r.caseDone(id)
	ends, rewards, rewardRefs := []any{}, []any{}, []string{}
	if first {
		for _, value := range arr(obj(row["CaseEnd"])["End"]) {
			endID := text(value)
			end := r.row("ExploreCaseEnd", endID)
			if len(end) == 0 {
				return nil, nil, fmt.Errorf("场景事件 %s 引用了不存在的结算：%s", id, endID)
			}
			ends = append(ends, value)
			if text(end["Order"]) == "GetReward" {
				ref := text(end["Value1"])
				if len(r.row("RewardConfig", ref)) == 0 {
					return nil, nil, fmt.Errorf("场景事件奖励配置不存在：%s", ref)
				}
				rewardRefs = append(rewardRefs, ref)
			}
		}
		ensure(r.state, "cases")[id] = true
		r.stateKeys["cases"] = append(r.stateKeys["cases"], id)
		if err := r.syncTasks(ensure(r.state, "tasks")); err != nil {
			return nil, nil, err
		}
		r.save()
		if r.err != nil {
			return nil, nil, r.err
		}
		// 与原型相同，先同步任务，再按更新后的玩家状态展开事件奖励
		for _, ref := range rewardRefs {
			rewards = append(rewards, r.unpackTaskRewards(arr(r.row("RewardConfig", ref)["Content"]), 1, 0, []string{ref, "Content"})...)
			if r.err != nil {
				return nil, nil, r.err
			}
		}
		if len(rewards) > 0 {
			if err := r.applyTaskRewards(rewards); err != nil {
				return nil, nil, err
			}
		}
	}
	out := record{"end_ids": ends, "box_reward": rewards, "guide_case_do": r.guideCases()}
	if !first {
		return out, nil, nil
	}
	sceneID := str(or(r.state["scene_id"], r.initialSceneID()))
	units, err := r.sceneUnits(sceneID, nil)
	if err != nil {
		return nil, nil, err
	}
	out["scene_id"], out["units"] = sceneID, units
	dp, err := r.exploreDP()
	return out, dp, err
}
