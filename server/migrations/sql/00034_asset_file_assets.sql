-- +goose Up
CREATE TABLE file_assets (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL CONSTRAINT file_assets_user_fk REFERENCES users ON DELETE NO ACTION,
    workspace_id uuid CONSTRAINT file_assets_workspace_fk REFERENCES workspaces ON DELETE NO ACTION,
    project_id uuid CONSTRAINT file_assets_project_fk REFERENCES projects ON DELETE NO ACTION,
    issue_id uuid CONSTRAINT file_assets_issue_fk REFERENCES issues ON DELETE NO ACTION,
    comment_id uuid CONSTRAINT file_assets_comment_fk REFERENCES issue_comments ON DELETE NO ACTION,
    draft_issue_id uuid CONSTRAINT file_assets_draft_issue_fk REFERENCES draft_issues ON DELETE NO ACTION,
    entity_type text NOT NULL,
    scope_kind text NOT NULL,
    attributes jsonb NOT NULL,
    size bigint NOT NULL CHECK (size BETWEEN 1 AND 104857600),
    is_uploaded boolean NOT NULL DEFAULT false,
    is_deleted boolean NOT NULL DEFAULT false,
    bound_at timestamptz,
    upload_expires_at timestamptz,
    storage_metadata jsonb NOT NULL,
    storage_revision bigint NOT NULL DEFAULT 0 CHECK (storage_revision >= 0),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    deleted_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT file_assets_deletion_check CHECK (is_deleted = (deleted_at IS NOT NULL)),
    CONSTRAINT file_assets_attributes_check CHECK ((
        jsonb_typeof(attributes) = 'object'
        AND jsonb_typeof(attributes->'name') = 'string'
        AND char_length(attributes->>'name') BETWEEN 1 AND 255
        AND jsonb_typeof(attributes->'content_type') = 'string'
        AND char_length(attributes->>'content_type') > 0
    ) IS TRUE),
    CONSTRAINT file_assets_scope_check CHECK (
        (scope_kind = 'user' AND entity_type IN ('USER_AVATAR','USER_COVER')
            AND workspace_id IS NULL AND project_id IS NULL AND issue_id IS NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'workspace' AND entity_type = 'WORKSPACE_LOGO'
            AND workspace_id IS NOT NULL AND project_id IS NULL AND issue_id IS NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'new_project' AND entity_type = 'PROJECT_COVER'
            AND workspace_id IS NOT NULL AND project_id IS NULL AND issue_id IS NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'project' AND entity_type = 'PROJECT_COVER'
            AND workspace_id IS NOT NULL AND project_id IS NOT NULL AND issue_id IS NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'issue' AND entity_type IN ('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION')
            AND workspace_id IS NOT NULL AND project_id IS NOT NULL AND issue_id IS NOT NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'new_comment' AND entity_type IN ('COMMENT_ATTACHMENT','COMMENT_DESCRIPTION')
            AND workspace_id IS NOT NULL AND project_id IS NOT NULL AND issue_id IS NOT NULL AND comment_id IS NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'comment' AND entity_type IN ('COMMENT_ATTACHMENT','COMMENT_DESCRIPTION')
            AND workspace_id IS NOT NULL AND project_id IS NOT NULL AND issue_id IS NOT NULL AND comment_id IS NOT NULL AND draft_issue_id IS NULL)
        OR (scope_kind = 'draft' AND entity_type IN ('DRAFT_ISSUE_ATTACHMENT','DRAFT_ISSUE_DESCRIPTION')
            AND workspace_id IS NOT NULL AND issue_id IS NULL AND comment_id IS NULL AND draft_issue_id IS NOT NULL)
    ),
    CONSTRAINT file_assets_storage_metadata_check CHECK ((
        jsonb_typeof(storage_metadata) = 'object'
        AND octet_length(storage_metadata::text) <= 16384
        AND storage_metadata ?& ARRAY['schema_version','origin','staging','final','candidates']
        AND storage_metadata - ARRAY['schema_version','origin','staging','final','candidates'] = '{}'::jsonb
        AND storage_metadata->'schema_version' = '1'::jsonb
        AND storage_metadata->>'origin' IN ('upload','copy')
        AND jsonb_typeof(storage_metadata->'staging') IN ('object','null')
        AND jsonb_typeof(storage_metadata->'final') IN ('object','null')
        AND (storage_metadata->>'origin' <> 'copy' OR storage_metadata->'staging' = 'null'::jsonb)
        AND CASE WHEN jsonb_typeof(storage_metadata->'candidates') = 'array'
                 THEN jsonb_array_length(storage_metadata->'candidates') <= 3 ELSE false END
    ) IS TRUE)
);
CREATE INDEX file_assets_user_id_idx ON file_assets (user_id);
CREATE INDEX file_assets_workspace_id_idx ON file_assets (workspace_id);
CREATE INDEX file_assets_project_id_idx ON file_assets (project_id);
CREATE INDEX file_assets_issue_id_idx ON file_assets (issue_id);
CREATE INDEX file_assets_comment_id_idx ON file_assets (comment_id);
CREATE INDEX file_assets_draft_issue_id_idx ON file_assets (draft_issue_id);
CREATE INDEX file_assets_deleted_at_idx ON file_assets (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX file_assets_created_at_id_idx ON file_assets (created_at, id) WHERE deleted_at IS NULL AND bound_at IS NULL;

-- +goose Down
DROP TABLE file_assets;
