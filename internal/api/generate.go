package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/service"
	"github.com/gzjacktang/single-ui/internal/util"
)

func ScanRealityTargets(c *gin.Context) {
	var request struct {
		Targets string `json:"targets"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	results, err := service.ScanRealityTargets(strings.TrimSpace(request.Targets))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func GeneratePort(c *gin.Context) {
	usedPorts := database.UsedPorts()
	for {
		port := util.GeneratePort()
		if msg := util.ValidatePort(port, usedPorts); msg != "" {
			continue
		}
		c.JSON(http.StatusOK, gin.H{"port": port})
		return
	}
}

func GenerateRealityTarget(c *gin.Context) {
	domain := util.GenerateRealityTarget()
	c.JSON(http.StatusOK, gin.H{"domain": domain})
}

func GenerateRealityKeyPair(c *gin.Context) {
	priv, pub, err := util.GenerateRealityKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"private_key": priv, "public_key": pub})
}

func GenerateShortIDs(c *gin.Context) {
	ids, err := util.GenerateShortIDs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"short_ids": ids})
}

func GenerateToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": util.GenerateString(16)})
}

func GenerateUUID(c *gin.Context) {
	uuid, err := util.GenerateUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"uuid": uuid})
}

func GeneratePassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"password": util.GenerateString(16)})
}

func GenerateWireguardKeyPair(c *gin.Context) {
	priv, pub, err := util.GenerateWireguardKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"private_key": priv, "public_key": pub})
}

func GenerateECHKeyPair(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写域名"})
		return
	}
	key, config, err := util.GenerateECHKeyPair(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ech_key": key, "ech_config": config})
}
