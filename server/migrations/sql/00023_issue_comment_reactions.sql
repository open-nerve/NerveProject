-- An actor can have one live row per decimal reaction code and comment.
-- +goose Up
CREATE TABLE comment_reactions (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
 comment_id uuid NOT NULL REFERENCES issue_comments ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
 reaction text NOT NULL CHECK (reaction ~ '^[0-9]{1,7}(-[0-9]{1,7}){0,15}$' AND length(reaction) <= 128),
 created_by_id uuid REFERENCES users ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE UNIQUE INDEX comment_reactions_comment_id_actor_id_reaction_key ON comment_reactions (comment_id, actor_id, reaction) WHERE deleted_at IS NULL;
CREATE INDEX comment_reactions_comment_id_idx ON comment_reactions (comment_id);
CREATE INDEX comment_reactions_workspace_id_idx ON comment_reactions (workspace_id);
CREATE INDEX comment_reactions_project_id_idx ON comment_reactions (project_id);
-- +goose Down
DROP TABLE comment_reactions;
