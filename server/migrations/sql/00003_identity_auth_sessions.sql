-- auth_sessions：一次登录一行，替代 Plane 的 Django 会话表 sessions（M2 设计 3.5、4.5）。
-- 旧代的刷新令牌不存：它们由令牌里的 MAC 标签认出（3.4）。

-- +goose Up
CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 当前这一代刷新令牌密文的 SHA-256
    token_hash bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    generation integer NOT NULL DEFAULT 0 CHECK (generation >= 0),
    user_agent text NOT NULL DEFAULT '',
    ip inet,
    -- 登录时刻加 auth.session_ttl，之后不变
    expires_at timestamptz NOT NULL,
    last_refreshed_at timestamptz,
    revoked_at timestamptz,
    revoke_reason varchar(20) CHECK (revoke_reason IN
        ('logout', 'password_changed', 'password_reset', 'email_changed', 'deactivated', 'reuse_detected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_sessions_revoked_consistent_check CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))
);
-- 撤销某个账户的全部会话
CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
-- 清理过期会话的定时任务（M2/P3）
CREATE INDEX auth_sessions_expires_at_idx ON auth_sessions (expires_at);

-- +goose Down
DROP TABLE auth_sessions;
