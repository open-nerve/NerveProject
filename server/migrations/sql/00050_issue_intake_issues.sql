-- intake_issues：Plane 的 intake_issues 表（18 列）按 M7 设计 4.2 与 P2a spec 2.2 保留 12 列，写法照 cycle_issues（00043）。
-- 一个条目与它的工作项一对一；提交者是条目与工作项共同的 created_by_id。删去 Plane 的 intake_id（Nerve 没有 intakes 表，
-- 收集箱是项目的开关 intake_view）、source、source_email、external_source、external_id、extra（只有应用内来源，M7-01，不建）。
-- status 由 Plane 的数字 -2…2 改为五个名字并由 CHECK 约束（Plane 不约束）；暂缓恰有日期、重复恰有目标（Plane 不约束）。
-- duplicate_to_id 不是外键（设计 §15 D2，Plane 是 SET NULL 的外键）：被指向的工作项删除时不动，读时只给看得到的目标。
-- 条目随工作项在原事务里软删，随项目、工作区的 issue 分批任务删除，60 天后由 issue 的清理在 issues 之前删去（spec 2.10）。

-- +goose Up
CREATE TABLE intake_issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- 存的状态：暂缓到期不改写，读时按那一刻算作待处理（spec 2.3）
    status varchar(20) NOT NULL CHECK (status IN ('pending', 'snoozed', 'accepted', 'declined', 'duplicate')),
    snoozed_till timestamptz,
    duplicate_to_id uuid,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT intake_issues_snoozed_till_check CHECK ((status = 'snoozed') = (snoozed_till IS NOT NULL)),
    CONSTRAINT intake_issues_duplicate_to_id_check CHECK ((status = 'duplicate') = (duplicate_to_id IS NOT NULL))
);
-- 一个工作项在未删除的行中至多一个条目（Plane 没有任何唯一，设计 4.2）；读工作项时按它左连接条目（spec 2.3）
CREATE UNIQUE INDEX intake_issues_issue_id_key ON intake_issues (issue_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（包括已删除的行），所以不带条件
CREATE INDEX intake_issues_workspace_id_idx ON intake_issues (workspace_id);
CREATE INDEX intake_issues_project_id_idx ON intake_issues (project_id);
CREATE INDEX intake_issues_issue_id_idx ON intake_issues (issue_id);
-- 项目的收集箱列表与待处理数（P2b）
CREATE INDEX intake_issues_project_id_status_idx ON intake_issues (project_id, status) WHERE deleted_at IS NULL;
-- 60 天清理找候选
CREATE INDEX intake_issues_deleted_at_idx ON intake_issues (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE intake_issues;
