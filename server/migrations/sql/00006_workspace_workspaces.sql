-- workspaces：Plane 的 workspaces 表（14 列）按 M3 设计 4.2 保留 10 列。约定见 M2 设计 3.13：
-- id 由应用生成（uuid.NewV7），created_at、updated_at 由应用按用例的时钟写入，DEFAULT now() 只是兜底。

-- +goose Up
CREATE TABLE workspaces (
    id uuid PRIMARY KEY,
    -- "至少一个字母或数字、不含网址"在领域层（M3 设计 3.10）
    name varchar(80) NOT NULL CHECK (name <> ''),
    -- 只有小写（3.10）；保留名在领域层
    slug varchar(48) NOT NULL CHECK (slug ~ '^[a-z0-9_-]+$'),
    -- 前端的选项（web/packages/constants/src/workspace.ts:10）
    organization_size varchar(20)
        CHECK (organization_size IN ('Just myself', '2-10', '11-50', '51-200', '201-500', '500+')),
    -- 取值在领域层校验（3.13）
    timezone varchar(255) NOT NULL DEFAULT 'UTC',
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- Plane 的全表唯一 workspace_slug_key 改为部分唯一：删除之后 slug 可以再用，不改名（3.10）
CREATE UNIQUE INDEX workspaces_slug_key ON workspaces (slug) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE workspaces;
