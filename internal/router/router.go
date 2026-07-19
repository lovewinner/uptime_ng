package router

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/handler"
	"uptime_ng/internal/middleware"
)

func Setup(r *gin.Engine, db *gorm.DB, hub *handler.WSHub, scheduler handler.MonitorScheduler) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "0.1.0"})
	})

	auth := handler.NewAuthHandler(db)
	settings := handler.NewSettingsHandler(db)

	// Public endpoints (no auth required)
	r.POST("/api/auth/register", auth.Register)
	r.POST("/api/auth/login", auth.Login)
	r.GET("/api/auth/registration-status", settings.GetRegistrationStatus)

	api := r.Group("/api")
	api.Use(middleware.AuthRequired())

	api.GET("/auth/profile", auth.Profile)
	api.GET("/auth/users", middleware.AdminRequired(), auth.ListUsers)
	api.PATCH("/auth/users/:id", middleware.AdminRequired(), auth.UpdateUser)

	// Settings (admin only)
	api.GET("/settings", middleware.AdminRequired(), settings.GetSettings)
	api.PUT("/settings/:key", middleware.AdminRequired(), settings.UpdateSetting)

	monitor := handler.NewMonitorHandler(db, scheduler)
	api.GET("/monitors", monitor.List)
	api.POST("/monitors", monitor.Create)
	api.POST("/monitors/ping-range", monitor.CreatePingRange)
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

	maintenance := handler.NewMaintenanceHandler(db)
	api.GET("/maintenance", maintenance.List)
	api.POST("/maintenance", maintenance.Create)
	api.PUT("/maintenance/:id", maintenance.Update)
	api.DELETE("/maintenance/:id", maintenance.Delete)

	hb := handler.NewHeartbeatHandler(db)
	api.GET("/monitors/:id/beats", hb.GetBeats)
	api.GET("/monitors/:id/beats/important", hb.GetImportantBeats)
	api.GET("/monitors/:id/incidents", hb.GetIncidents)
	api.GET("/monitors/:id/status", hb.GetStatus)
	api.GET("/monitors/status", hb.GetRecentStatus)

	sla := handler.NewSLAHandler(db)
	api.GET("/monitors/:id/uptime", sla.GetUptime)
	api.GET("/monitors/:id/uptime/data", sla.GetUptimeData)
	api.GET("/monitors/:id/uptime/summary", sla.GetUptimeSummary)
	api.GET("/monitors/uptime/overall", sla.GetOverall)

	ie := handler.NewImportExportHandler(db, scheduler)
	api.GET("/monitors/export", ie.ExportMonitors)
	api.POST("/monitors/import/preview", ie.ImportPreview)
	api.POST("/monitors/import", ie.ImportExecute)

	r.GET("/api/ws", middleware.WSAuthRequired(), func(c *gin.Context) {
		hub.HandleWebSocket(c)
	})

	// Static files: serve built Vue frontend
	r.Static("/assets", "./dist/assets")
	r.Static("/favicon.ico", "./dist/favicon.ico")
	r.StaticFile("/favicon.ico", "./dist/favicon.ico")

	r.NoRoute(func(c *gin.Context) {
		// SPA fallback: serve index.html for all non-API routes
		if c.Request.Method == "GET" && !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.File("./dist/index.html")
			return
		}
		c.JSON(404, gin.H{"error": "not found", "code": "not_found"})
	})
}
