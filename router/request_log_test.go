package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPHandlerLogsRequestsWithoutCredentials(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"resource", http.MethodPost, "/player/getResource", `{"base_version":"1.1.224.1","device_id":"synthetic-body-secret"}`, http.StatusOK},
		{"gate", http.MethodGet, "/player/enter", "", http.StatusOK},
		{"oversized body", http.MethodPost, "/player/login", strings.Repeat("x", (1<<20)+1), http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			captureLogs(t, &logs)
			application := SetupRouter()
			request := httptest.NewRequest(test.method, test.path+"?token=synthetic-query-secret", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer synthetic-header-secret")
			response := httptest.NewRecorder()
			application.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			text := logs.String()
			if !strings.Contains(text, "收到 HTTP 登录请求") || !strings.Contains(text, test.method) || !strings.Contains(text, test.path) {
				t.Fatalf("未记录请求到达：%s", text)
			}
			for _, secret := range []string{"synthetic-query-secret", "synthetic-body-secret", "synthetic-header-secret"} {
				if strings.Contains(text, secret) {
					t.Fatalf("日志包含认证或设备信息：%s", secret)
				}
			}
		})
	}
}
