package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ReadAssetKey 优先使用环境变量，未设置时读取本地文件
// 沿用原启动脚本的读取规则，仅去除文件末尾的 CR 和 LF
func ReadAssetKey(envName, filePath string) ([]byte, error) {
	if key := os.Getenv(envName); key != "" {
		return []byte(key), nil
	}
	if filePath == "" {
		return nil, fmt.Errorf("缺少资源密钥环境变量 %s", envName)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取资源密钥文件：%w", err)
	}
	key := strings.TrimRight(string(data), "\r\n")
	if key == "" {
		return nil, errors.New("本地资源密钥文件为空")
	}
	return []byte(key), nil
}
