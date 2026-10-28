package game

import (
	"errors"
	"testing"
	"time"
)

type monthCardTableSource struct {
	table record
	err   error
	calls int
}

func (s *monthCardTableSource) LoadTable(name string) (map[string]any, error) {
	if name == "ShopMonthCard" {
		s.calls++
		return s.table, s.err
	}
	return record{}, nil
}

func TestMonthCardLoadsTableOnlyWhenRequested(t *testing.T) {
	source := &monthCardTableSource{table: record{"SyntheticCard": record{}}}
	r, err := New(source, nil, Options{Now: func() time.Time { return time.Unix(1800000000, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 0 {
		t.Fatal("没有月卡请求时就读取了月卡配置")
	}
	for i := 0; i < 2; i++ {
		messages, err := r.Handle(CommandMonthCard, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 1 || len(obj(messages[0].Payload["month_card"])) != 1 {
			t.Fatalf("月卡请求回包不完整：%v", messages)
		}
	}
	if source.calls != 1 {
		t.Fatalf("重复请求重新加载月卡表：calls=%d", source.calls)
	}
}

func TestMonthCardDoesNotHideTableReadError(t *testing.T) {
	want := errors.New("synthetic table read failure")
	source := &monthCardTableSource{err: want}
	r, err := New(source, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	messages, err := r.Handle(CommandMonthCard, nil)
	if !errors.Is(err, want) || len(messages) != 0 {
		t.Fatalf("月卡配置加载错误被当成成功回包：messages=%v err=%v", messages, err)
	}
}
