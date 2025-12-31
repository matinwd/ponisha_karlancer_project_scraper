package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"

	"ponisha-go/internal/services/scraping"
)

type Scheduler struct {
	cron    *cron.Cron
	service *scraping.Service
	spec    string
}

func New(spec string, service *scraping.Service) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		service: service,
		spec:    spec,
	}
}

func (s *Scheduler) Start() error {
	_, err := s.cron.AddFunc(s.spec, func() {
		log.Printf("scheduled scrape triggered")
		go s.service.Run(context.Background())
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}
