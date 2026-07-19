package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"uptime_ng/internal/config"
	"uptime_ng/internal/engine"
	"uptime_ng/internal/handler"
	"uptime_ng/internal/migration"
	"uptime_ng/internal/model"
	"uptime_ng/internal/router"
)

func Run() {
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if config.AppConfig.JWT.UsesDefaultSecret() {
		log.Println("WARNING: using the default JWT secret; set UPTIME_NG_JWT_SECRET before production use")
	}

	db := openDB()
	defer closeDB(db)

	runMigrations(db)

	r := setupRouter(db)
	startServer(r)
}

func openDB() *gorm.DB {
	dsn := config.AppConfig.Database.DSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	log.Println("Database connected successfully")
	return db
}

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func runMigrations(db *gorm.DB) {
	if err := migration.Apply(db, "migrations"); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	log.Println("Database migrations applied")

	if err := db.AutoMigrate(
		&model.User{},
		&model.Monitor{},
		&model.Heartbeat{},
		&model.StatMinutely{},
		&model.StatHourly{},
		&model.StatDaily{},
		&model.Notification{},
		&model.MonitorNotification{},
		&model.Tag{},
		&model.MonitorTag{},
		&model.Incident{},
		&model.MaintenanceWindow{},
		&model.SLAReport{},
		&model.Setting{},
	); err != nil {
		log.Fatalf("Failed to auto-migrate: %v", err)
	}

	log.Println("Database migration completed")

	seedDefaultAdmin(db)
}

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	hub := handler.NewWSHub()
	go hub.Run()
	sch := engine.NewScheduler(db, hub)
	router.Setup(r, db, hub, sch)

	if err := sch.StartAll(); err != nil {
		log.Printf("Warning: Failed to start monitors: %v", err)
	}
	log.Printf("Scheduler started with %d monitors", sch.RunningCount())

	startStatCleanupCron(db)

	return r
}

func startStatCleanupCron(db *gorm.DB) {
	c := cron.New()
	c.AddFunc("@every 5m", func() {
		var monitors []model.Monitor
		db.Where("active = ?", true).Find(&monitors)
		for _, m := range monitors {
			calc := engine.NewUptimeCalculator(m.ID, db)
			calc.Init()
			calc.CleanupOldData()
		}
		log.Printf("Stat cleanup completed")
	})
	c.Start()
}

func startServer(r *gin.Engine) {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	addr := fmt.Sprintf("%s:%d", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	log.Printf("uptime_ng server starting on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
