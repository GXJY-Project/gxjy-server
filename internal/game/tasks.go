package game

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// loadQueryTable 保留原型查询时加载配置的时机，资源损坏沿用 Go 的显式失败边界
func (r *Runtime) loadQueryTable(name string) error {
	if _, loaded := r.tables[name]; loaded {
		return nil
	}
	table, err := r.catalog.Table(name)
	if err != nil {
		return fmt.Errorf("加载业务配置 %s: %w", name, err)
	}
	keys, err := r.catalog.Keys(name)
	if err != nil {
		return err
	}
	r.tables[name], r.keys[name] = table, keys
	return nil
}

// 章节顺序及活动豁免来自 Python CHAPTER_ORDER / CHAPTER_EXEMPT
func taskChapter(id string) int {
	if strings.HasPrefix(id, "SM99") || strings.HasPrefix(id, "SM05TE3") {
		return -1
	}
	for n, prefix := range []string{"SMSP", "SM01", "SM02", "SM03", "SM04", "SM05"} {
		if strings.HasPrefix(id, prefix) {
			return n
		}
	}
	return -1
}

func taskStatus(done record, id string) int64 { return iv(obj(done[id])["status"]) }

func (r *Runtime) chapterUnlocked(n int, done record) bool {
	for chapter := 0; chapter < n; chapter++ {
		for id, value := range r.tables["TaskBase"] {
			row, ok := value.(map[string]any)
			if ok && !strings.HasPrefix(id, "_") && str(row["Type"]) == "STORYMAIN" && taskChapter(id) == chapter && taskStatus(done, id) < 1 {
				return false
			}
		}
	}
	return true
}

func taskIDs(value any) []string {
	ids := []string{}
	for _, v := range arr(value) {
		if truth(v) {
			ids = append(ids, str(v))
		}
	}
	return ids
}

// taskCondition 对齐 task_cond_met，不放行原型尚不能判定的条件
func (r *Runtime) taskCondition(row record, done record) bool {
	cond, ok := row["Condition"].(map[string]any)
	if !ok {
		return false
	}
	params := or(cond["params"], []any{})
	if _, ok := params.([]any); !ok {
		params = []any{params}
	}
	ids := taskIDs(params)
	switch text(cond["TaskCondition"]) {
	case "TC_Level":
		return integer(or(r.state["level"], 1), 1) >= iv(cond["factor"])
	case "TC_Task", "TC_TaskFinish":
		hits := int64(0)
		for _, id := range ids {
			if taskStatus(done, id) >= 1 {
				hits++
			}
		}
		return len(ids) > 0 && hits >= max(1, integer(or(cond["factor"], 1), 1))
	case "TC_PassStage":
		for _, id := range ids {
			if !truth(obj(obj(r.state["points"])[id])["pass"]) {
				return false
			}
		}
		return len(ids) > 0
	case "TC_Explore_CaseDo", "TC_Explore_TaskTypeCaseDo", "TC_Explore_BattleCaseDo", "TC_Explore_BattleCaseDo_Win", "TC_Explore_TaskTypeCaseDo_Single", "TC_Explore_TaskTypeCaseDo_Start":
		hits := int64(0)
		for _, id := range ids {
			// case_done 判断键存在，包括值为 false 的已有记录
			if _, exists := obj(r.state["cases"])[id]; exists {
				hits++
			}
		}
		return len(ids) > 0 && hits >= max(1, integer(or(cond["factor"], 1), 1))
	case "TC_TeamMember_Count":
		need := int64(0)
		if p := arr(params); len(p) > 0 {
			// 原型条件表达式先判断 params，空列表时 factor 也不会使用
			need = integer(or(cond["factor"], p[0]), 0)
		}
		return int64(len(r.ownedHeroes())) >= need
	}
	return false
}

var taskMainLine = regexp.MustCompile(`^(SM[A-Z0-9]*?)T[A-Za-z0-9]*$`)
var taskOtherLine = regexp.MustCompile(`^(.*?)T\d+$`)

func taskLine(id string) string {
	head := strings.SplitN(id, "_", 2)[0]
	for _, pattern := range []*regexp.Regexp{taskMainLine, taskOtherLine} {
		if match := pattern.FindStringSubmatch(head); match != nil {
			return match[1]
		}
	}
	return head
}

