package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AlexisHutin/stream-aggregation-service/config"
	"github.com/AlexisHutin/stream-aggregation-service/controllers"
	"github.com/gin-gonic/gin"
)

func main() {
	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/analysis", controllers.AnalysisHandler)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, nil)
	})

	port := fmt.Sprintf(":%d", configs.Port)
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	// Graceful shutdown management
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting server on %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe error: %v", err)
		}
	}()

	// Wait for termination signal
	<-stop
	log.Println("Shutdown signal received, shutting down HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server Shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("Server gracefully stopped")
}
