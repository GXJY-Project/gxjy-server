package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GXJY-Project/gxjy-server/internal/protocol"
)

var errFrontSettingBinary = errors.New("客户端设置不支持二进制值")

func protocolValueJSON(value protocol.Value) (json.RawMessage, error) {
	var converted any
	switch value.Kind {
	case protocol.KindNil:
		converted = nil
	case protocol.KindBool:
		converted = value.Boolean
	case protocol.KindInteger:
		if value.Negative {
			if value.Magnitude > 1<<63 {
				return nil, errors.New("负整数超出 JSON 范围")
			}
			if value.Magnitude == 1<<63 {
				converted = int64(-1 << 63)
			} else {
				converted = -int64(value.Magnitude)
			}
		} else {
			converted = value.Magnitude
		}
	case protocol.KindFloat:
		converted = value.Float64
	case protocol.KindString:
		converted = value.Text
	case protocol.KindBinary:
		return nil, errFrontSettingBinary
	case protocol.KindArray:
		items := make([]json.RawMessage, 0, len(value.Elements))
		for _, item := range value.Elements {
			encoded, err := protocolValueJSON(item)
			if err != nil {
				return nil, err
			}
			items = append(items, encoded)
		}
		converted = items
	case protocol.KindObject:
		items := make(map[string]json.RawMessage, len(value.Fields))
		for _, item := range value.Fields {
			if item.Key.Kind != protocol.KindString {
				return nil, errors.New("设置对象含非文本键")
			}
			encoded, err := protocolValueJSON(item.Value)
			if err != nil {
				return nil, err
			}
			items[item.Key.Text] = encoded
		}
		converted = items
	default:
		return nil, fmt.Errorf("未知协议值类型 %d", value.Kind)
	}
	encoded, err := json.Marshal(converted)
	return json.RawMessage(encoded), err
}

func anyProtocolValue(value any) (protocol.Value, error) {
	switch typed := value.(type) {
	case nil:
		return protocol.Nil(), nil
	case int:
		return protocol.Int(int64(typed)), nil
	case int64:
		return protocol.Int(typed), nil
	case uint64:
		return protocol.Uint(typed), nil
	case float64:
		return protocol.Float(typed), nil
	case bool:
		return protocol.Bool(typed), nil
	case string:
		return protocol.String(typed), nil
	case json.Number:
		text := typed.String()
		if strings.ContainsAny(text, ".eE") {
			number, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return protocol.Value{}, err
			}
			return protocol.Float(number), nil
		}
		if strings.HasPrefix(text, "-") {
			number, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return protocol.Value{}, err
			}
			return protocol.Int(number), nil
		}
		number, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return protocol.Value{}, err
		}
		return protocol.Uint(number), nil
	case []any:
		items := make([]protocol.Value, 0, len(typed))
		for _, item := range typed {
			converted, err := anyProtocolValue(item)
			if err != nil {
				return protocol.Value{}, err
			}
			items = append(items, converted)
		}
		return protocol.Array(items...), nil
	case interface {
		OrderedKeys() []string
		OrderedValue(string) any
	}:
		// B224 卡池默认选择第一项，不能让普通 map 排序改变原型的 normal 优先顺序
		fields := []protocol.Field{}
		for _, key := range typed.OrderedKeys() {
			converted, err := anyProtocolValue(typed.OrderedValue(key))
			if err != nil {
				return protocol.Value{}, err
			}
			fields = append(fields, field(key, converted))
		}
		return protocol.Object(fields...), nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fields := make([]protocol.Field, 0, len(keys))
		for _, key := range keys {
			converted, err := anyProtocolValue(typed[key])
			if err != nil {
				return protocol.Value{}, err
			}
			fields = append(fields, field(key, converted))
		}
		return protocol.Object(fields...), nil
	default:
		return protocol.Value{}, fmt.Errorf("未知 JSON 值类型 %T", value)
	}
}

func decodeRequestJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}
