-- api_tokens：个人访问令牌（PAT）。Plane 的 api_tokens 表（17 列）按 M2 设计 4.4 保留 12 列；
-- 原来存明文的 token 列改为只存整个令牌的 SHA-256（3.4）。

-- +goose Up
CREATE TABLE api_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 整个令牌（nrv_pat_ 加 43 个字符）的 SHA-256；令牌原文只在创建时返回一次
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    -- 不传时由应用生成 32 位十六进制（Plane 的 uuid4().hex）
    label varchar(255) NOT NULL CHECK (label <> ''),
    description text NOT NULL DEFAULT '',
    -- 为空表示永不过期；创建时必须晚于当前时刻（领域层校验）
    expired_at timestamptz,
    -- 最多每分钟写一次（3.5）
    last_used timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- 撤销就是软删除
    deleted_at timestamptz
);
-- 列表：未撤销的令牌按创建时间倒序，同一时刻按 id 倒序（5.2）
CREATE INDEX api_tokens_user_id_created_at_idx ON api_tokens (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE api_tokens;
