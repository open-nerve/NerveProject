-- issues：Plane 的 issues 表（28 列）按 M4 设计 4.2 保留 20 列。
-- 编号取自项目行的计数列 last_issue_sequence（总体设计 5.3）；描述、起止日期的列随表建出，进入接口在 P4、P2（3.7）。

-- +goose Up
CREATE TABLE issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 父子在同一项目（3.9）；自己不能做自己的父工作项
    parent_id uuid REFERENCES issues ON DELETE CASCADE,
    -- Plane 可空，改为非空：创建总有状态（3.7）。单独删状态拒绝未删除工作项；父级异步删除后，过期状态或父行的 CASCADE 可回收仍为活标记的工作项（3.6）。
    state_id uuid NOT NULL REFERENCES states ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description_html text NOT NULL DEFAULT '<p></p>',
    description_stripped text,
    priority varchar(30) NOT NULL DEFAULT 'none' CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none')),
    start_date date,
    target_date date,
    -- 没有默认值：取自项目的计数列
    sequence_id integer NOT NULL CHECK (sequence_id > 0),
    sort_order double precision NOT NULL DEFAULT 65535,
    completed_at timestamptz,
    -- Plane 是 date
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT issues_not_own_parent_check CHECK (parent_id <> id),
    CONSTRAINT issues_dates_check CHECK (start_date <= target_date),
    -- 全表唯一，不是部分唯一：计数器从不回退，编号从不重用（3.7）；也服务按编号取
    CONSTRAINT issues_project_id_sequence_id_key UNIQUE (project_id, sequence_id)
);
-- 物理级联、子工作项、在用的状态按它们找行，所以不带条件（4.2）
CREATE INDEX issues_workspace_id_idx ON issues (workspace_id);
CREATE INDEX issues_project_id_idx ON issues (project_id);
CREATE INDEX issues_parent_id_idx ON issues (parent_id);
CREATE INDEX issues_state_id_idx ON issues (state_id);
-- 同一状态的 sort_order 的最大值、按状态筛选（3.7）
CREATE INDEX issues_project_id_state_id_idx ON issues (project_id, state_id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE issues;
