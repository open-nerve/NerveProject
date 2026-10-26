-- +goose Up
CREATE TABLE issue_links (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    title varchar(255),
    url text NOT NULL CHECK (url <> '' AND length(url) <= 2048),
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (
        CASE WHEN jsonb_typeof(metadata) = 'object' THEN
            metadata - ARRAY['title', 'favicon', 'url', 'favicon_url', 'error'] = '{}'::jsonb
            AND NOT jsonb_path_exists(metadata, 'strict $.* ? (@.type() != "string" && @.type() != "null")')
        ELSE false END
    ),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT issue_links_issue_id_url_key EXCLUDE USING hash ((issue_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)
);
CREATE INDEX issue_links_issue_id_idx ON issue_links (issue_id);
CREATE INDEX issue_links_workspace_id_idx ON issue_links (workspace_id);
CREATE INDEX issue_links_project_id_idx ON issue_links (project_id);

-- +goose Down
DROP TABLE issue_links;
