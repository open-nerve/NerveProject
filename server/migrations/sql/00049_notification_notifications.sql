-- notifications：Plane 的 notifications 表（21 列）按 M7 设计 4.2 与 P1b spec 2.2 保留 19 列。
-- 只有工作项一种通知：entity_identifier 改为非空的 issue_id，entity_name 删除；title、message、message_html、
-- message_stripped 前端不读，删除。project_id 由 Plane 的可空改为非空。
-- issue_id 与 issue_created_by_id 不是外键（M4 设计 4.16）：工作项删除时它的通知在原事务里软删（spec 2.8），
-- 读时按快照里的创建者判断只看自己的人能否看到。
-- sender 由 Plane 的 in_app:issue_activities:<类型> 改为类型本身，四种由 CHECK 约束（Plane 不约束）。
-- data 是发送时的快照，顶层恰为 issue、change 两个对象（spec 2.5；Plane 可空、不约束，键为 issue、issue_activity）。
-- triggered_by_id 为空即 nerve 自己。审计列同别的表（设计 4.2）：nerve 写的行 created_by_id、updated_by_id 都为空；
-- 软删时 deleted_at、updated_at 取事件的时刻，updated_by_id 取删除者（自动的删除为空）；收件人的修改写 updated_by_id。
-- event_id、change_seq 是来源事件与那个变化在事件变化列表里的位置（创建本身为 -1，spec R4）：同一个收件人在未删除的
-- 行中至多一行，任务重试与重复投递不多写。

-- +goose Up
CREATE TABLE notifications (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    receiver_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    triggered_by_id uuid REFERENCES users ON DELETE SET NULL,
    issue_id uuid NOT NULL,
    issue_created_by_id uuid,
    sender varchar(20) NOT NULL CHECK (sender IN ('created', 'assigned', 'subscribed', 'mentioned')),
    data jsonb NOT NULL CHECK (
        CASE WHEN jsonb_typeof(data) = 'object' THEN
            data ?& array['issue', 'change']
            AND data - array['issue', 'change'] = '{}'::jsonb
            AND jsonb_typeof(data -> 'issue') = 'object'
            AND jsonb_typeof(data -> 'change') = 'object'
        ELSE false END
    ),
    event_id uuid NOT NULL,
    change_seq smallint NOT NULL CHECK (change_seq >= -1),
    read_at timestamptz,
    archived_at timestamptz,
    snoozed_till timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 幂等：一个事件的一个变化对一个收件人在未删除的行中至多一行
CREATE UNIQUE INDEX notifications_event_id_change_seq_receiver_id_key ON notifications (event_id, change_seq, receiver_id)
    WHERE deleted_at IS NULL;
-- 收件人在一个工作区的通知列表（新的在前）
CREATE INDEX notifications_receiver_list_idx ON notifications (receiver_id, workspace_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
-- 收件人在一个工作区的未读数（未读、未归档）
CREATE INDEX notifications_receiver_unread_idx ON notifications (receiver_id, workspace_id)
    WHERE deleted_at IS NULL AND read_at IS NULL AND archived_at IS NULL;
-- 工作项删除时在原事务里找它的通知
CREATE INDEX notifications_workspace_id_issue_id_idx ON notifications (workspace_id, issue_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX notifications_workspace_id_idx ON notifications (workspace_id);
CREATE INDEX notifications_project_id_idx ON notifications (project_id);
CREATE INDEX notifications_receiver_id_idx ON notifications (receiver_id);
-- 60 天清理找候选
CREATE INDEX notifications_deleted_at_idx ON notifications (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE notifications;
