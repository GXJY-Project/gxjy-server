package assets

import (
	"reflect"
	"testing"
)

func TestPayloadOrdersPreservesObjectsInsideRewardArrays(t *testing.T) {
	orders, err := payloadOrders([]byte(`{"json":{"Reward":{"Content":[{"z-first":[{"second":2,"first":1}],"a-second":[]}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]string{"Reward/Content/0": {"z-first", "a-second"}, "Reward/Content/0/z-first/0": {"second", "first"}} {
		if !reflect.DeepEqual(orders[path], want) {
			t.Fatalf("数组内对象顺序不一致：path=%s got=%v want=%v", path, orders[path], want)
		}
	}
}
