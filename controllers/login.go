package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/gin-gonic/gin"
)

const maxLoginBody = 1 << 20

// LoginHandler 处理客户端启动时使用的 HTTP 接口
type LoginHandler struct {
	GateHost string
	HTTPPort int
	WSPort   int
	Token    string
}

// Handle 保留原型的路径分派和 JSON 输出规则
func (handler LoginHandler) Handle(ctx *gin.Context) {
	writer, request := ctx.Writer, ctx.Request
	body, err := io.ReadAll(io.LimitReader(request.Body, maxLoginBody+1))
	if err != nil || len(body) > maxLoginBody {
		http.Error(writer, "请求正文无效", http.StatusBadRequest)
		return
	}
	response := handler.Response(request.URL.Path, body, time.Now())
	ctx.Status(http.StatusOK)
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		global.Logger.Warnf("写入 HTTP 登录响应失败：path:%q error:%v", request.URL.Path, err)
		return
	}
}

// Response 构建登录响应，显式时间用于现有 Python 差分样本验证
func (handler LoginHandler) Response(path string, body []byte, now time.Time) any {
	lowerPath := strings.ToLower(path)
	if lowerPath == "/healthz" {
		return map[string]any{
			"status":      "ok",
			"server_time": now.Unix(),
		}
	}
	if strings.Contains(lowerPath, "version.manifest") || strings.Contains(lowerPath, "project.manifest") {
		return map[string]any{
			"version":           "1.0.0",
			"packageUrl":        "",
			"remoteManifestUrl": "",
			"remoteVersionUrl":  "",
			"assets":            map[string]any{},
			"searchPaths":       []any{},
		}
	}

	unix := now.Unix()
	gateURL := "ws://" + handler.GateHost + ":" + strconv.Itoa(handler.WSPort) + "/game"
	if strings.Contains(lowerPath, "getresource") {
		var request map[string]any
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		if err := decoder.Decode(&request); err != nil {
			// make_login_response 对非法资源版本 JSON 使用空请求，该兼容回退已有差分证据
			request = nil
		}
		version := request["base_version"]
		if !loginTruthy(version) {
			version = request["version"]
		}
		if !loginTruthy(version) {
			version = ""
		}
		return map[string]any{
			"result": 0,
			"notice": map[string]any{
				"content_url": []any{},
				"signature":   "",
				"url":         "",
			},
			"resources": map[string]any{
				"cdn":       []string{"http://" + handler.GateHost + ":" + strconv.Itoa(handler.HTTPPort) + "/gameres/"},
				"version":   version,
				"remoteCfg": map[string]any{},
			},
			"maintain":    false,
			"server_time": unix,
		}
	}
	if strings.Contains(lowerPath, "login") {
		return map[string]any{
			"result":    0,
			"recommend": "1",
			"token":     handler.Token,
			"sec_infos": map[string]any{
				"1": map[string]any{
					"sec_id": "1", "name": "本地服务器", "msg": "",
					"tag": 0, "status": 1, "master_id": "1",
					"open_time": unix, "followers": []string{"1"},
				},
			},
			"role": map[string]any{
				"openid": "local-openid",
				"info_map": map[string]any{
					"1": map[string]any{
						"role_id": "10001", "role_name": "测试玩家",
						"level": 1, "exterior": map[string]any{},
						"online_timestamp": unix,
					},
				},
			},
			"server_time": unix,
		}
	}

	gate := map[string]any{
		"result": 0, "gate_ip": handler.GateHost, "ip": handler.GateHost,
		"port": handler.WSPort, "gate_port": handler.WSPort,
		"type": "ws", "protocol": "ws", "url": gateURL,
		"token": handler.Token, "sec_id": "1", "server_time": unix,
	}
	gate["gate"] = map[string]any{
		"gate_ip": handler.GateHost, "ip": handler.GateHost,
		"port": handler.WSPort, "type": "ws", "protocol": "ws", "url": gateURL,
	}
	gate["sec_info"] = map[string]any{
		"sec_id": "1", "gate_ip": handler.GateHost, "ip": handler.GateHost,
		"port": handler.WSPort, "type": "ws", "protocol": "ws", "url": gateURL,
	}
	gate["info"] = map[string]any{
		"gate_ip": handler.GateHost, "ip": handler.GateHost,
		"port": handler.WSPort, "type": "ws", "protocol": "ws",
	}
	return gate
}

// loginTruthy 保留 Python base_version or version 的值类型和空值行为
func loginTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case json.Number:
		n, err := v.Float64()
		return err == nil && n != 0
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true
	}
}
