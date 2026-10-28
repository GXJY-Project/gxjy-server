package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGameUpgraderAllowsAllOrigins(t *testing.T) {
	origins := []string{
		"",
		"http://127.0.0.1:8080",
		"https://example.com",
		"null",
	}

	for _, origin := range origins {
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/game", nil)
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		if !gameUpgrader.CheckOrigin(request) {
			t.Fatalf("来源被拒绝：%q", origin)
		}
	}
}
