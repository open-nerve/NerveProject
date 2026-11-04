/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useState, type ReactNode } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { useUser } from "@/hooks/store/user";
import { useProfileImage } from "@/hooks/use-profile-image";
import { ImageUploadModal } from "./image-upload-modal";
import { ProfileImage } from "./profile-image";

/** Only the selected account field is saved. Names and onboarding completion belong to their own forms. */
type Props = { image: "avatar" | "cover"; preview?: ReactNode; showLabel?: boolean };
export const UserImageUpload = observer(function UserImageUpload({ image, preview, showLabel = true }: Props) {
  const [open, setOpen] = useState(false);
  const { t } = useTranslation();
  const { data: user, updateCurrentUser } = useUser();
  const source = { kind: "self", image } as const;
  const read = useProfileImage(source);
  const title = t(image === "avatar" ? "image_upload.change_avatar" : "image_upload.change_cover");
  return (
    <>
      {preview && (
        <button
          type="button"
          aria-label={title}
          className="rounded-lg focus-visible:outline-2 focus-visible:outline-accent-strong"
          onClick={() => setOpen(true)}
          disabled={!user}
        >
          {preview}
        </button>
      )}
      {showLabel && (
        <Button type="button" variant="secondary" size="sm" onClick={() => setOpen(true)} disabled={!user}>
          {title}
        </Button>
      )}
      {read.error !== undefined && (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          disabled={read.loading}
          onClick={() => {
            void read.retry().catch(() => {
              /* The image reader retains its refusal for another explicit attempt. */
            });
          }}
        >
          {t("image_upload.retry_image")}
        </Button>
      )}
      <ImageUploadModal
        isOpen={open}
        title={title}
        target={{ purpose: image === "avatar" ? "USER_AVATAR" : "USER_COVER", scope: { kind: "user" } }}
        currentImage={
          <ProfileImage source={source} alt={title} className="max-h-72 max-w-full rounded-md object-contain" />
        }
        canRemove={!!(image === "avatar" ? user?.avatar_asset_id : user?.cover_image_asset_id)}
        onSubmit={async (asset) => {
          await updateCurrentUser(
            image === "avatar" ? { avatar_asset_id: asset?.id ?? null } : { cover_image_asset_id: asset?.id ?? null }
          );
        }}
        onClose={() => setOpen(false)}
      />
    </>
  );
});
