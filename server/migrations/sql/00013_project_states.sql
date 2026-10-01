-- states：Plane 的 states 表（18 列）按 M3 设计 4.9 保留 14 列。
-- 分诊状态只由 group = 'triage' 识别，删除 is_triage；slug 没有读取者，删除（3.17）。建项目时建出默认的 6 个状态。

-- +goose Up
CREATE TABLE states (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description text NOT NULL DEFAULT '',
    color varchar(255) NOT NULL,
    sequence double precision NOT NULL DEFAULT 65535,
    -- Plane 的 StateGroup；列名照搬，SQL 中带引号
    "group" varchar(20) NOT NULL DEFAULT 'backlog'
        CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage')),
    "default" boolean NOT NULL DEFAULT false,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 名称在项目内唯一，区分大小写（照搬）
CREATE UNIQUE INDEX states_project_id_name_key ON states (project_id, name) WHERE deleted_at IS NULL;
-- 每个项目至多一个默认状态、至多一个分诊状态（3.17；名字按 4 节开头的补充）
CREATE UNIQUE INDEX states_project_id_default_key ON states (project_id) WHERE "default" AND deleted_at IS NULL;
CREATE UNIQUE INDEX states_project_id_triage_key ON states (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX states_workspace_id_idx ON states (workspace_id);
CREATE INDEX states_project_id_idx ON states (project_id);

-- +goose Down
DROP TABLE states;
