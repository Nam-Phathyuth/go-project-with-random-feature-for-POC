-- name: SaveTask :execresult
INSERT INTO tasks (title, content, status) VALUES (?, ?, ?);
-- name: FindTaskById :one
select * from tasks where id = ?;
-- name: GetAllTask :many
select * from tasks where deleted_at is null;