-- labels：Plane 的 labels 表（15 列）按 M3 设计 4.10 保留 12 列。
-- 只有项目的标签，project_id 非空；最多两层，名称在项目内唯一、不分大小写（3.16）。description 没有读取者，删除。

-- +goose Up
CREATE TABLE labels (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 两层的规则在领域层，并发下由项目行的锁保证（3.16）；自己不能做自己的父标签
    parent_id uuid REFERENCES labels ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    color varchar(255) NOT NULL DEFAULT '',
    sort_order double precision NOT NULL DEFAULT 65535,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT labels_not_own_parent_check CHECK (parent_id <> id)
);
-- 名称在项目内唯一，不分大小写（3.16）
CREATE UNIQUE INDEX labels_project_id_name_key ON labels (project_id, lower(name)) WHERE deleted_at IS NULL;
-- 删除父标签时找子标签，也服务物理级联，所以不带条件（4 节开头）
CREATE INDEX labels_parent_id_idx ON labels (parent_id);
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX labels_workspace_id_idx ON labels (workspace_id);
CREATE INDEX labels_project_id_idx ON labels (project_id);

-- +goose Down
DROP TABLE labels;
