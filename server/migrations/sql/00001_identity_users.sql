-- users：Plane 的 users 表（40 列）按 M2 设计 4.2 保留 10 列。约定见 M2 设计 3.13：
-- id 由应用生成（uuid.NewV7），created_at、updated_at 由应用按用例的时钟写入，DEFAULT now() 只是兜底。

-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY,
    -- 领域层先规范化（小写、去掉首尾空白）再校验；CHECK 挡住绕过应用写入的大写和空白（4.2）
    email varchar(255) NOT NULL UNIQUE CHECK (email = lower(email) AND email !~ '[[:space:]]'),
    -- argon2id 的 PHC 字符串（3.8）
    password varchar(128) NOT NULL,
    first_name varchar(255) NOT NULL DEFAULT '',
    last_name varchar(255) NOT NULL DEFAULT '',
    display_name varchar(255) NOT NULL CHECK (display_name <> ''),
    user_timezone varchar(255) NOT NULL DEFAULT 'UTC',
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
