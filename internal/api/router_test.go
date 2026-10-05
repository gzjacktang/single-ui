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
	for _, route := range router.Routes() {
		if strings.Contains(route.Path, "/detect/") {
			t.Fatalf("removed detection route is still registered: %s", route.Path)
		}
		if strings.Contains(route.Path, "/log/slinx") {
			t.Fatalf("legacy log route is still registered: %s", route.Path)
		}
	}
}
