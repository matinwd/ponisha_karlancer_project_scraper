package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ponisha-go/internal/config"
	"ponisha-go/internal/db"
	dbsqlc "ponisha-go/internal/db/sqlc"
	"ponisha-go/internal/httpapi"
	"ponisha-go/internal/providers/karlancer"
	"ponisha-go/internal/providers/ponisha"
	"ponisha-go/internal/repositories"
	sqlcrepo "ponisha-go/internal/repositories/sqlc"
	"ponisha-go/internal/scheduler"
	"ponisha-go/internal/services/scraping"
	"ponisha-go/internal/telegram"
)

type Builder struct {
	cfg          *config.Config
	basePath     string
	ensureSchema bool

	pool     *pgxpool.Pool
	repo     repositories.ProjectRepository
	notifier scraping.Notifier
	scrapers []scraping.SiteScraper
	client   *http.Client

	scheduler *scheduler.Scheduler
	server    *http.Server
}

type BuilderOption func(*Builder)

func NewBuilder(cfg *config.Config, options ...BuilderOption) *Builder {
	builder := &Builder{
		cfg:          cfg,
		ensureSchema: true,
	}
	for _, option := range options {
		option(builder)
	}
	return builder
}

func WithBasePath(basePath string) BuilderOption {
	return func(b *Builder) {
		b.basePath = basePath
	}
}

func WithEnsureSchema(enabled bool) BuilderOption {
	return func(b *Builder) {
		b.ensureSchema = enabled
	}
}

func WithDBPool(pool *pgxpool.Pool) BuilderOption {
	return func(b *Builder) {
		b.pool = pool
	}
}

func WithRepository(repo repositories.ProjectRepository) BuilderOption {
	return func(b *Builder) {
		b.repo = repo
	}
}

func WithNotifier(notifier scraping.Notifier) BuilderOption {
	return func(b *Builder) {
		b.notifier = notifier
	}
}

func WithScrapers(scrapers []scraping.SiteScraper) BuilderOption {
	return func(b *Builder) {
		b.scrapers = scrapers
	}
}

func WithHTTPClient(client *http.Client) BuilderOption {
	return func(b *Builder) {
		b.client = client
	}
}

func WithScheduler(scheduler *scheduler.Scheduler) BuilderOption {
	return func(b *Builder) {
		b.scheduler = scheduler
	}
}

func WithHTTPServer(server *http.Server) BuilderOption {
	return func(b *Builder) {
		b.server = server
	}
}

func (b *Builder) Build(ctx context.Context) (*App, error) {
	if b.cfg == nil {
		return nil, errors.New("config is required")
	}

	basePath := b.basePath
	if basePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		basePath = wd
	}

	app := &App{Config: b.cfg}
	if b.pool == nil {
		pool, err := db.NewPool(ctx, b.cfg.PostgresDSN())
		if err != nil {
			return nil, err
		}
		b.pool = pool
		app.ownsPool = true
	}
	app.Pool = b.pool

	if b.ensureSchema {
		path, err := filepath.Abs(basePath)
		if err != nil {
			return nil, err
		}
		if err := db.EnsureSchema(ctx, b.pool, path); err != nil {
			return nil, err
		}
	}

	if b.repo == nil {
		queries := dbsqlc.New(b.pool)
		b.repo = sqlcrepo.NewProjectRepository(queries)
	}
	app.Repo = b.repo

	if b.notifier == nil {
		b.notifier = telegram.NewSender(b.cfg.TelegramToken, b.cfg.TelegramChat, b.cfg.TelegramThreadID)
	}
	app.Notifier = b.notifier

	if b.client == nil {
		b.client = &http.Client{Timeout: 15 * time.Second}
	}

	if b.scrapers == nil {
		b.scrapers = []scraping.SiteScraper{
			ponisha.NewScraper(b.client),
			karlancer.NewScraper(b.client),
		}
	}
	app.Scrapers = b.scrapers

	app.ScrapeService = scraping.NewService(app.Repo, app.Notifier, app.Scrapers)

	if b.scheduler == nil {
		b.scheduler = scheduler.New(b.cfg.CronSpec, app.ScrapeService)
	}
	app.Scheduler = b.scheduler

	if b.server == nil {
		handler := httpapi.NewHandler(app.ScrapeService)
		b.server = &http.Server{
			Addr:              ":" + b.cfg.HTTPPort,
			Handler:           handler.Router(),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}
	app.Server = b.server

	return app, nil
}
