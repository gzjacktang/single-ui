package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/job"
)

func RefreshRuleset(c *gin.Context) {
	job.RefreshRuleset()
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
