-- cycles：Plane 的 cycles 表按 M6 设计 4.2 与 P1 spec 2.2 保留 15 列。
-- 起止日期改为项目时区下的日历日（date），两者同空同有、开始不晚于结束；状态不入库，读取时推算（设计 3.1）。
-- owned_by 由 Plane 的非空 CASCADE 改为可空 SET NULL（同 M3 对项目负责人的改法）。
-- view_props、logo_props、timezone、version、external_source、external_id 没有读者或改用项目的时区，删除；
-- progress_snapshot 在 P4 随冻结加入。

-- +goose Up
CREATE TABLE cycles (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description text NOT NULL DEFAULT '',
    start_date date,
    end_date date,
    owned_by_id uuid REFERENCES users ON DELETE SET NULL,
    sort_order double precision NOT NULL DEFAULT 65535,
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT cycles_dates_check CHECK (
        (start_date IS NULL AND end_date IS NULL)
        OR (start_date IS NOT NULL AND end_date IS NOT NULL AND start_date <= end_date)
    )
);
-- 项目的迭代列表与归档目录（默认顺序 sort_order, created_at, id）
CREATE INDEX cycles_project_list_idx ON cycles (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL;
-- 物理级联（60 天清理）按它们找子行，所以不带条件
CREATE INDEX cycles_workspace_id_idx ON cycles (workspace_id);
CREATE INDEX cycles_project_id_idx ON cycles (project_id);
-- 60 天清理找候选
CREATE INDEX cycles_deleted_at_idx ON cycles (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE cycles;
