package game

import "strings"

// board 保留 Python MSG_REQ_ROLE_BOARD / MSG_SAVE_BOARD_CHANGE 的缺省值和字段形状
func (r *Runtime) board() record {
	return record{"wife": copyValue(or(r.state["wife"], "H002")), "back_now": copyValue(r.state["back_now"]), "mode": copyValue(r.state["mode"])}
}

func (r *Runtime) roleBoard() (record, error) {
	if err := r.loadQueryTable("BackGroundHome"); err != nil {
		return nil, err
	}
	backgrounds := record{}
	for _, id := range r.keys["BackGroundHome"] {
		if _, ok := r.tables["BackGroundHome"][id].(map[string]any); ok && !strings.HasPrefix(id, "_") {
			backgrounds[id] = 1
		}
	}
	return record{"board_info": record{"back_all": backgrounds, "back_new": record{}}, "board": r.board()}, nil
}

// saveBoardChange 按原型保存三项外观，mode 的零、false 与缺失/null 分开处理
// 原型只检查英雄 IsShow，不做拥有校验，也不验证背景所有权；本次保留并记录边界
func (r *Runtime) saveBoardChange(payload record) {
	if background := payload["back_id"]; truth(background) {
		r.state["back_now"] = copyValue(background)
	}
	if hero := payload["hero_id"]; truth(hero) {
		row := r.row("HeroBase", hero)
		if len(row) > 0 && integer(or(row["IsShow"], 0), 0) == 1 {
			r.state["wife"] = copyValue(hero)
		}
	}
	if mode := payload["mode"]; mode != nil {
		r.state["mode"] = copyValue(mode)
	}
	r.save()
}
