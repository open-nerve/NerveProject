-- workspace_member_invites：Plane 的 workspace_member_invites 表（13 列）按 M3 设计 4.4 保留 11 列。
-- 不存令牌（3.8）：链接里的令牌由签名密钥派生的 MAC 从邀请的 id 算出，删除 token 列；message 没有写入方，删除。
-- 接受时记 accepted、responded_at 并软删除这一行；忽略时只记 responded_at，这一行留着、仍占着这个邮箱（3.8）。

-- +goose Up
CREATE TABLE workspace_member_invites (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 存规范化之后的邮箱（3.13），与 users.email 的 CHECK 相同，另外不能为空
    email varchar(255) NOT NULL CHECK (email <> '' AND email = lower(email) AND email !~ '[[:space:]]'),
    -- 访客 5、成员 15、管理员 20，同 workspace_members.role
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    accepted boolean NOT NULL DEFAULT false,
    responded_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    -- 接受的邀请一定有回应的时刻
    CONSTRAINT workspace_member_invites_responded_check CHECK (responded_at IS NOT NULL OR NOT accepted)
);
-- 一个工作区里一个邮箱至多一份未删除的邀请；已忽略的没有删除，仍占着这个邮箱（3.8）
CREATE UNIQUE INDEX workspace_member_invites_workspace_id_email_key ON workspace_member_invites (workspace_id, email)
    WHERE deleted_at IS NULL;
-- 停用按邮箱删除发给它的邀请（3.9，P6）；注册策略按邀请的 id 读，不用它（4.4）
CREATE INDEX workspace_member_invites_email_idx ON workspace_member_invites (email) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_member_invites_workspace_id_idx ON workspace_member_invites (workspace_id);

-- +goose Down
DROP TABLE workspace_member_invites;
