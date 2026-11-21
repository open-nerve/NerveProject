-- user_favorites：Plane 的 user_favorites 表按 M7 设计 3.8、4.2 与 P3c spec 2.3 建。每行是一个账户在一个工作区里的一个收藏：
-- 五种 entity_type（project、cycle、module、view、folder；view 兼指项目视图与工作区视图，R4），目标的 id（不是外键，
-- 目标在 project、planning、navigation 各自的表里）、目标所属的项目（项目收藏为项目自己，工作区视图与文件夹为空）、
-- 文件夹的名字、所在的文件夹（只一层）与次序。删去 Plane 的 is_folder（由 entity_type 定）、entity_data 与 project 之外的
-- 冗余外键。建时由用例按业务时钟给出时刻，表上不设默认值（同 00052）。
-- 用户删除与各种连带都是软删（R10），两条部分唯一都带 deleted_at IS NULL，软删之后能重建。
-- 删项目、工作区由 navigation.delete_rows 软删（工作区那一步只删 project_id 为空的行，spec 2.6）；60 天清理在 issue_views
-- 之后清它（module.go），物理删文件夹时外键连带物理删它的子行（子行与文件夹同一时刻软删过）。

-- +goose Up
CREATE TABLE user_favorites (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 目标所属的项目：项目收藏为项目自己，迭代、模块、项目视图为它们的项目，工作区视图与文件夹为空
    project_id uuid REFERENCES projects ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    entity_type text NOT NULL,
    -- 目标的 id，不是外键；文件夹为空
    entity_identifier uuid,
    -- 只文件夹有；去首尾空白、按字符数至多 255 由领域层校验
    name varchar(255),
    -- 所在的文件夹；物理删文件夹带走它的子行
    parent_id uuid REFERENCES user_favorites ON DELETE CASCADE,
    sequence double precision NOT NULL,
    -- 最后一次写的人：自己，或连带时触发的人
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT user_favorites_entity_type_check CHECK (entity_type IN ('project', 'cycle', 'module', 'view', 'folder')),
    -- 文件夹：有名字，没有目标、项目与父（一层）；别的：有目标，没有名字
    CONSTRAINT user_favorites_shape_check CHECK (CASE WHEN entity_type = 'folder'
            THEN entity_identifier IS NULL AND project_id IS NULL AND parent_id IS NULL AND name IS NOT NULL
            ELSE entity_identifier IS NOT NULL AND name IS NULL END),
    -- 项目收藏的项目是它自己；迭代、模块必有项目；视图的项目可空（工作区视图）
    CONSTRAINT user_favorites_project_id_check CHECK (CASE entity_type
            WHEN 'project' THEN coalesce(project_id = entity_identifier, false)
            WHEN 'cycle' THEN project_id IS NOT NULL
            WHEN 'module' THEN project_id IS NOT NULL
            ELSE true END),
    CONSTRAINT user_favorites_name_check CHECK (name <> ''),
    -- 有限值：PostgreSQL 的 NaN 等于自己，NOT IN 也拒得掉它
    CONSTRAINT user_favorites_sequence_check CHECK (sequence NOT IN ('NaN', 'Infinity', '-Infinity'))
);
-- 同一个人在未删的收藏里只收藏一个目标一次（文件夹没有目标，不进这条）；条目的插入以它为冲突目标
CREATE UNIQUE INDEX user_favorites_user_id_entity_type_entity_identifier_key ON user_favorites (user_id, entity_type, entity_identifier)
    WHERE deleted_at IS NULL;
-- 同一个人在同一工作区未删的文件夹里名字不分大小写唯一；建与改名撞上由适配器按约束名换成 409
CREATE UNIQUE INDEX user_favorites_user_id_workspace_id_name_key ON user_favorites (user_id, workspace_id, lower(name))
    WHERE entity_type = 'folder' AND deleted_at IS NULL;
-- 连带：目标删除、归档时按目标找所有人的收藏
CREATE INDEX user_favorites_entity_type_entity_identifier_idx ON user_favorites (entity_type, entity_identifier) WHERE deleted_at IS NULL;
-- 列表：本人在这个工作区的收藏
CREATE INDEX user_favorites_user_id_workspace_id_idx ON user_favorites (user_id, workspace_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX user_favorites_workspace_id_idx ON user_favorites (workspace_id);
CREATE INDEX user_favorites_project_id_idx ON user_favorites (project_id);
CREATE INDEX user_favorites_user_id_idx ON user_favorites (user_id);
CREATE INDEX user_favorites_parent_id_idx ON user_favorites (parent_id);
-- 60 天清理找候选
CREATE INDEX user_favorites_deleted_at_idx ON user_favorites (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE user_favorites;
