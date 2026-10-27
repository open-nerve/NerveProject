-- +goose Up
CREATE TABLE draft_issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid REFERENCES projects ON DELETE CASCADE,
    payload jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(payload) = 'object'),
    created_by_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX draft_issues_author_created_at_idx ON draft_issues (workspace_id, created_by_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX draft_issues_workspace_id_idx ON draft_issues (workspace_id);
CREATE INDEX draft_issues_project_id_idx ON draft_issues (project_id);

-- +goose Down
DROP TABLE draft_issues;
