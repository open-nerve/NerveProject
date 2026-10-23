-- issue_relations：Plane 的 issue_relations 表（11 列）按 M4 设计 4.6 照搬 11 列。
-- 一对工作项一行（3.10）：阻塞反过来存为对方被本工作项阻塞；project_id 是 issue_id 那一端的项目。

-- +goose Up
CREATE TABLE issue_relations (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- issue_id 那一端的项目（Plane 写请求路径的项目，3.10）
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    related_issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- Plane 默认 blocked_by、不检查取值；Nerve 不设默认，只有页面的三种存储值（阻塞存为反向的 blocked_by）
    relation_type varchar(20) NOT NULL CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to')),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT issue_relations_not_self_check CHECK (issue_id <> related_issue_id)
);
-- 一对工作项之间最多一条未删除的关联，不论类型和方向（取代 Plane 有方向的键，3.10）
CREATE UNIQUE INDEX issue_relations_pair_key ON issue_relations (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id))
    WHERE deleted_at IS NULL;
-- 两个方向的读取和物理级联按它们找行，所以不带条件（4.6）
CREATE INDEX issue_relations_issue_id_idx ON issue_relations (issue_id);
CREATE INDEX issue_relations_related_issue_id_idx ON issue_relations (related_issue_id);
CREATE INDEX issue_relations_workspace_id_idx ON issue_relations (workspace_id);
CREATE INDEX issue_relations_project_id_idx ON issue_relations (project_id);

-- +goose Down
DROP TABLE issue_relations;
