-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX issues_name_trgm_idx ON issues USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX issues_name_trgm_idx;
