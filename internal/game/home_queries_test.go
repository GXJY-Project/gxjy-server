package game

import (
	"reflect"
	"testing"
)

func TestHomeQueriesMatchPython(t *testing.T) {
	testPathsMatchPython(t, "testdata/home_queries_python.json", nil)
}

func TestHomeQueriesDoNotDiscardNonemptyState(t *testing.T) {
	for _, test := range []struct {
		command uint16
		key     string
	}{
		{CommandMails, "mails"}, {CommandFaceNotices, "notices"},
		{CommandRecommendFriends, "friends"}, {CommandRecommendGuilds, "clubs"},
		{CommandGuildInvites, "member_info"}, {CommandSearchGuild, "club_info"},
		{CommandLastGuilds, "clubs"}, {CommandChapterAutoInfo, "patrol_auto"},
		{CommandCheckMonthCard, "month_card"},
	} {
		r := initQueryRuntime(t)
		r.state[test.key] = "synthetic-corrupt-state"
		before := r.Snapshot()
		messages, err := r.Handle(test.command, record{})
		if err == nil || len(messages) != 0 || !reflect.DeepEqual(before, r.Snapshot()) {
			t.Fatalf("command=%d 隐藏了损坏或不支持的状态：%v", test.command, err)
		}
	}
	r := initQueryRuntime(t)
	if messages, err := r.Handle(CommandSummitRequest, nil); err == nil || len(messages) != 0 {
		t.Fatal("巅峰引用缺失时仍返回成功")
	}
}
