package repositories

import (
	"context"
	"errors"

	"ponisha-go/internal/model"
)

var ErrNotFound = errors.New("record not found")

type ProjectRepository interface {
	CreateIfNotExists(ctx context.Context, input model.ProjectCreate) (model.Project, bool, error)
}
