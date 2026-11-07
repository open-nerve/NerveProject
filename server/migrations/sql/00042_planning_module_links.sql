-- module_links：Plane 的 module_links 表（12 列）按 M6 设计 4.2 与 P2 spec 2.2 保留 12 列，写法照 M4 的 issue_links。
-- url 由 Plane 的 varchar(200) 改为 text、非空且不超过 2048；metadata 由后台抓取写入，约束为与 issue_links 相同的键、
-- 每个值为字符串或 null（Plane 不约束）。同一模块的未删除链接中地址唯一（Plane 只在代码里按原样比较、不加锁）。

-- +goose Up
CREATE TABLE module_links (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    module_id uuid NOT NULL REFERENCES modules ON DELETE CASCADE,
    title varchar(255),
    url text NOT NULL CHECK (url <> '' AND length(url) <= 2048),
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (
        CASE WHEN jsonb_typeof(metadata) = 'object' THEN
            metadata - ARRAY['title', 'favicon', 'url', 'favicon_url', 'error'] = '{}'::jsonb
            AND NOT jsonb_path_exists(metadata, 'strict $.* ? (@.type() != "string" && @.type() != "null")')
        ELSE false END
    ),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    -- B-tree 唯一索引容不下所有合法的 2048 字符多字节地址，所以用 hash 的排除约束，比较仍是原文相等（同 issue_links，M4 设计）
    CONSTRAINT module_links_module_id_url_key EXCLUDE USING hash ((module_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)
);
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX module_links_module_id_idx ON module_links (module_id);
CREATE INDEX module_links_workspace_id_idx ON module_links (workspace_id);
CREATE INDEX module_links_project_id_idx ON module_links (project_id);
-- 60 天清理找候选
CREATE INDEX module_links_deleted_at_idx ON module_links (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE module_links;
