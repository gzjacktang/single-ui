package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDetectionRoutesAreNotRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, "/private/admin/panel")
	logRoutes := map[string]bool{}
	for _, route := range router.Routes() {
		if strings.Contains(route.Path, "/detect/") {
			t.Fatalf("removed detection route is still registered: %s", route.Path)
		}
		if route.Path == "/private/admin/panel/api/log/sbox" || route.Path == "/private/admin/panel/api/log/slinx" {
			logRoutes[route.Path] = true
		}
	}
	if len(logRoutes) != 2 {
		t.Fatalf("expected new and legacy log routes, got %v", logRoutes)
	}
}

func TestCoreUpdateRoutesStayUnderPanelPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, "/private/admin/panel")
	want := map[string]bool{
		"GET /private/admin/panel/api/core/update/check": false,
		"POST /private/admin/panel/api/core/update":      false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if route.Path == "/api/core/update" || route.Path == "/api/core/update/check" {
			t.Fatalf("core update route escaped the configured panel path: %s", route.Path)
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("missing route %s", route)
		}
	}
	for _, target := range []struct{ method, path string }{
		{http.MethodGet, "/private/admin/panel/api/core/update/check"},
		{http.MethodPost, "/private/admin/panel/api/core/update"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(target.method, target.path, nil)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated %s %s returned %d, want 401", target.method, target.path, response.Code)
		}
	}
}
