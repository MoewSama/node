package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/route"
	"github.com/slinxlink/node/internal/service"
	"github.com/slinxlink/node/internal/util"
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

	if ib.Protocol == "cloudflared" {
		if ib.CfToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写 Cloudflare Tunnel Token"})
			return
		}
		if ib.CfOrigin != "" {
			// Origin 域名格式校验（去掉可能的端口/协议残留，只留主机名）
			host := strings.TrimPrefix(strings.TrimPrefix(ib.CfOrigin, "https://"), "http://")
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if !util.ValidateDomain(host) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Origin 需为有效域名（CF Public Hostname 的子域）"})
				return
			}
			ib.CfOrigin = host
		}
		if ib.CfProtocol != "" && ib.CfProtocol != "quic" && ib.CfProtocol != "http2" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "protocol 仅支持 quic / http2"})
			return
		}
		if ib.CfEdgeIPVersion != 0 && ib.CfEdgeIPVersion != 4 && ib.CfEdgeIPVersion != 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "edge_ip_version 仅支持 0(自动) / 4 / 6"})
			return
		}
		if ib.CfDatagramVersion != "" && ib.CfDatagramVersion != "v2" && ib.CfDatagramVersion != "v3" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "datagram_version 仅支持 v2 / v3"})
			return
		}
		if ib.CfDatagramVersion != "" && ib.CfProtocol != "quic" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "datagram_version 仅在 protocol 为 quic 时有效"})
			return
		}
		if ib.CfHAConnections < 0 || ib.CfHAConnections > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "HA 连接数须在 0-4 之间（0 表示默认）"})
			return
		}
	}

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

	// Hysteria2 伪装：sing-box 的对象形式必须带 type（file / proxy / string），
	// 参数缺失会生成无法启动的配置（核心校验失败后静默沿用旧配置，表现为伪装不生效），
	// 因此在保存阶段就把非法组合挡掉。
	if ib.Protocol == "hysteria" && ib.MasqueradeEnabled {
		if ib.TLSType != "tls" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hysteria2 伪装依赖 TLS 证书，请将安全类型设为 TLS"})
			return
		}
		switch strings.TrimSpace(ib.MasqueradeType) {
		case "", "default":
			ib.MasqueradeType = "default"
		case "proxy":
			rawURL := strings.TrimSpace(ib.MasqueradeURL)
			u, err := url.Parse(rawURL)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "伪装反向代理地址需为完整 URL，例如 https://www.example.com"})
				return
			}
			ib.MasqueradeURL = rawURL
		case "file":
			directory := strings.TrimSpace(ib.MasqueradePath)
			if directory == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请填写伪装静态文件目录（服务器上的绝对路径）"})
				return
			}
			if !path.IsAbs(directory) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "伪装静态文件目录需为绝对路径，例如 /var/www/html"})
				return
			}
			ib.MasqueradePath = directory
		case "string":
			if ib.MasqueradeCode < 100 || ib.MasqueradeCode > 599 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "伪装自定义内容的状态码需在 100 - 599 之间"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的伪装类型: " + ib.MasqueradeType})
			return
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

	// 清理 Board 表里关联该入站的引用
	database.DB.Where("inbound = ?", id).Update("inbound", 0)

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
