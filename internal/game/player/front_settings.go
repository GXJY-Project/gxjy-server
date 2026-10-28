package player

import (
	"bytes"
	"encoding/json"
)

// FrontSettings 保存客户端本地标记
type FrontSettings struct {
	Common    map[string]json.RawMessage `json:"common"`
	NeedReset map[string]json.RawMessage `json:"need_reset"`
}

// NewFrontSettings 创建空设置
func NewFrontSettings() FrontSettings {
	return FrontSettings{
		Common:    make(map[string]json.RawMessage),
		NeedReset: make(map[string]json.RawMessage),
	}
}

// Ensure 补齐缺少的设置组
func (settings *FrontSettings) Ensure() {
	if settings.Common == nil {
		settings.Common = make(map[string]json.RawMessage)
	}
	if settings.NeedReset == nil {
		settings.NeedReset = make(map[string]json.RawMessage)
	}
}

// Merge 按 Python 字典 update 规则合并两个设置组
func (settings *FrontSettings) Merge(incoming FrontSettings) {
	settings.Ensure()
	for key, value := range incoming.Common {
		settings.Common[key] = bytes.Clone(value)
	}
	for key, value := range incoming.NeedReset {
		settings.NeedReset[key] = bytes.Clone(value)
	}
}

// Clone 返回可安全读取的副本
func (settings FrontSettings) Clone() FrontSettings {
	cloned := NewFrontSettings()
	cloned.Merge(settings)
	return cloned
}

func (settings *FrontSettings) UnmarshalJSON(data []byte) error {
	var groups map[string]json.RawMessage
	if err := json.Unmarshal(data, &groups); err != nil || groups == nil {
		*settings = NewFrontSettings()
		return nil
	}
	decoded := NewFrontSettings()
	decodeGroup(groups["common"], &decoded.Common)
	decodeGroup(groups["need_reset"], &decoded.NeedReset)
	*settings = decoded
	return nil
}

func decodeGroup(data json.RawMessage, target *map[string]json.RawMessage) {
	if len(data) == 0 {
		return
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err == nil && values != nil {
		*target = values
	}
}
