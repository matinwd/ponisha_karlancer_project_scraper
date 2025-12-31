package karlancer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"ponisha-go/internal/model"
	"ponisha-go/internal/providers/common"
)

type KarlancerScraper struct {
	client *http.Client
	base   string
}

func NewScraper(client *http.Client) *KarlancerScraper {
	return &KarlancerScraper{client: client, base: "https://www.karlancer.com/api/publics/search/projects"}
}

func (k *KarlancerScraper) Source() string {
	return "karlancer"
}

type karlancerResponse struct {
	Data *struct {
		CurrentPage int                `json:"current_page"`
		LastPage    int                `json:"last_page"`
		Data        []karlancerProject `json:"data"`
	} `json:"data"`
}

type karlancerProject struct {
	ID          any    `json:"id"`
	UUID        any    `json:"uuid"`
	AltID       any    `json:"_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	MinBudget   any    `json:"min_budget"`
	MaxBudget   any    `json:"max_budget"`
	BudgetFrom  any    `json:"budget_from"`
	BudgetTo    any    `json:"budget_to"`
	AmountMin   any    `json:"amount_min"`
	AmountMax   any    `json:"amount_max"`
	PriceMin    any    `json:"price_min"`
	PriceMax    any    `json:"price_max"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
	ApprovedAt  string `json:"approved_at"`
	ExpiredAt   string `json:"expired_at"`
	ExpiredAlt  string `json:"expiredAt"`
	BidsCount   *int   `json:"bids_count"`
	BidsAlt     *int   `json:"bidsCount"`
	Skills      []struct {
		Name  string `json:"name"`
		Title string `json:"title"`
	} `json:"skills"`
}

func (k *KarlancerScraper) Scrape(ctx context.Context) ([]model.ScrapedProject, error) {
	log.Printf("[karlancer] page 1/1")
	firstPage, lastPage, err := k.fetchPage(ctx, 1)
	if err != nil {
		return nil, err
	}

	log.Printf("[karlancer] page 1 found %d items (total pages: %d)", len(firstPage), lastPage)
	if lastPage <= 1 {
		return firstPage, nil
	}

	projects := make([]model.ScrapedProject, 0, len(firstPage)*(lastPage))
	projects = append(projects, firstPage...)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(4)

	var mu sync.Mutex
	for page := 2; page <= lastPage; page++ {
		current := page
		group.Go(func() error {
			log.Printf("[karlancer] page %d/%d", current, lastPage)
			items, _, err := k.fetchPage(gctx, current)
			if err != nil {
				return err
			}
			log.Printf("[karlancer] page %d found %d items (total pages: %d)", current, len(items), lastPage)
			mu.Lock()
			projects = append(projects, items...)
			mu.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return projects, err
	}

	return projects, nil
}

func (k *KarlancerScraper) fetchPage(ctx context.Context, page int) ([]model.ScrapedProject, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, k.base, nil)
	if err != nil {
		return nil, 0, err
	}

	q := req.URL.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("order", "newest")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.karlancer.com/")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	var payload karlancerResponse
	if err := decoder.Decode(&payload); err != nil {
		return nil, 0, err
	}

	data := payload.Data
	if data == nil {
		return nil, 0, nil
	}

	projects := make([]model.ScrapedProject, 0, len(data.Data))
	for _, p := range data.Data {
		id := pickID(p.ID, p.UUID, p.AltID)
		if id == "" {
			continue
		}

		amountMin := common.ToInt64(p.MinBudget)
		if amountMin == 0 {
			amountMin = common.ToInt64(p.BudgetFrom)
		}
		if amountMin == 0 {
			amountMin = common.ToInt64(p.AmountMin)
		}
		if amountMin == 0 {
			amountMin = common.ToInt64(p.PriceMin)
		}

		amountMax := common.ToInt64(p.MaxBudget)
		if amountMax == 0 {
			amountMax = common.ToInt64(p.BudgetTo)
		}
		if amountMax == 0 {
			amountMax = common.ToInt64(p.AmountMax)
		}
		if amountMax == 0 {
			amountMax = common.ToInt64(p.PriceMax)
		}

		if !model.IsAboveThreshold(amountMin, amountMax) {
			continue
		}

		linkSlug := p.URL
		if linkSlug == "" {
			linkSlug = id
		}

		project := model.ScrapedProject{
			Source:          "karlancer",
			ExternalID:      id,
			Title:           pickTitle(p.Title),
			Link:            fmt.Sprintf("https://www.karlancer.com/projects/%s", linkSlug),
			BudgetText:      common.FormatBudgetText(amountMin, amountMax),
			AmountMin:       amountMin,
			AmountMax:       amountMax,
			Description:     p.Description,
			ApprovedAt:      pickString(p.PublishedAt, p.ApprovedAt),
			BiddingClosedAt: pickString(p.ExpiredAt, p.ExpiredAlt),
			BidsCount:       pickIntPtr(p.BidsCount, p.BidsAlt),
			Skills:          collectSkills(p.Skills),
		}

		projects = append(projects, project)
	}

	return projects, data.LastPage, nil
}

func pickID(values ...any) string {
	for _, v := range values {
		if s := common.ToString(v); s != "" {
			return s
		}
	}
	return ""
}

func pickTitle(title string) string {
	if title == "" {
		return "بدون عنوان"
	}
	return title
}

func pickString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func pickIntPtr(values ...*int) *int {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func collectSkills(skills []struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}) []string {
	if len(skills) == 0 {
		return nil
	}

	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		value := skill.Name
		if value == "" {
			value = skill.Title
		}
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
