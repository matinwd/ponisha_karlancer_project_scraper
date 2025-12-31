package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ponisha-go/internal/app"
	"ponisha-go/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	builder := app.NewBuilder(&cfg)
	application, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("app build error: %v", err)
	}

	if err := application.Start(); err != nil {
		log.Fatalf("app start error: %v", err)
	}

	waitForShutdown(application)
}

func waitForShutdown(application *app.App) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
