package model

type ScrapedProject struct {
	Source          string
	ExternalID      string
	Title           string
	Link            string
	BudgetText      string
	AmountMin       int64
	AmountMax       int64
	Description     string
	Skills          []string
	ApprovedAt      string
	BiddingClosedAt string
	BidsCount       *int
}
