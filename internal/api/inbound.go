package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/core"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/route"
	"github.com/gzjacktang/single-ui/internal/service"
	"github.com/gzjacktang/single-ui/internal/util"
)

func GetInbounds(c *gin.Context) {
	var inbounds []database.Inbound
	database.DB.Find(&inbounds)
	c.JSON(http.StatusOK, inbounds)
}

func SaveInbound(c *gin.Context) {
	var ib database.Inbound
	if err := c.ShouldBindJSON(&ib); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usedPorts := database.UsedPorts(ib.ID)

	if msg := util.ValidatePort(ib.Port, usedPorts); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	supported := map[string]bool{"vless": true, "vmess": true, "hysteria": true, "trojan": true, "tuic": true, "anytls": true, "shadowsocks": true, "tunnel": true}
	if !supported[ib.Protocol] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的入站协议"})
		return
	}
	if ib.Protocol == "tunnel" {
		ib.TunnelAddress = strings.TrimSpace(ib.TunnelAddress)
		if ib.TunnelAddress == "" || strings.ContainsAny(ib.TunnelAddress, " /\\") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写正确的转发目标地址"})
			return
		}
		if ib.TunnelPort < 1 || ib.TunnelPort > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "转发目标端口必须在 1-65535 之间"})
			return
		}
		if ib.TunnelNetwork != "tcp" && ib.TunnelNetwork != "udp" && ib.TunnelNetwork != "tcp,udp" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "转发网络只能是 TCP、UDP 或 TCP+UDP"})
			return
		}
		ib.TLSType = "none"
		ib.HopEnabled = false
	}
	if ib.Protocol == "shadowsocks" {
		ib.ShadowsocksMethod = util.ShadowsocksMethod(ib.ShadowsocksMethod)
		if !util.SupportedShadowsocksMethod(ib.ShadowsocksMethod) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的 Shadowsocks 加密方式"})
			return
		}
		if ib.ShadowsocksPassword == "" && ib.ID != 0 {
			var existing database.Inbound
			if database.DB.First(&existing, ib.ID).Error == nil && existing.Protocol == "shadowsocks" {
				ib.ShadowsocksPassword = existing.ShadowsocksPassword
			}
		}
		if ib.ShadowsocksMethod == util.Shadowsocks2022Method && ib.ShadowsocksPassword == "" {
			key, err := util.GenerateShadowsocks2022Key()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 Shadowsocks 密钥失败"})
				return
			}
			ib.ShadowsocksPassword = key
		}
		if ib.ShadowsocksMethod == util.Shadowsocks2022Method && !util.ValidShadowsocks2022Key(ib.ShadowsocksPassword) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Shadowsocks 服务端密钥必须是 32 字节 Base64"})
			return
		}
		ib.TLSType = "none"
		ib.Transport = "raw"
		ib.HopEnabled = false
		ib.ObfsType = ""
	} else {
		ib.ShadowsocksMethod = ""
		ib.ShadowsocksPassword = ""
	}
	normalizeInboundSecurity(&ib)

	if ib.ObfsType != "" {
		if ib.ObfsPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写混淆密码"})
			return
		}
		if ib.ObfsType == "gecko" {
			if ib.ObfsMinPacketSize <= 0 || ib.ObfsMaxPacketSize <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "包大小必须大于 0"})
				return
			}
			if ib.ObfsMaxPacketSize <= ib.ObfsMinPacketSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": "最大包大小必须大于最小包大小"})
				return
			}
		}
	}

	if ib.Protocol == "tuic" {
		if ib.TuicAuthTimeout < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "认证超时不能小于 1 秒"})
			return
		}
		if ib.TuicHeartbeat < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "心跳间隔不能小于 1 秒"})
			return
		}
	}

	if ib.HopEnabled {
		if msg := service.ValidateHopPort(ib.HopPort, ib.ID); msg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		if msg := service.ValidateHopInterval(ib.HopInterval); msg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
	}

	if ib.TLSType == "tls" {
		var ids []int
		json.Unmarshal([]byte(ib.Certs), &ids)
		if len(ids) == 0 || ids[0] == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请选择证书"})
			return
		}
		if ib.ECHEnabled {
			if ib.ECHKey == "" || ib.ECHConfig == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "开启 ECH 时，密钥和配置不能为空"})
				return
			}
		}
	}

	if ib.TLSType == "reality" {
		if ib.RealityPrivateKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写私钥"})
			return
		}
		if ib.RealityPublicKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写公钥"})
			return
		}
		if ib.RealityShortIDs == "" || ib.RealityShortIDs == "[]" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写短 ID"})
			return
		}
		if ib.RealityServer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写目标服务器"})
			return
		}
		if ib.RealityServerPort == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写目标端口"})
			return
		}
		if ib.RealityServerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写 SNI"})
			return
		}
	}

	if ib.ID == 0 {
		database.DB.Create(&ib)
		util.Info("[inbound] 添加入站: %s:%d", ib.Protocol, ib.Port)
	} else {
		var old database.Inbound
		database.DB.First(&old, ib.ID)
		database.DB.Save(&ib)
		util.Info("[inbound] 更新入站: %s:%d", ib.Protocol, ib.Port)

		if old.Port != ib.Port {
			route.CleanupRule("inbound", fmt.Sprintf("%d", old.Port), fmt.Sprintf("%d", ib.Port))
		}
	}

	go core.Default.Apply()
	c.JSON(http.StatusOK, ib)
}

func normalizeInboundSecurity(ib *database.Inbound) {
	if ib.TLSType != "tls" {
		ib.ServerName = ""
		ib.CipherSuites = ""
		ib.TLSMinVersion = ""
		ib.TLSMaxVersion = ""
		ib.Insecure = false
		ib.ALPN = ""
		ib.Certs = ""
		ib.ECHEnabled = false
		ib.ECHKey = ""
		ib.ECHConfig = ""
	}
	if ib.Protocol != "vless" {
		ib.Flow = ""
	}
}

func DeleteInbound(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var ib database.Inbound
	database.DB.First(&ib, id)

	// 清理 User 表里关联该入站的引用
	var users []database.User
	database.DB.Find(&users)
	for _, u := range users {
		var ids []int
		json.Unmarshal([]byte(u.Inbounds), &ids)
		newIDs := []int{}
		for _, i := range ids {
			if i != id {
				newIDs = append(newIDs, i)
			}
		}
		updated, _ := json.Marshal(newIDs)
		database.DB.Model(&u).Update("inbounds", string(updated))
	}

	// 清理 Rule 表里关联该入站的引用
	route.CleanupRule("inbound", fmt.Sprintf("%d", ib.Port), "")

	database.DB.Delete(&database.Inbound{}, id)
	util.Info("[inbound] 删除入站: %d", id)
	go core.Default.Apply()
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func ToggleInbound(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ib database.Inbound
	if database.DB.First(&ib, id).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不存在"})
		return
	}
	ib.Enable = !ib.Enable
	database.DB.Save(&ib)
	util.Info("[inbound] %s入站: %s:%d", map[bool]string{true: "启用", false: "禁用"}[ib.Enable], ib.Protocol, ib.Port)

	if !ib.Enable {
		route.CleanupRule("inbound", fmt.Sprintf("%d", ib.Port), "")
	}

	go core.Default.Apply()
	c.JSON(http.StatusOK, ib)
}

func QuickInbound(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
