package main

import (
	"log"

	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/db"
	"github.com/NishanthMolleti/kairos/handlers"
	"github.com/NishanthMolleti/kairos/middleware"
	"github.com/NishanthMolleti/kairos/scheduler"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database := db.Connect(cfg.DatabaseURL)
	db.RunMigrations(database)

	r := gin.Default()
	r.Use(middleware.CORS(cfg.FrontendURL))

	authH := handlers.NewAuthHandler(cfg, database)
	r.GET("/auth/login", authH.Login)
	r.GET("/auth/callback", authH.Callback)
	r.POST("/auth/logout", authH.Logout)

	api := r.Group("/api", middleware.AuthRequired(cfg.JWTSecret))
	{
		userH := handlers.NewUserHandler(database)
		api.GET("/user", userH.GetUser)

		syncH := handlers.NewSyncHandler(database)
		api.POST("/sync", syncH.Sync)

		metricsH := handlers.NewMetricsHandler(database)
		api.GET("/metrics/sleep", metricsH.Sleep)
		api.GET("/metrics/readiness", metricsH.Readiness)
		api.GET("/metrics/activity", metricsH.Activity)
		api.GET("/metrics/hrv", metricsH.HRV)
		api.GET("/metrics/heartrate", metricsH.HeartRate)
		api.GET("/metrics/spo2", metricsH.SpO2)
		api.GET("/metrics/stress", metricsH.Stress)
		api.GET("/metrics/workouts", metricsH.Workouts)
	}

	c := scheduler.Start(database)
	defer c.Stop()

	log.Printf("Kairos backend running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
