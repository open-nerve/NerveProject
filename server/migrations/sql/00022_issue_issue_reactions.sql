-- An actor can have one live row per decimal reaction code and issue.
-- +goose Up
CREATE TABLE issue_reactions (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
 issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
 reaction text NOT NULL CHECK (reaction ~ '^[0-9]{1,7}(-[0-9]{1,7}){0,15}$' AND length(reaction) <= 128),
 created_by_id uuid REFERENCES users ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_reactions_issue_id_actor_id_reaction_key ON issue_reactions (issue_id, actor_id, reaction) WHERE deleted_at IS NULL;
CREATE INDEX issue_reactions_issue_id_idx ON issue_reactions (issue_id);
CREATE INDEX issue_reactions_workspace_id_idx ON issue_reactions (workspace_id);
CREATE INDEX issue_reactions_project_id_idx ON issue_reactions (project_id);
-- +goose Down
DROP TABLE issue_reactions;
