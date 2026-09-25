-- profiles：Plane 的 profiles 表（29 列）按 M2 设计 4.3 保留 11 列，默认值和取值范围取自 Plane 的模型。

-- +goose Up
CREATE TABLE profiles (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL UNIQUE REFERENCES users ON DELETE CASCADE,
    theme varchar(20) NOT NULL DEFAULT 'system'
        CHECK (theme IN ('system', 'light', 'dark', 'light-contrast', 'dark-contrast')),
    is_tour_completed boolean NOT NULL DEFAULT false,
    -- 恰好这四个键、值都是布尔值；外层的 CASE 让数组、标量和 JSON null 也得到 check_violation（3.13）
    onboarding_step jsonb NOT NULL
        DEFAULT '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'
        CHECK (CASE WHEN jsonb_typeof(onboarding_step) = 'object' THEN
            onboarding_step ?& array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']
            AND (onboarding_step - array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']) = '{}'::jsonb
            AND jsonb_typeof(onboarding_step -> 'profile_complete') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_create') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_invite') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_join') = 'boolean'
        ELSE false END),
    is_onboarded boolean NOT NULL DEFAULT false,
    -- 客户端写入的工作区 id，不是外键（和 Plane 一样，3.2）
    last_workspace_id uuid,
    language varchar(255) NOT NULL DEFAULT 'en' CHECK (language IN ('en', 'zh-CN')),
    start_of_the_week smallint NOT NULL DEFAULT 0 CHECK (start_of_the_week BETWEEN 0 AND 6),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE profiles;
