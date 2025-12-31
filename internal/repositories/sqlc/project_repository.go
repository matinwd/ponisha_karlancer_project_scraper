package sqlc

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	db "ponisha-go/internal/db/sqlc"
	"ponisha-go/internal/model"
)

type ProjectRepository struct {
	queries *db.Queries
}

func NewProjectRepository(queries *db.Queries) *ProjectRepository {
	return &ProjectRepository{queries: queries}
}

func (r *ProjectRepository) CreateIfNotExists(ctx context.Context, input model.ProjectCreate) (model.Project, bool, error) {
	project, err := r.queries.CreateProjectIfNotExists(ctx, db.CreateProjectIfNotExistsParams{
		Source:     input.Source,
		ExternalID: input.ExternalID,
		Title:      input.Title,
		Link:       input.Link,
		BudgetText: input.BudgetText,
		AmountMin:  input.AmountMin,
		AmountMax:  input.AmountMax,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Project{}, false, nil
	}
	if err != nil {
		return model.Project{}, false, err
	}
	return mapProject(project), true, nil
}

func mapProject(project db.Project) model.Project {
	var createdAt time.Time
	if project.CreatedAt.Valid {
		createdAt = project.CreatedAt.Time
	}
	return model.Project{
		ID:         project.ID,
		Source:     project.Source,
		ExternalID: project.ExternalID,
		Title:      project.Title,
		Link:       project.Link,
		BudgetText: project.BudgetText,
		AmountMin:  project.AmountMin,
		AmountMax:  project.AmountMax,
		CreatedAt:  createdAt,
	}
}
