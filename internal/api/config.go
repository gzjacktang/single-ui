package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
)

func GetConfig(c *gin.Context) {
	var config database.Config
	database.DB.First(&config)
	c.JSON(http.StatusOK, config)
}

func UpdateConfig(c *gin.Context) {
	var req database.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var prev database.Config
	database.DB.First(&prev)

	path, err := util.NormalizePanelPath(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Path = path

	if req.LogPath != "" && !strings.HasSuffix(req.LogPath, ".log") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日志路径必须以 .log 结尾"})
		return
	}

	req.ID = prev.ID
	req.Username = prev.Username
	req.Password = prev.Password
	req.SecretKey = prev.SecretKey
	req.Port = prev.Port
	req.IPv4 = prev.IPv4
	req.IPv6 = prev.IPv6
	req.StartedAt = prev.StartedAt
	req.Repo = prev.Repo
	req.SubEnable = false
	req.SubPath = ""
	req.SubPort = 0
	req.BoardEnable = false
	database.DB.Save(&req)

	util.InitLog(req.LogPath, req.LogLevel, req.LogEnable)

	util.Info("[config] 面板配置已更新")
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func ResetConfig(c *gin.Context) {
	var prev database.Config
	database.DB.First(&prev)

	ipv4, ipv6 := util.GetPublicIPs()
	config := database.Config{
		SecretKey:         prev.SecretKey,
		Username:          prev.Username,
		Password:          prev.Password,
		Port:              prev.Port,
		Path:              prev.Path,
		IPv4:              ipv4,
		IPv6:              ipv6,
		Domain:            prev.Domain,
		SubEnable:         false,
		RulesetAutoUpdate: false,
		LogEnable:         true,
		LogLevel:          "info",
		LogPath:           "data/slinx.log",
		BoardEnable:       false,
		Repo:              "https://github.com/gzjacktang/single-ui",
	}
	config.ID = prev.ID

	database.DB.Save(&config)
	util.Info("[config] 面板配置已重置")
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
