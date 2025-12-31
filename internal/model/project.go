package model

import "time"

type Project struct {
	ID         int32
	Source     string
	ExternalID string
	Title      string
	Link       string
	BudgetText string
	AmountMin  int64
	AmountMax  int64
	CreatedAt  time.Time
}

type ProjectCreate struct {
	Source     string
	ExternalID string
	Title      string
	Link       string
	BudgetText string
	AmountMin  int64
	AmountMax  int64
}
