-- name: GetUserByEmail :one
SELECT id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at
FROM users
WHERE email = $1;

-- name: GetUserByNickname :one
SELECT id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at
FROM users
WHERE nickname = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, full_name, first_name, last_name, nickname)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
RETURNING id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at;

-- name: UpdateUserDefaultCurrency :one
UPDATE users
SET default_currency_code = $2
WHERE id = $1
RETURNING id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at;

-- name: UpdateUserProfile :one
UPDATE users
SET full_name = $2,
    first_name = $3,
    last_name = $4,
    nickname = $5,
    default_currency_code = $6
WHERE id = $1
RETURNING id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at;

-- name: UpdateUserEmail :one
UPDATE users
SET email = $2
WHERE id = $1
RETURNING id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: UpdateUserAvatarPath :one
UPDATE users
SET avatar_path = $2
WHERE id = $1
RETURNING id, email, password_hash, full_name, first_name, last_name, nickname, avatar_path, default_currency_code, created_at;
