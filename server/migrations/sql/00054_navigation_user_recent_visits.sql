-- user_recent_visits：Plane 的 user_recent_visits 表按 M7 设计 3.9、4.2 与 P3d spec 2.3 建。每行是一个账户在一个工作区里对
-- 一个目标的最近一次访问：两种 entity_type（issue、project；Plane 的 entity_name 改名并以 CHECK 限两值，D37），目标的 id
-- （不是外键，目标在 issue、project 各自的表里）、目标所属的项目（项目的访问为项目自己；两种都在项目里，所以非空，R2）、
-- 访问的时刻。建时由用例按业务时钟给出时刻，表上不设默认值（同 00052、00053）。
-- 每人每工作区未删的行至多 20 条：记录在同一事务里物理删排在第 20 条之后的（spec 2.5、2.8）；工作项删除的连带软删所有人
-- 指向它的行，删项目、工作区由 navigation.delete_rows 软删（spec 2.6）；60 天清理在 user_favorites 之后清它（module.go）。

-- +goose Up
CREATE TABLE user_recent_visits (
    id uuid PRIMARY KEY,
    -- 目标所在的工作区
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 目标所属的项目：项目的访问为项目自己，工作项的访问为它的项目
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    entity_type text NOT NULL,
    -- 目标的 id，不是外键
    entity_identifier uuid NOT NULL,
    visited_at timestamptz NOT NULL,
    -- 建的人是访问者；改的人是重访者，或连带时删除的人
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT user_recent_visits_entity_type_check CHECK (entity_type IN ('issue', 'project')),
    -- 项目的访问的项目是它自己（两列都非空，不用 coalesce）
    CONSTRAINT user_recent_visits_project_id_check CHECK (entity_type <> 'project' OR project_id = entity_identifier)
);
-- 同一个人在同一工作区未删的访问里一个目标只一行；记录的 upsert 以它为冲突目标，ON CONFLICT 的谓词与这里逐字相同。
-- 名字里的 entity 指 (entity_type, entity_identifier) 两列：照列名全写的
-- user_recent_visits_user_id_workspace_id_entity_type_entity_identifier_key 有 73 字节，超过 PostgreSQL 标识符的 63 字节，
-- 建索引时会被悄悄截断（只发 NOTICE），名字的守卫就钉不住。
CREATE UNIQUE INDEX user_recent_visits_user_id_workspace_id_entity_key ON user_recent_visits (user_id, workspace_id, entity_type, entity_identifier)
    WHERE deleted_at IS NULL;
-- 连带：工作项删除时按工作区与目标找所有人的访问（设计 4.3）
CREATE INDEX user_recent_visits_workspace_id_entity_identifier_idx ON user_recent_visits (workspace_id, entity_identifier) WHERE deleted_at IS NULL;
-- 列表与裁剪：本人在这个工作区的访问，按访问时刻新的在前、同一时刻按 id
CREATE INDEX user_recent_visits_user_id_workspace_id_visited_at_idx ON user_recent_visits (user_id, workspace_id, visited_at DESC, id DESC)
    WHERE deleted_at IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX user_recent_visits_workspace_id_idx ON user_recent_visits (workspace_id);
CREATE INDEX user_recent_visits_project_id_idx ON user_recent_visits (project_id);
CREATE INDEX user_recent_visits_user_id_idx ON user_recent_visits (user_id);
-- 60 天清理找候选
CREATE INDEX user_recent_visits_deleted_at_idx ON user_recent_visits (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE user_recent_visits;
