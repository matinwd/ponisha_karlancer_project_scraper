package scraping

import (
	"context"

	"ponisha-go/internal/model"
)

type SiteScraper interface {
	Source() string
	Scrape(ctx context.Context) ([]model.ScrapedProject, error)
}

type Notifier interface {
	SendAlert(project model.ScrapedProject)
}
