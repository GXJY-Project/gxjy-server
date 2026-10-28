package assets

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	googleuuid "github.com/google/uuid"
)

var (
	ErrAssetKeyRequired = errors.New("REF 配置需要 GXJY_ASSET_KEY")
	ErrTableNotFound    = errors.New("配置表不存在")
	ErrTablePayload     = errors.New("配置表没有 json 载荷")
	ErrBundleFormat     = errors.New("Bundle 配置格式无效")
)

// TableLoader 读取 Cocos Creator resources Bundle 中的 JSON 配置表。
type TableLoader struct {
	Root         string
	BundleConfig string
	Key          []byte

	mu       sync.RWMutex
	bundle   *bundleIndex
	tables   map[string]map[string]any
	visiting map[string]bool
	orders   map[string]map[string][]string
}

type bundleIndex struct {
	Paths map[string]json.RawMessage `json:"paths"`
	UUIDs []string                   `json:"uuids"`
}

// NewTableLoader 创建配置加载器。Root 指向 assets/assets 目录。
func NewTableLoader(root, bundleConfig string, key []byte) *TableLoader {
	return &TableLoader{
		Root:         root,
		BundleConfig: bundleConfig,
		Key:          bytes.Clone(key),
		tables:       make(map[string]map[string]any),
		visiting:     make(map[string]bool),
		orders:       make(map[string]map[string][]string),
	}
}

// DecodeCocosUUID 将 22 字符压缩 UUID 还原为标准 UUID。
func DecodeCocosUUID(value string) (string, error) {
	if len(value) != 22 {
		return value, nil
	}

	first, err := hex.DecodeString(value[:2])
	if err != nil {
		return "", fmt.Errorf("%w：压缩 UUID 前缀无效", ErrBundleFormat)
	}
	remainder, err := base64.RawStdEncoding.DecodeString(value[2:])
	if err != nil {
		return "", fmt.Errorf("%w：压缩 UUID 编码无效", ErrBundleFormat)
	}
	raw := append(first, remainder...)
	parsed, err := googleuuid.FromBytes(raw)
	if err != nil {
		return "", fmt.Errorf("%w：%v", ErrBundleFormat, err)
	}
	return parsed.String(), nil
}

// TablePath 从 Bundle 的 paths 和 uuids 映射中寻找配置文件。
func (loader *TableLoader) TablePath(name string) (string, error) {
	index, err := loader.loadBundle()
	if err != nil {
		return "", err
	}
	wanted := "config/data/" + name
	for rawIndex, rawPath := range index.Paths {
		var entry []json.RawMessage
		if err := json.Unmarshal(rawPath, &entry); err != nil || len(entry) == 0 {
			continue
		}
		var path string
		if err := json.Unmarshal(entry[0], &path); err != nil || path != wanted {
			continue
		}
		position, err := strconv.Atoi(rawIndex)
		if err != nil || position < 0 || position >= len(index.UUIDs) {
			return "", fmt.Errorf("%w：表 %s 的 UUID 下标无效", ErrBundleFormat, name)
		}
		uuid, err := DecodeCocosUUID(index.UUIDs[position])
		if err != nil {
			return "", err
		}
		return filepath.Join(loader.Root, "resources", "import", uuid[:2], uuid+".json"), nil
	}
	return "", fmt.Errorf("%w：%s", ErrTableNotFound, name)
}

// LoadTable 加载主表、_subJsonCount 子表和 _subFiles 引用表。
func (loader *TableLoader) LoadTable(name string) (map[string]any, error) {
	if _, err := loader.loadBundle(); err != nil {
		return nil, err
	}
	loader.mu.Lock()
	defer loader.mu.Unlock()
	return loader.loadTableLocked(name)
}

func (loader *TableLoader) loadTableLocked(name string) (map[string]any, error) {
	if cached, ok := loader.tables[name]; ok {
		return cloneMap(cached), nil
	}
	if loader.visiting[name] {
		return nil, fmt.Errorf("%w：%s 出现循环引用", ErrBundleFormat, name)
	}
	loader.visiting[name] = true
	defer delete(loader.visiting, name)

	table, err := loader.loadOne(name)
	if err != nil {
		return nil, err
	}
	result := cloneMap(table)
	order := cloneOrders(loader.orders[name])
	count, err := integerValue(result["_subJsonCount"])
	if err != nil {
		return nil, fmt.Errorf("%w：%s._subJsonCount", ErrBundleFormat, name)
	}
	for index := 1; index <= count; index++ {
		sub, err := loader.loadOne(fmt.Sprintf("%s_sub%d", name, index))
		if err != nil {
			return nil, err
		}
		mergeOrdered(result, sub, order, loader.orders[fmt.Sprintf("%s_sub%d", name, index)])
	}

	if subFiles, ok := result["_subFiles"].(map[string]any); ok {
		for _, subName := range order["_subFiles"] {
			if _, exists := subFiles[subName]; !exists {
				continue
			}
			sub, err := loader.loadTableLocked(subName)
			if err != nil {
				// Python 原版会忽略 _subFiles 的单项加载错误。
				continue
			}
			mergeOrdered(result, sub, order, loader.orders[subName])
		}
	}
	delete(result, "_subFiles")
	delete(result, "_subJsonCount")
	kept := make([]string, 0, len(order[""]))
	for _, key := range order[""] {
		if _, ok := result[key]; ok {
			kept = append(kept, key)
		}
	}
	order[""] = kept
	loader.orders[name] = order
	loader.tables[name] = cloneMap(result)
	return result, nil
}

