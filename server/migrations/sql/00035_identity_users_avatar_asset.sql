-- +goose Up
ALTER TABLE users
    ADD COLUMN avatar_asset_id uuid,
    ADD COLUMN cover_image_asset_id uuid,
    ADD CONSTRAINT users_avatar_asset_fk FOREIGN KEY (avatar_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION,
    ADD CONSTRAINT users_cover_image_asset_fk FOREIGN KEY (cover_image_asset_id) REFERENCES file_assets(id) ON DELETE NO ACTION;
CREATE INDEX users_avatar_asset_id_idx ON users (avatar_asset_id);
CREATE INDEX users_cover_image_asset_id_idx ON users (cover_image_asset_id);

-- +goose Down
ALTER TABLE users DROP COLUMN cover_image_asset_id, DROP COLUMN avatar_asset_id;
