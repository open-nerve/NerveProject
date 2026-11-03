-- +goose Up
ALTER TABLE workspaces
    ADD COLUMN logo_asset_id uuid,
    ADD CONSTRAINT workspaces_logo_asset_fk FOREIGN KEY (logo_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION;
CREATE INDEX workspaces_logo_asset_id_idx ON workspaces (logo_asset_id);

-- +goose Down
ALTER TABLE workspaces DROP COLUMN logo_asset_id;
