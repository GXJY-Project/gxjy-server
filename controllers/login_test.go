package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	config.InitLog()
	global.Logger.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func loginRouter(handler LoginHandler) *gin.Engine {
	r := gin.New()
	r.Any("/*path", handler.Handle)
	return r
}

func TestLoginGetResourceEchoesVersion(t *testing.T) {
	handler := LoginHandler{GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765, Token: "test-token"}
	request := httptest.NewRequest(http.MethodPost, "/player/getResource", strings.NewReader(`{"base_version":"1.13.3279.1"}`))
	response := httptest.NewRecorder()
	loginRouter(handler).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	resources, ok := body["resources"].(map[string]any)
	if !ok || resources["version"] != "1.13.3279.1" {
		t.Fatalf("resources = %#v", body["resources"])
	}
}

func TestLoginEnterReturnsGate(t *testing.T) {
	handler := LoginHandler{GateHost: "10.0.0.8", HTTPPort: 8080, WSPort: 8765, Token: "test-token"}
	request := httptest.NewRequest(http.MethodPost, "/player/enter", nil)
	response := httptest.NewRecorder()
	loginRouter(handler).ServeHTTP(response, request)
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["url"] != "ws://10.0.0.8:8765/game" || body["token"] != "test-token" {
		t.Fatalf("gate = %#v", body)
	}
}

func TestHealthCheck(t *testing.T) {
	handler := LoginHandler{GateHost: "127.0.0.1", HTTPPort: 8080, WSPort: 8765}
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	loginRouter(handler).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("health = %#v", body)
	}
}

type failedLoginWriter struct{ header http.Header }

func (w failedLoginWriter) Header() http.Header { return w.header }
func (w failedLoginWriter) WriteHeader(int)     {}
func (w failedLoginWriter) Write([]byte) (int, error) {
	return 0, errors.New("synthetic response write failure")
}

func TestLoginResponseWriteFailureIsLogged(t *testing.T) {
	var output bytes.Buffer

	previous := global.Logger.Out
	global.Logger.SetOutput(&output)
	t.Cleanup(func() { global.Logger.SetOutput(previous) })
	handler := LoginHandler{GateHost: "127.0.0.1", WSPort: 8765, HTTPPort: 8080}
	loginRouter(handler).ServeHTTP(failedLoginWriter{header: http.Header{}}, httptest.NewRequest("GET", "/healthz", nil))
	if !strings.Contains(output.String(), "synthetic response write failure") {
		t.Fatal("HTTP 写入错误没有记录")
	}
}
