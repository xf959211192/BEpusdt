package router

import (
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/v03413/bepusdt/app"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/model"
)

// registerLocalHealth 只向回环请求返回监控所需字段，不返回配置或凭据。
func registerLocalHealth(engine *gin.Engine) {
	started := time.Now().Unix()
	engine.GET("/internal/health", func(ctx *gin.Context) {
		host, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
		if err != nil || !net.ParseIP(host).IsLoopback() {
			ctx.AbortWithStatus(403)
			return
		}
		var pending int64
		if err := model.Db.Model(&model.Order{}).Where("status IN ?", []int{model.OrderStatusWaiting, model.OrderStatusConfirming}).Count(&pending).Error; err != nil {
			ctx.JSON(503, gin.H{"ok": false})
			return
		}
		ctx.JSON(200, gin.H{"ok": true, "version": app.Version, "started_at": started, "pending_orders": pending, "scan": conf.GetStats()})
	})
}
