package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGameHandlerLogsRequestsBeforeUpgrade(t *testing.T) {
	for _, test := range []struct {
		path   string
		status int
	}{
		{"/game", http.StatusBadRequest},
		{"/wrong-path", http.StatusNotFound},
	} {
		t.Run(test.path, func(t *testing.T) {
			var logs bytes.Buffer
			captureLogs(t, &logs)
			application, err := New(Config{
				HTTPAddress: "127.0.0.1:8080", WSAddress: "127.0.0.1:8765",
				GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765,
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, test.path+"?token=synthetic-query-secret", nil)
			request.Header.Set("Authorization", "Bearer synthetic-header-secret")
			response := httptest.NewRecorder()
			application.GameHandler().ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			text := logs.String()
			if !strings.Contains(text, "收到 WebSocket 请求") || !strings.Contains(text, test.path) {
				t.Fatalf("未记录升级前的请求：%s", text)
			}
			if strings.Contains(text, "synthetic-query-secret") || strings.Contains(text, "synthetic-header-secret") {
				t.Fatal("WebSocket 请求日志包含认证信息")
			}
			if strings.Contains(text, "WebSocket 连接已建立") {
				t.Fatal("失败的升级被记录为成功连接")
			}
		})
	}
}
