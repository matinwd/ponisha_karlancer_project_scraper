package scraping

import (
	"context"
	"errors"
	"log"
	"sync"

	"golang.org/x/sync/errgroup"

	"ponisha-go/internal/model"
	"ponisha-go/internal/repositories"
)

type Service struct {
	repo     repositories.ProjectRepository
	notifier Notifier
	scrapers []SiteScraper

	mu      sync.Mutex
	running bool
}

func NewService(repo repositories.ProjectRepository, notifier Notifier, scrapers []SiteScraper) *Service {
	return &Service{repo: repo, notifier: notifier, scrapers: scrapers}
}

func (s *Service) Run(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("scrape already running; skipping")
		return
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	if err := s.scrape(ctx); err != nil {
		log.Printf("scrape error: %v", err)
	}
}

func (s *Service) scrape(ctx context.Context) error {
	log.Printf("Scraping started")

	type result struct {
		source   string
		projects []model.ScrapedProject
	}

	results := make(chan result, len(s.scrapers))
	group, gctx := errgroup.WithContext(ctx)

	for _, scraper := range s.scrapers {
		sc := scraper
		group.Go(func() error {
			log.Printf("[%s] scraping...", sc.Source())
			projects, err := sc.Scrape(gctx)
			if err != nil {
				log.Printf("[%s] scrape failed: %v", sc.Source(), err)
				results <- result{source: sc.Source(), projects: []model.ScrapedProject{}}
				return nil
			}
			log.Printf("[%s] found %d projects", sc.Source(), len(projects))
			results <- result{source: sc.Source(), projects: projects}
			return nil
		})
	}

	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("scrape group error: %v", err)
	}
	close(results)

	stats := map[string]*scrapeStats{}

	for res := range results {
		st := stats[res.source]
		if st == nil {
			st = &scrapeStats{}
			stats[res.source] = st
		}
		st.fetched += len(res.projects)

		for _, project := range res.projects {
			if !isAboveThreshold(project) {
				st.belowThreshold++
				continue
			}
			st.overThreshold++

			saved, created, err := s.repo.CreateIfNotExists(ctx, model.ProjectCreate{
				Source:     project.Source,
				ExternalID: project.ExternalID,
				Title:      project.Title,
				Link:       project.Link,
				BudgetText: project.BudgetText,
				AmountMin:  project.AmountMin,
				AmountMax:  project.AmountMax,
			})
			if err != nil {
				log.Printf("[%s] insert failed: %v", project.Source, err)
				continue
			}
			if !created {
				st.duplicates++
				if project.Source == "karlancer" {
					log.Printf("[karlancer] duplicate high-budget project: externalId=%s title=%s amountMin=%d amountMax=%d link=%s",
						project.ExternalID, project.Title, project.AmountMin, project.AmountMax, project.Link,
					)
				}
				continue
			}
			st.saved++
			telegramProject := project
			telegramProject.Source = saved.Source
			telegramProject.Link = saved.Link
			s.notifier.SendAlert(telegramProject)
		}
	}

	for source, st := range stats {
		log.Printf("[%s] summary: fetched=%d overThreshold=%d saved=%d duplicates=%d belowThreshold=%d",
			source, st.fetched, st.overThreshold, st.saved, st.duplicates, st.belowThreshold,
		)
	}

	return nil
}

type scrapeStats struct {
	fetched        int
	overThreshold  int
	belowThreshold int
	duplicates     int
	saved          int
}

func isAboveThreshold(p model.ScrapedProject) bool {
	return model.IsAboveThreshold(p.AmountMin, p.AmountMax)
}