func (loader *TableLoader) loadBundle() (*bundleIndex, error) {
	loader.mu.RLock()
	cached := loader.bundle
	loader.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	data, err := os.ReadFile(loader.BundleConfig)
	if err != nil {
		return nil, err
	}
	data, err = loader.decodeAsset(data)
	if err != nil {
		return nil, err
	}
	var parsed bundleIndex
	if err := decodeJSON(data, &parsed); err != nil {
		return nil, fmt.Errorf("%w：%v", ErrBundleFormat, err)
	}
	if parsed.Paths == nil || parsed.UUIDs == nil {
		return nil, ErrBundleFormat
	}

	loader.mu.Lock()
	if loader.bundle == nil {
		loader.bundle = &parsed
	}
	cached = loader.bundle
	loader.mu.Unlock()
	return cached, nil
}

func (loader *TableLoader) loadOne(name string) (map[string]any, error) {
	path, err := loader.tablePathWithoutLock(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err = loader.decodeAsset(data)
	if err != nil {
		return nil, fmt.Errorf("读取配置表 %s：%w", name, err)
	}

	var root any
	if err := decodeJSON(data, &root); err != nil {
		return nil, fmt.Errorf("读取配置表 %s：%w", name, err)
	}
	payload := extractTablePayload(root)
	if payload == nil {
		return nil, fmt.Errorf("%w：%s", ErrTablePayload, name)
	}
	orders, err := payloadOrders(data)
	if err != nil {
		return nil, err
	}
	loader.orders[name] = orders
	return payload, nil
}

func (loader *TableLoader) tablePathWithoutLock(name string) (string, error) {
	index := loader.bundle
	if index == nil {
		var err error
		index, err = loader.loadBundle()
		if err != nil {
			return "", err
		}
	}
	wanted := "config/data/" + name
	for rawIndex, rawPath := range index.Paths {
		var entry []json.RawMessage
		if err := json.Unmarshal(rawPath, &entry); err != nil || len(entry) == 0 {
			continue
		}
		var path string
		if err := json.Unmarshal(entry[0], &path); err != nil || path != wanted {
			continue
		}
		position, err := strconv.Atoi(rawIndex)
		if err != nil || position < 0 || position >= len(index.UUIDs) {
			return "", fmt.Errorf("%w：表 %s 的 UUID 下标无效", ErrBundleFormat, name)
		}
		uuid, err := DecodeCocosUUID(index.UUIDs[position])
		if err != nil {
			return "", err
		}
		return filepath.Join(loader.Root, "resources", "import", uuid[:2], uuid+".json"), nil
	}
	return "", fmt.Errorf("%w：%s", ErrTableNotFound, name)
}

func (loader *TableLoader) decodeAsset(data []byte) ([]byte, error) {
	var err error
	if len(data) >= 4 && bytes.Equal(data[:4], []byte("REF:")) {
		if len(loader.Key) == 0 {
			return nil, ErrAssetKeyRequired
		}
		data, err = DecodeREF(data, loader.Key)
		if err != nil {
			return nil, err
		}
	}
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		decompressed, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		data = decompressed
	}
	return data, nil
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func extractTablePayload(root any) map[string]any {
	if object, ok := root.(map[string]any); ok {
		payload, _ := object["json"].(map[string]any)
		return payload
	}
	array, ok := root.([]any)
	if !ok || len(array) <= 5 {
		return nil
	}
	levelOne, ok := array[5].([]any)
	if !ok || len(levelOne) == 0 {
		return nil
	}
	levelTwo, ok := levelOne[0].([]any)
	if !ok || len(levelTwo) <= 2 {
		return nil
	}
	payload, _ := levelTwo[2].(map[string]any)
	return payload
}

func integerValue(value any) (int, error) {
	if value == nil {
		return 0, nil
	}
	switch number := value.(type) {
	case json.Number:
		parsed, err := strconv.Atoi(number.String())
		return parsed, err
	case float64:
		return int(number), nil
	case int:
		return number, nil
	default:
		return 0, fmt.Errorf("不是整数：%T", value)
	}
}

func mergeMissing(destination, source map[string]any) {
	for key, value := range source {
		if key == "_subJsonCount" {
			continue
		}
		if _, exists := destination[key]; !exists {
			destination[key] = value
		}
	}
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = cloneTableValue(value)
	}
	return cloned
}

func cloneTableValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneTableValue(item)
		}
		return out
	default:
		return value
	}
}
