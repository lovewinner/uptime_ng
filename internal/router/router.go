package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/handler"
	"uptime_ng/internal/middleware"
)

func Setup(r *gin.Engine, db *gorm.DB) *handler.WSHub {
	hub := handler.NewWSHub()
	go hub.Run()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "0.1.0"})
	})

	auth := handler.NewAuthHandler(db)

	r.POST("/api/auth/register", auth.Register)
	r.POST("/api/auth/login", auth.Login)

	api := r.Group("/api")
	api.Use(middleware.AuthRequired())

	api.GET("/auth/profile", auth.Profile)
	api.GET("/auth/users", middleware.AdminRequired(), auth.ListUsers)
	api.PATCH("/auth/users/:id", middleware.AdminRequired(), auth.UpdateUser)

	monitor := handler.NewMonitorHandler(db)
	api.GET("/monitors", monitor.List)
	api.POST("/monitors", monitor.Create)
	api.GET("/monitors/:id", monitor.Get)
	api.PUT("/monitors/:id", monitor.Update)
	api.DELETE("/monitors/:id", monitor.Delete)
	api.POST("/monitors/:id/resume", monitor.Resume)
	api.POST("/monitors/:id/pause", monitor.Pause)

	notif := handler.NewNotificationHandler(db)
	api.GET("/notifications", notif.List)
	api.POST("/notifications", notif.Create)
	api.GET("/notifications/:id", notif.Get)
	api.PUT("/notifications/:id", notif.Update)
	api.DELETE("/notifications/:id", notif.Delete)
	api.POST("/notifications/:id/test", notif.Test)

	hb := handler.NewHeartbeatHandler(db)
	api.GET("/monitors/:id/beats", hb.GetBeats)
	api.GET("/monitors/:id/beats/important", hb.GetImportantBeats)
	api.GET("/monitors/:id/incidents", hb.GetIncidents)
	api.GET("/monitors/status", hb.GetRecentStatus)

	sla := handler.NewSLAHandler(db)
	api.GET("/monitors/:id/uptime", sla.GetUptime)
	api.GET("/monitors/:id/uptime/data", sla.GetUptimeData)
	api.GET("/monitors/uptime/overall", sla.GetOverall)

	ie := handler.NewImportExportHandler(db)
	api.GET("/monitors/export", ie.ExportMonitors)
	api.POST("/monitors/import/preview", ie.ImportPreview)
	api.POST("/monitors/import", ie.ImportExecute)

	// WebSocket
	api.GET("/ws", func(c *gin.Context) {
		hub.HandleWebSocket(c)
	})

	// Static files: serve built Vue frontend
	r.Static("/assets", "./dist/assets")
	r.Static("/favicon.ico", "./dist/favicon.ico")
	r.StaticFile("/favicon.ico", "./dist/favicon.ico")

	r.NoRoute(func(c *gin.Context) {
		// SPA fallback: serve index.html for all non-API routes
		if c.Request.Method == "GET" && len(c.Request.URL.Path) > 0 && c.Request.URL.Path[0:5] != "/api/" {
			c.File("./dist/index.html")
			return
		}
		c.JSON(404, gin.H{"error": "not found"})
	})

	return hub
}