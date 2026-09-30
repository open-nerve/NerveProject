-- name: CreateUser :exec
-- The other columns take their defaults; the audit columns come from the use case's clock (M2 design 3.13).
INSERT INTO users (id, email, password, display_name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

-- name: GetUser :one
SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
FROM users
WHERE id = sqlc.arg(id);

-- name: FindLoginAccount :one
-- What login reads before its transaction; the hash is its snapshot (M2 design 3.5).
SELECT id, password
FROM users
WHERE email = sqlc.arg(email);

-- name: GetPasswordAccount :one
-- What changing the password reads before its transaction: the address for the password rules,
-- the hash as the snapshot (M2 design 3.5).
SELECT email, password
FROM users
WHERE id = sqlc.arg(id);

-- name: LockUserForCredentials :one
-- The account row lock of M2 design 3.5. FOR NO KEY UPDATE conflicts with itself and with
-- FOR UPDATE, so the credential transactions of one account run one after another; it does not
-- conflict with the FOR KEY SHARE that foreign-key checks take, so inserting rows that reference
-- the account does not wait.
SELECT password, is_active
FROM users
WHERE id = sqlc.arg(id)
FOR NO KEY UPDATE;

-- name: UpdateUser :one
-- PATCH /me: only the fields that are set change (M2 design 3.14); the rest keep what a concurrent write left.
UPDATE users
SET updated_at    = sqlc.arg(now),
    first_name    = CASE WHEN sqlc.arg(set_first_name)::boolean THEN sqlc.arg(first_name)::text ELSE first_name END,
    last_name     = CASE WHEN sqlc.arg(set_last_name)::boolean THEN sqlc.arg(last_name)::text ELSE last_name END,
    display_name  = CASE WHEN sqlc.arg(set_display_name)::boolean THEN sqlc.arg(display_name)::text ELSE display_name END,
    user_timezone = CASE WHEN sqlc.arg(set_user_timezone)::boolean THEN sqlc.arg(user_timezone)::text ELSE user_timezone END
WHERE id = sqlc.arg(id)
RETURNING id, email, first_name, last_name, display_name, user_timezone, created_at;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: UpdatePasswordHash :exec
UPDATE users
SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: LockAccountByEmail :one
-- The account row lock of M2 design 3.5 for the server administrator's commands, which name the
-- account by its address: one statement finds and locks the row, inside the command's
-- transaction, so the account cannot change between the two.
SELECT id
FROM users
WHERE email = sqlc.arg(email)
FOR NO KEY UPDATE;

-- name: ChangeEmail :exec
-- nerve users set-email (M2 decision 1). users_email_key rejects an address another account has.
UPDATE users
SET email = sqlc.arg(email), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ActivateUser :exec
UPDATE users
SET is_active = true, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ShareAccount :one
-- Accounts (M3 design 6.5): the first lock of a transaction that gives the account a workspace
-- membership or answers an invitation to its address, declining too (3.6 conventions 1 and 6).
-- FOR SHARE conflicts with deactivation's FOR NO KEY UPDATE, so the two run one after the other
-- and is_active is read under the lock; two FOR SHARE do not wait for each other.
SELECT id, email, is_active
FROM users
WHERE id = sqlc.arg(id)
FOR SHARE;

-- name: ShareAccountByEmail :one
-- ShareAccount for the server administrator's commands, which name the account by its address.
SELECT id, email, is_active
FROM users
WHERE email = sqlc.arg(email)
FOR SHARE;

-- name: PublicProfiles :many
-- MemberProfiles (M3 design 6.5): the public profile of each account of ids, deactivated ones too, by id. No lock: a
-- transaction that holds a workspace's lock reads an address this way (3.6 convention 1).
SELECT id, email, first_name, last_name, display_name
FROM users
WHERE id = ANY (sqlc.arg(ids)::uuid[])
ORDER BY id;
