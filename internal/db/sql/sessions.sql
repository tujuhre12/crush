-- name: CreateSession :one
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    message_count,
    prompt_tokens,
    completion_tokens,
    cost,
    summary_message_id,
    updated_at,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    null,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING *;

-- name: GetSessionByID :one
SELECT *
FROM sessions
WHERE id = ? LIMIT 1;

-- name: ListSessions :many
SELECT *
FROM sessions
WHERE parent_session_id is NULL
ORDER BY created_at DESC;

-- name: UpdateSession :one
UPDATE sessions
SET
    title = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    summary_message_id = ?,
    cost = ?
WHERE id = ?
RETURNING *;


-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;

-- name: ListChildSessions :many
SELECT *
FROM sessions
WHERE parent_session_id = ?
ORDER BY created_at ASC;

-- name: ListAllSessions :many
SELECT *
FROM sessions
ORDER BY created_at DESC;

-- name: SearchSessionsByTitle :many
SELECT *
FROM sessions
WHERE title LIKE ?
ORDER BY created_at DESC;

-- name: SearchSessionsByTitleAndText :many
SELECT DISTINCT s.*
FROM sessions s
JOIN messages m ON s.id = m.session_id
WHERE s.title LIKE ? AND m.parts LIKE ?
ORDER BY s.created_at DESC;

-- name: SearchSessionsByText :many
SELECT DISTINCT s.*
FROM sessions s
JOIN messages m ON s.id = m.session_id
WHERE m.parts LIKE ?
ORDER BY s.created_at DESC;

-- name: GetSessionStats :one
SELECT 
    COUNT(*) as total_sessions,
    SUM(message_count) as total_messages,
    SUM(prompt_tokens) as total_prompt_tokens,
    SUM(completion_tokens) as total_completion_tokens,
    SUM(cost) as total_cost,
    AVG(cost) as avg_cost_per_session
FROM sessions;

-- name: GetSessionStatsByDay :many
SELECT 
    date(created_at, 'unixepoch') as day,
    COUNT(*) as session_count,
    SUM(message_count) as message_count,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    SUM(cost) as total_cost,
    AVG(cost) as avg_cost
FROM sessions
GROUP BY date(created_at, 'unixepoch')
ORDER BY day DESC;

-- name: GetSessionStatsByWeek :many
SELECT 
    date(created_at, 'unixepoch', 'weekday 0', '-6 days') as week_start,
    COUNT(*) as session_count,
    SUM(message_count) as message_count,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    SUM(cost) as total_cost,
    AVG(cost) as avg_cost
FROM sessions
GROUP BY date(created_at, 'unixepoch', 'weekday 0', '-6 days')
ORDER BY week_start DESC;

-- name: GetSessionStatsByMonth :many
SELECT 
    strftime('%Y-%m', datetime(created_at, 'unixepoch')) as month,
    COUNT(*) as session_count,
    SUM(message_count) as message_count,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    SUM(cost) as total_cost,
    AVG(cost) as avg_cost
FROM sessions
GROUP BY strftime('%Y-%m', datetime(created_at, 'unixepoch'))
ORDER BY month DESC;
