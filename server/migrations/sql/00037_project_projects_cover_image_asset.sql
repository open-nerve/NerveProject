-- +goose Up
ALTER TABLE projects
    ADD COLUMN cover_image_asset_id uuid,
    ADD COLUMN cover_builtin_id text,
    ADD CONSTRAINT projects_cover_image_asset_fk FOREIGN KEY (cover_image_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION,
    ADD CONSTRAINT projects_cover_builtin_id_check CHECK (
        cover_builtin_id IS NULL OR
        (length(cover_builtin_id) = 8 AND cover_builtin_id ~ '^cover-(0[1-9]|1[0-9]|2[0-9])$')
    ),
    ADD CONSTRAINT projects_cover_choice_check CHECK (cover_image_asset_id IS NULL OR cover_builtin_id IS NULL);
CREATE INDEX projects_cover_image_asset_id_idx ON projects (cover_image_asset_id);

-- +goose Down
ALTER TABLE projects DROP COLUMN cover_builtin_id, DROP COLUMN cover_image_asset_id;
