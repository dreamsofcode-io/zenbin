-- name: CheckSnippet24Hours :one
SELECT COUNT(*)
FROM snippet
WHERE creator_ip = (sqlc.arg(ip)::varchar)::inet
AND date_trunc('day', created_at) = date_trunc('day', NOW());

-- name: InsertSnippet :one
INSERT INTO snippet(id, creator_ip, content, created_at) VALUES ($1, (sqlc.arg(ip)::varchar)::inet, $2, now()) RETURNING *;

-- name: FindSnippetByID :one
SELECT * FROM snippet WHERE id = $1;

-- name: InsertSnippetCheck :one
WITH snippet_count AS (
    SELECT COUNT(*) AS count
    FROM snippet
    WHERE creator_ip = (sqlc.arg(ip)::varchar)::inet
  AND date_trunc('day', created_at) = date_trunc('day', NOW())
)
INSERT INTO snippet (id, content, creator_ip, created_at)
SELECT $1, $2,(sqlc.arg(ip)::varchar)::inet, now()
FROM snippet_count
WHERE count < 5
RETURNING *;
