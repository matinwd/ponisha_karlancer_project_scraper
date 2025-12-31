package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"

	"ponisha-go/internal/services/scraping"
)

type Handler struct {
	service *scraping.Service
}

func NewHandler(service *scraping.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/scraping", h.handleScrape)
	r.Route("/debug/pprof", func(r chi.Router) {
		r.Get("/", pprof.Index)
		r.Get("/cmdline", pprof.Cmdline)
		r.Get("/profile", pprof.Profile)
		r.Get("/symbol", pprof.Symbol)
		r.Post("/symbol", pprof.Symbol)
		r.Get("/trace", pprof.Trace)
		r.Get("/allocs", pprof.Handler("allocs").ServeHTTP)
		r.Get("/block", pprof.Handler("block").ServeHTTP)
		r.Get("/goroutine", pprof.Handler("goroutine").ServeHTTP)
		r.Get("/heap", pprof.Handler("heap").ServeHTTP)
		r.Get("/mutex", pprof.Handler("mutex").ServeHTTP)
		r.Get("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	})
	return r
}

func (h *Handler) handleScrape(w http.ResponseWriter, r *http.Request) {
	go h.service.Run(context.Background())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Scraping started"})
}
