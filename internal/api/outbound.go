package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/route"
	"github.com/slinxlink/node/internal/util"
)

func GetOutbounds(c *gin.Context) {
	var obs []database.Outbound
	database.DB.Find(&obs)
	c.JSON(http.StatusOK, obs)
}

func SaveOutbound(c *gin.Context) {
	var ob database.Outbound
	if err := c.ShouldBindJSON(&ob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ob.Protocol != "vless" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前仅支持 vless 出站"})
		return
	}
	if ob.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写备注名"})
		return
	}
	if ob.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写落地地址"})
		return
	}
	if ob.Port < 1 || ob.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "端口须在 1-65535 之间"})
		return
	}
	if ob.UUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写 UUID"})
		return
	}
	if ob.Transport != "raw" && ob.Transport != "websocket" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "传输仅支持 raw / websocket"})
		return
	}
	if ob.TLSType != "none" && ob.TLSType != "tls" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "安全仅支持 none / tls"})
		return
	}

	// 备注名即 sing-box 出站 tag，必须唯一且合法
	if msg := util.ValidateTag(ob.Name); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	var count int64
	database.DB.Model(&database.Outbound{}).Where("name = ? AND id != ?", ob.Name, ob.ID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注名已存在（tag 需唯一）"})
		return
	}

	var old database.Outbound
	hadOld := database.DB.First(&old, ob.ID).Error == nil

	if ob.ID == 0 {
		database.DB.Create(&ob)
		util.Info("[outbound] 添加出站: %s", ob.Name)
	} else {
		database.DB.Save(&ob)
		util.Info("[outbound] 更新出站: %s", ob.Name)
		if hadOld && old.Name != ob.Name {
			route.CleanupRule("endpoint", old.Name, ob.Name)
		}
	}

	go core.Default.Apply()
	c.JSON(http.StatusOK, ob)
}

func DeleteOutbound(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var ob database.Outbound
	if database.DB.First(&ob, id).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不存在"})
		return
	}

	route.CleanupRule("endpoint", ob.Name, "")
	database.DB.Delete(&database.Outbound{}, id)
	util.Info("[outbound] 删除出站: %s", ob.Name)

	go core.Default.Apply()
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func ToggleOutbound(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ob database.Outbound
	if database.DB.First(&ob, id).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不存在"})
		return
	}
	ob.Enable = !ob.Enable
	database.DB.Save(&ob)
	util.Info("[outbound] %s出站: %s", map[bool]string{true: "启用", false: "禁用"}[ob.Enable], ob.Name)

	if !ob.Enable {
		route.CleanupRule("endpoint", ob.Name, "")
	}

	go core.Default.Apply()
	c.JSON(http.StatusOK, ob)
}