// taskForwardVisible 对齐 _task_forward_visible，链式任务仅下发距离最短的当前节点
func (r *Runtime) taskForwardVisible(done record) map[string]bool {
	lines := map[string][]string{}
	for _, id := range r.keys["TaskBase"] {
		if _, ok := r.tables["TaskBase"][id].(map[string]any); ok && !strings.HasPrefix(id, "_") {
			line := taskLine(id)
			lines[line] = append(lines[line], id)
		}
	}
	allow := map[string]bool{}
	for _, ids := range lines {
		chained := false
		for _, id := range ids {
			chained = chained || truth(r.row("TaskBase", id)["PreTask"])
		}
		if !chained {
			for _, id := range ids {
				allow[id] = true
			}
			continue
		}
		finished := map[string]bool{}
		candidates := []string{}
		for _, id := range ids {
			pres := taskIDs(r.row("TaskBase", id)["PreTask"])
			if taskStatus(done, id) >= 1 {
				blocked := false
				for _, p := range pres {
					blocked = blocked || taskStatus(done, p) == 1
				}
				if !blocked {
					allow[id], finished[id] = true, true
				}
			} else {
				eligible := true
				for _, p := range pres {
					eligible = eligible && taskStatus(done, p) >= 1
				}
				if eligible {
					candidates = append(candidates, id)
				}
			}
		}
		distance := func(id string) int {
			type node struct {
				id    string
				depth int
			}
			queue := []node{{id, 0}}
			seen := map[string]bool{}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				if seen[cur.id] {
					continue
				}
				seen[cur.id] = true
				pres := taskIDs(r.row("TaskBase", cur.id)["PreTask"])
				if len(pres) == 0 {
					return cur.depth + 1
				}
				for _, p := range pres {
					if finished[p] || taskStatus(done, p) >= 1 {
						return cur.depth + 1
					}
					queue = append(queue, node{p, cur.depth + 1})
				}
			}
			return 1 << 20
		}
		sort.Slice(candidates, func(i, j int) bool {
			a, b := distance(candidates[i]), distance(candidates[j])
			return a < b || a == b && candidates[i] < candidates[j]
		})
		if len(candidates) > 0 {
			allow[candidates[0]] = true
		}
	}
	return allow
}

func (r *Runtime) syncTasks(done record) error {
	changedTotal := 0
	for round := 0; round < 6; round++ {
		changed := 0
		for _, id := range r.keys["TaskBase"] {
			row, ok := r.tables["TaskBase"][id].(map[string]any)
			if !ok || strings.HasPrefix(id, "_") || taskStatus(done, id) >= 1 {
				continue
			}
			eligible := true
			for _, p := range taskIDs(row["PreTask"]) {
				eligible = eligible && taskStatus(done, p) >= 1
			}
			if !eligible || !r.chapterUnlocked(taskChapter(id), done) || !r.taskCondition(row, done) {
				continue
			}
			rec := copyValue(obj(done[id])).(record)
			if factor := iv(obj(row["Condition"])["factor"]); factor != 0 {
				rec["process"] = factor
			}
			rec["status"] = 1
			done[id] = rec
			changed++
			rewards := []any{}
			for _, ref := range arr(row["Reward"]) {
				rewards = append(rewards, r.rewardDeep(str(ref))...)
			}
			if r.err != nil {
				return r.err
			}
			if len(rewards) > 0 {
				if err := r.applyTaskRewards(rewards); err != nil {
					return err
				}
			}
			rec["status"] = 2
		}
		changedTotal += changed
		if changed == 0 {
			break
		}
	}
	for id := range r.taskForwardVisible(done) {
		if _, exists := done[id]; !exists {
			done[id] = record{"status": 0, "process": 0, "reset_time": 0}
		}
	}
	// 原型仅在任务推进时保存，单纯登记可见的 status=0 任务不立即落盘
	if changedTotal > 0 {
		r.save()
	}
	return r.err
}

func (r *Runtime) taskPayload() (record, error) {
	if err := r.loadQueryTable("TaskBase"); err != nil {
		return nil, err
	}
	done := ensure(r.state, "tasks")
	if value := r.state["tasks"]; value != nil {
		if _, ok := value.(map[string]any); !ok {
			return nil, errors.New("任务存档不是对象")
		}
	}
	for id, value := range done {
		if !truth(value) {
			continue
		}
		rec, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("任务记录不是对象：%s", id)
		}
		for _, field := range []string{"status", "process", "reset_time"} {
			if _, err := checkedInteger(rec[field]); err != nil {
				return nil, fmt.Errorf("任务记录 %s 的 %s 无效: %w", id, field, err)
			}
		}
	}
	if err := r.syncTasks(done); err != nil {
		return nil, err
	}
	forward := r.taskForwardVisible(done)
	out := record{}
	for _, id := range r.keys["TaskBase"] {
		row := r.row("TaskBase", id)
		kind := text(row["Type"])
		if kind == "" || !forward[id] {
			continue
		}
		if _, ok := row["Reward"].([]any); !ok {
			continue
		}
		cond, ok := row["Condition"].(map[string]any)
		if !ok || cond["factor"] == nil {
			continue
		}
		rec := obj(done[id])
		status := taskStatus(done, id)
		if status < 1 && !r.chapterUnlocked(taskChapter(id), done) {
			status = 3
		}
		if status < 1 {
			for _, p := range taskIDs(row["PreTask"]) {
				if taskStatus(done, p) < 1 {
					status = 3
				}
			}
		}
		guide := int64(1)
		if targets := arr(row["ExploreTaskTarget"]); len(targets) > 1 {
			if process := iv(rec["process"]); process >= int64(len(targets))-1 {
				guide = int64(len(targets))
			} else {
				guide = max(1, process+1)
			}
		}
		ensure(out, kind)[id] = record{"process": iv(rec["process"]), "status": status, "reset_time": iv(rec["reset_time"]), "guide": guide}
	}
	return out, nil
}
