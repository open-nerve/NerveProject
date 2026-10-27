-- +goose Up
CREATE INDEX api_tokens_deleted_at_idx ON api_tokens(deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX api_tokens_deleted_at_idx;
