-- Description mentions retain active members, including guests who cannot see the issue.
-- +goose Up
CREATE TABLE issue_mentions (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
 issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
 mention_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
 created_by_id uuid REFERENCES users ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_mentions_issue_id_mention_id_key ON issue_mentions (issue_id, mention_id) WHERE deleted_at IS NULL;
CREATE INDEX issue_mentions_mention_id_idx ON issue_mentions (mention_id) WHERE deleted_at IS NULL;
CREATE INDEX issue_mentions_issue_id_idx ON issue_mentions (issue_id);
CREATE INDEX issue_mentions_workspace_id_idx ON issue_mentions (workspace_id);
CREATE INDEX issue_mentions_project_id_idx ON issue_mentions (project_id);
-- +goose Down
DROP TABLE issue_mentions;
