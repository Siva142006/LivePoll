package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"livepoll/config"
	"livepoll/database"
	"livepoll/routes"
	"livepoll/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if cfg == nil {
		log.Fatal("failed to load configuration")
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoDB, err := database.ConnectMongo(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("failed to connect MongoDB: %v", err)
	}

	redisClient, err := database.ConnectRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect Redis: %v", err)
	}

	hub := websocket.NewHub()
	go hub.Run()
	if err := routes.StartRedisSubscriber(ctx, redisClient, hub); err != nil {
		log.Fatalf("failed to start Redis subscriber: %v", err)
	}

	r := routes.SetupRouter(cfg, mongoDB, redisClient, hub)
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("LivePoll backend listening on %s", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := redisClient.Close(); err != nil {
		log.Printf("redis close error: %v", err)
	}
	if err := mongoDB.Client().Disconnect(shutdownCtx); err != nil {
		log.Printf("mongo close error: %v", err)
	}
	log.Println("LivePoll server stopped cleanly")
}
