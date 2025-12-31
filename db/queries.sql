-- name: GetProjectBySourceExternalID :one
SELECT id, source, external_id, title, link, budget_text, amount_min, amount_max, created_at
FROM projects
WHERE source = $1 AND external_id = $2
LIMIT 1;

-- name: CreateProjectIfNotExists :one
INSERT INTO projects (
  source,
  external_id,
  title,
  link,
  budget_text,
  amount_min,
  amount_max
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (source, external_id) DO NOTHING
RETURNING id, source, external_id, title, link, budget_text, amount_min, amount_max, created_at;
