package ponisha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/sync/errgroup"

	"ponisha-go/internal/model"
	"ponisha-go/internal/providers/common"
)

type PonishaScraper struct {
	client *http.Client
}

const (
	ponishaBaseURL   = "https://ponisha.ir/search/projects"
	ponishaUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	ponishaPageLimit = 4
)

func NewScraper(client *http.Client) *PonishaScraper {
	return &PonishaScraper{client: client}
}

func (p *PonishaScraper) Source() string {
	return "ponisha"
}

func (p *PonishaScraper) Scrape(ctx context.Context) ([]model.ScrapedProject, error) {
	firstPage, totalPages, err := p.fetchFirstPage(ctx)
	if err != nil {
		return nil, err
	}
	if totalPages <= 1 {
		return firstPage, nil
	}

	return p.fetchRemainingPages(ctx, totalPages, firstPage)
}

func (p *PonishaScraper) fetchFirstPage(ctx context.Context) ([]model.ScrapedProject, int, error) {
	page := 1
	log.Printf("[ponisha] page %d/%d", page, page)
	projects, totalPages, err := p.fetchPage(ctx, buildPonishaURL(page))
	if err != nil {
		log.Printf("[ponisha] failed on page %d: %v", page, err)
		return nil, 0, err
	}
	log.Printf("[ponisha] page %d found %d items (total pages: %d)", page, len(projects), totalPages)
	return projects, totalPages, nil
}

func (p *PonishaScraper) fetchRemainingPages(ctx context.Context, totalPages int, seed []model.ScrapedProject) ([]model.ScrapedProject, error) {
	projects := make([]model.ScrapedProject, 0, len(seed)*totalPages)
	projects = append(projects, seed...)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(ponishaPageLimit)

	var mu sync.Mutex
	for page := 2; page <= totalPages; page++ {
		page := page
		group.Go(func() error {
			log.Printf("[ponisha] page %d/%d", page, totalPages)
			pageProjects, _, err := p.fetchPage(gctx, buildPonishaURL(page))
			if err != nil {
				log.Printf("[ponisha] failed on page %d: %v", page, err)
				return nil
			}
			log.Printf("[ponisha] page %d found %d items (total pages: %d)", page, len(pageProjects), totalPages)
			mu.Lock()
			projects = append(projects, pageProjects...)
			mu.Unlock()
			return nil
		})
	}

	_ = group.Wait()
	return projects, nil
}

func (p *PonishaScraper) fetchPage(ctx context.Context, url string) ([]model.ScrapedProject, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", ponishaUserAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return extractPonishaProjects(doc)
}

func extractPonishaProjects(doc *goquery.Document) ([]model.ScrapedProject, int, error) {
	var payload map[string]any
	if err := decodeNextPayload(doc, &payload); err != nil {
		return nil, 0, fmt.Errorf("json parse error: %w", err)
	}
	if payload == nil {
		return nil, 0, nil
	}

	target := findProjectsQuery(payload)
	if target == nil {
		return nil, 0, nil
	}

	data := nestedMap(target, "state", "data")
	if data == nil {
		return nil, 0, nil
	}

	totalPages := readTotalPages(data)

	list, ok := data["data"].([]any)
	if !ok {
		return nil, totalPages, nil
	}

	projects := make([]model.ScrapedProject, 0, len(list))
	for _, item := range list {
		project, ok := parseProject(item)
		if !ok {
			continue
		}
		projects = append(projects, project)
	}

	return projects, totalPages, nil
}

func buildPonishaURL(page int) string {
	return fmt.Sprintf("%s?page=%d&order=approved_at%%7Cdesc&promotion=-&filterByProjectStatus=open", ponishaBaseURL, page)
}

func decodeNextPayload(doc *goquery.Document, out *map[string]any) error {
	script := doc.Find("script#__NEXT_DATA__").First().Text()
	if script == "" {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(script))
	decoder.UseNumber()
	return decoder.Decode(out)
}

func findProjectsQuery(payload map[string]any) map[string]any {
	for _, q := range findQueries(payload) {
		if hasProjectPagination(q) {
			return q
		}
	}
	return nil
}

func readTotalPages(data map[string]any) int {
	meta := nestedMap(data, "meta", "pagination")
	if meta == nil {
		return 0
	}
	val, ok := meta["total_pages"]
	if !ok {
		return 0
	}
	return int(common.ToInt64(val))
}

func parseProject(item any) (model.ScrapedProject, bool) {
	p, ok := item.(map[string]any)
	if !ok {
		return model.ScrapedProject{}, false
	}

	id := common.ToString(p["id"])
	if id == "" {
		return model.ScrapedProject{}, false
	}

	slug := common.ToString(p["slug"])
	amountMin := common.ToInt64(p["amount_min"])
	amountMax := common.ToInt64(p["amount_max"])
	if !model.IsAboveThreshold(amountMin, amountMax) {
		return model.ScrapedProject{}, false
	}

	project := model.ScrapedProject{
		Source:          "ponisha",
		ExternalID:      id,
		Title:           common.ToString(p["title"]),
		Link:            fmt.Sprintf("https://ponisha.ir/project/%s/%s", id, slug),
		BudgetText:      common.FormatBudgetText(amountMin, amountMax),
		AmountMin:       amountMin,
		AmountMax:       amountMax,
		Description:     common.ToString(p["description"]),
		ApprovedAt:      common.ToString(p["approved_at"]),
		BiddingClosedAt: common.ToString(p["bidding_closed_at"]),
	}

	if bids, ok := p["project_bids_count"]; ok {
		b := int(common.ToInt64(bids))
		project.BidsCount = &b
	}

	project.Skills = extractSkillNames(p["skills"])
	return project, true
}

func extractSkillNames(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	skills := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := common.ToString(m["name"])
		if name != "" {
			skills = append(skills, name)
		}
	}
	return skills
}

func findQueries(payload map[string]any) []map[string]any {
	props := nestedMap(payload, "props", "pageProps", "dehydratedState")
	if props == nil {
		return nil
	}
	queriesRaw, ok := props["queries"].([]any)
	if !ok {
		return nil
	}

	queries := make([]map[string]any, 0, len(queriesRaw))
	for _, q := range queriesRaw {
		if m, ok := q.(map[string]any); ok {
			queries = append(queries, m)
		}
	}
	return queries
}

func hasProjectPagination(query map[string]any) bool {
	queryKey, ok := query["queryKey"].([]any)
	if ok && len(queryKey) >= 2 {
		if common.ToString(queryKey[0]) == "search" && common.ToString(queryKey[1]) == "projects" {
			if nestedMap(query, "state", "data", "meta", "pagination") != nil {
				return true
			}
		}
	}
	return nestedMap(query, "state", "data", "meta", "pagination") != nil
}

func nestedMap(root map[string]any, keys ...string) map[string]any {
	current := root
	for _, key := range keys {
		value, ok := current[key]
		if !ok {
			return nil
		}
		child, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		current = child
	}
	return current
}
