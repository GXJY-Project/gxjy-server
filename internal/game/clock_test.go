package game

import (
	"testing"
	"time"
)

// 原 make_role_info 只获取一次快照时间，两个时间字段必须一致
func TestRoleTimestampsAgreeAcrossSecondBoundary(t *testing.T) {
	second := int64(1800000000)
	r := &Runtime{state: defaults(), tables: map[string]record{}, keys: map[string][]string{}, options: Options{Now: func() time.Time {
		second++
		return time.Unix(second, 0)
	}}}
	role := r.roleInfo()
	if role["server_timestamp"] != role["server_time"] {
		t.Fatalf("同一角色快照出现不同时间：%v / %v", role["server_timestamp"], role["server_time"])
	}
}
