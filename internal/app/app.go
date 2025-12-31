package app

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"ponisha-go/internal/config"
	"ponisha-go/internal/repositories"
	"ponisha-go/internal/scheduler"
	"ponisha-go/internal/services/scraping"
)

type App struct {
	Config        *config.Config
	Pool          *pgxpool.Pool
	Repo          repositories.ProjectRepository
	Notifier      scraping.Notifier
	Scrapers      []scraping.SiteScraper
	ScrapeService *scraping.Service
	Scheduler     *scheduler.Scheduler
	Server        *http.Server

	ownsPool bool
}

func (a *App) Start() error {
	if err := a.Scheduler.Start(); err != nil {
		return err
	}

	go func() {
		log.Printf("HTTP server listening on %s", a.Server.Addr)
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.Scheduler.Stop()
	if err := a.Server.Shutdown(ctx); err != nil {
		return err
	}
	if a.ownsPool {
		a.Pool.Close()
	}
	return nil
}
