-- projects：Plane 的 projects 表（36 列）按 M3 设计 4.6 保留 22 列，另加计数列 last_issue_sequence，共 23 列。
-- 负责人、默认负责人改为 ON DELETE SET NULL（3.15）；last_issue_sequence 是工作项编号的计数列，M4 取号，M3 不读写它。
-- 删除工作区时随之软删除（DeleteWorkspaceProjects，3.3）。

-- +goose Up
CREATE TABLE projects (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- Plane 的禁用字符（serializers/project.py:39-44，3.19）
    name varchar(255) NOT NULL CHECK (name <> '' AND name !~ '[&+,:;$^}{*=?@#\|''<>.()%!-]'),
    description text NOT NULL DEFAULT '',
    -- 转成大写后 1–10 个，只能是 A-Z、0-9 和 ÇŞĞİÖÜ（3.19）；列类型仍是 varchar(12)
    identifier varchar(12) NOT NULL CHECK (identifier ~ '^[A-Z0-9ÇŞĞİÖÜ]{1,10}$'),
    -- 0 私密、2 公开（快照的 >= 0 收紧）
    network smallint NOT NULL DEFAULT 2 CHECK (network IN (0, 2)),
    project_lead_id uuid REFERENCES users ON DELETE SET NULL,
    default_assignee_id uuid REFERENCES users ON DELETE SET NULL,
    cycle_view boolean NOT NULL DEFAULT false,
    module_view boolean NOT NULL DEFAULT false,
    issue_views_view boolean NOT NULL DEFAULT false,
    intake_view boolean NOT NULL DEFAULT false,
    guest_view_all_features boolean NOT NULL DEFAULT false,
    -- 模型的验证器 0–12
    archive_in integer NOT NULL DEFAULT 0 CHECK (archive_in BETWEEN 0 AND 12),
    -- {} 表示没有图标；键和值类型与接口的结构相同，字段都可选（3.19、5.2）
    logo_props jsonb NOT NULL DEFAULT '{}',
    -- 取值在领域层校验（3.13）
    timezone varchar(255) NOT NULL DEFAULT 'UTC',
    last_issue_sequence integer NOT NULL DEFAULT 0 CHECK (last_issue_sequence >= 0),
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    -- 是对象、键的集合、出现的每个键的值类型，嵌套的 emoji、icon 也查到底（M2 设计 3.13；M3 设计 4.6）。
    -- 嵌套的 -> 要加括号：- 的优先级高于 ->
    CONSTRAINT projects_logo_props_check CHECK (CASE WHEN jsonb_typeof(logo_props) = 'object' THEN
            (logo_props - array['in_use', 'emoji', 'icon']) = '{}'::jsonb
        AND (NOT logo_props ? 'in_use' OR logo_props -> 'in_use' IN ('"emoji"'::jsonb, '"icon"'::jsonb))
        AND (NOT logo_props ? 'emoji' OR CASE WHEN jsonb_typeof(logo_props -> 'emoji') = 'object' THEN
                ((logo_props -> 'emoji') - array['value', 'url']) = '{}'::jsonb
            AND (NOT (logo_props -> 'emoji') ? 'value' OR jsonb_typeof(logo_props -> 'emoji' -> 'value') = 'string')
            AND (NOT (logo_props -> 'emoji') ? 'url' OR jsonb_typeof(logo_props -> 'emoji' -> 'url') = 'string')
          ELSE false END)
        AND (NOT logo_props ? 'icon' OR CASE WHEN jsonb_typeof(logo_props -> 'icon') = 'object' THEN
                ((logo_props -> 'icon') - array['name', 'color', 'background_color']) = '{}'::jsonb
            AND (NOT (logo_props -> 'icon') ? 'name' OR jsonb_typeof(logo_props -> 'icon' -> 'name') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'color' OR jsonb_typeof(logo_props -> 'icon' -> 'color') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'background_color'
                 OR jsonb_typeof(logo_props -> 'icon' -> 'background_color') = 'string')
          ELSE false END)
      ELSE false END)
);
-- 标识、名称在工作区的未删除项目中唯一（3.19）；删除之后可以再用。按工作区列出项目也用后一个
CREATE UNIQUE INDEX projects_workspace_id_identifier_key ON projects (workspace_id, identifier) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX projects_workspace_id_name_key ON projects (workspace_id, name) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX projects_workspace_id_idx ON projects (workspace_id);

-- +goose Down
DROP TABLE projects;
