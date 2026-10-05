package api

import (
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
