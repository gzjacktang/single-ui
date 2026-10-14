package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/core"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
)

func GetCore(c *gin.Context) {
	var record database.Core
	database.DB.First(&record)
	c.JSON(http.StatusOK, record)
}

func UpdateCore(c *gin.Context) {
	var cfg database.Core
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if msg := core.Update(cfg); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	util.Info("[core] 核心配置已更新")
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func ResetCore(c *gin.Context) {
	core.Reset()
	util.Info("[core] 核心配置已重置")
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func GetCoreStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": core.Default.Status(),
	})
}

func CheckCoreBinaryUpdate(c *gin.Context) {
	info, err := core.DefaultBinaryUpdater.Check(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func UpdateCoreBinary(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "版本参数无效"})
		return
	}
	info, err := core.DefaultBinaryUpdater.Update(c.Request.Context(), req.Version)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrCoreUpdateBusy) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func StartCore(c *gin.Context) {
	if err := core.Default.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func StopCore(c *gin.Context) {
	if err := core.Default.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func RestartCore(c *gin.Context) {
	if err := core.Default.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func GetCoreConfig(c *gin.Context) {
	data, err := core.Default.ReadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func GetCoreProcess(c *gin.Context) {
	info, err := core.Default.Process()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}
