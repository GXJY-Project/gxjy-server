package game

import (
	"errors"
	"math"
	"testing"

	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/storage"
)

type taskQuerySource struct {
	err     error
	calls   int
	missing bool
}

func (s *taskQuerySource) LoadTable(name string) (map[string]any, error) {
	if name == "TaskBase" {
		s.calls++
		if s.missing {
			return nil, nil
		}
		return record{}, s.err
	}
	return record{}, nil
}

func TestTaskQueryRejectsMissingTable(t *testing.T) {
	r, err := New(&taskQuerySource{missing: true}, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	messages, err := r.Handle(CommandTasks, nil)
	if !errors.Is(err, hero.ErrTableNotFound) || len(messages) != 0 {
		t.Fatalf("缺表被当成空任务：messages=%v err=%v", messages, err)
	}
}

func TestTaskQueryRejectsCorruptRecords(t *testing.T) {
	for _, value := range []any{true, record{"status": "bad"}, record{"process": "9223372036854775808"}, record{"reset_time": record{"bad": 1}}} {
		r, err := New(&taskQuerySource{}, nil, Options{})
		if err != nil {
			t.Fatal(err)
		}
		r.state["tasks"] = record{"Synthetic": value}
		messages, err := r.Handle(CommandTasks, record{})
		if err == nil || len(messages) != 0 {
			t.Fatalf("损坏任务记录仍成功：messages=%v err=%v", messages, err)
		}
	}
}

func TestTaskRewardOverflowDoesNotAcknowledgeOrOverwriteQuantity(t *testing.T) {
	source := goldenTables{Tables: map[string]record{}, Keys: map[string][]string{}}
	for _, name := range requiredTables {
		source.Tables[name] = record{}
	}
	source.Tables["TaskBase"] = record{"Ready": record{"Type": "DAILY", "Reward": []any{"R"}, "Condition": record{"TaskCondition": "TC_Level", "factor": 1}}}
	source.Keys["TaskBase"] = []string{"Ready"}
	source.Tables["RewardConfig"] = record{"R": record{"Content": []any{record{"type": 2, "code": "SyntheticCurrency", "amount": int64(math.MaxInt64)}}}}
	source.Keys["RewardConfig"] = []string{"R"}
	source.Tables["ItemConfig"] = record{}
	r, err := New(source, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r.state["currency"] = record{"SyntheticCurrency": 1}
	messages, err := r.Handle(CommandTasks, record{})
	if err == nil || len(messages) != 0 {
		t.Fatalf("奖励数量溢出仍成功：messages=%v err=%v", messages, err)
	}
	if iv(obj(r.state["currency"])["SyntheticCurrency"]) != 1 {
		t.Fatal("溢出覆盖了原数量")
	}
	if taskStatus(obj(r.state["tasks"]), "Ready") == 2 {
		t.Fatal("失败任务被标为已领取")
	}
}

func TestTaskQueryPreservesTableErrors(t *testing.T) {
	want := errors.New("synthetic task table error")
	source := &taskQuerySource{err: want}
	r, err := New(source, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 0 {
		t.Fatal("查询前加载了 TaskBase")
	}
	messages, err := r.Handle(CommandTasks, nil)
	if !errors.Is(err, want) || len(messages) != 0 {
		t.Fatalf("配置错误被隐藏：messages=%v err=%v", messages, err)
	}
}

type taskFailingStore struct {
	*storage.JSONStore
	err error
}

func (s *taskFailingStore) Save(value any) error {
	if s.err != nil {
		return s.err
	}
	return s.JSONStore.Save(value)
}

func TestTaskQueryDoesNotAcknowledgeSaveFailure(t *testing.T) {
	store := &taskFailingStore{JSONStore: storage.NewJSONStore(t.TempDir())}
	f := record{"Ready": record{"Type": "DAILY", "Reward": []any{}, "Condition": record{"TaskCondition": "TC_Level", "factor": 1}}}
	source := goldenTables{Tables: map[string]record{}, Keys: map[string][]string{}}
	for _, name := range requiredTables {
		source.Tables[name] = record{}
	}
	source.Tables["TaskBase"], source.Keys["TaskBase"] = f, []string{"Ready"}
	r, err := New(source, store, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("synthetic task save error")
	store.err = want
	messages, err := r.Handle(CommandTasks, record{})
	if !errors.Is(err, want) || len(messages) != 0 {
		t.Fatalf("存档失败仍返回成功：messages=%v err=%v", messages, err)
	}
}
