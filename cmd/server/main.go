package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/silentpass/silentpass/internal/config"
	"github.com/silentpass/silentpass/internal/database"
	"github.com/silentpass/silentpass/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	deps := &router.Deps{}

	// PostgreSQL
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("WARNING: database unavailable, using in-memory storage: %v", err)
	} else {
		log.Println("connected to PostgreSQL")
		deps.DB = db.Pool
		defer db.Close()
	}

	// Redis
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err == nil {
		rdb := redis.NewClient(redisOpt)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("WARNING: Redis unavailable, using in-memory rate limiter: %v", err)
		} else {
			log.Println("connected to Redis")
			deps.Redis = rdb
			defer rdb.Close()
		}
	}

	r := router.New(cfg, deps)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("SilentPass server starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}
