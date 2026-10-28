package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
)

// monthCards 对齐 Python Session.dispatch 的 MSG_REQ_MONTH_CARD 分支
// IsShow 的零值先经 or 1 回退，不能改成按字段值直接判断是否可购买
func (r *Runtime) monthCards() (record, error) {
	if _, loaded := r.tables["ShopMonthCard"]; !loaded {
		table, err := r.catalog.Table("ShopMonthCard")
		if err != nil && !errors.Is(err, hero.ErrTableNotFound) {
			// 沿用 Go 已确认的配置加载失败边界，不用默认月卡掩盖资源损坏
			return nil, fmt.Errorf("加载业务配置 ShopMonthCard: %w", err)
		}
		r.tables["ShopMonthCard"] = obj(table)
	}
	stored := ensure(r.state, "month_card")
	if value := r.state["month_card"]; value != nil {
		if _, ok := value.(map[string]any); !ok {
			return nil, errors.New("月卡存档不是对象")
		}
	}
	cards := record{}
	for id, value := range r.tables["ShopMonthCard"] {
		row, ok := value.(map[string]any)
		if !ok || strings.HasPrefix(id, "_") {
			continue
		}
		entry := stored[id]
		if truth(entry) {
			if _, ok := entry.(map[string]any); !ok {
				return nil, fmt.Errorf("月卡记录不是对象：%s", id)
			}
		}
		cards[id] = record{
			"unlock":        integer(or(row["IsShow"], 1), 1) != 0,
			"inactive_time": integer(or(obj(entry)["until"], 0), 0),
		}
	}
	if len(cards) == 0 {
		// 这两项来自原型的空表回退，不据此推断正式协议的默认订阅状态
		for _, id := range []string{"ShopMonthCard_01", "ShopMonthCard_Permanent"} {
			cards[id] = record{"unlock": true, "inactive_time": int64(0)}
		}
	}
	return cards, nil
}
