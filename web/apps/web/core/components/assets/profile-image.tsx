/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Avatar, type AvatarProps } from "@makeplane/propel/components/avatar";
import { cn } from "@nerve/utils";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { ProfileImageSource } from "@/store/asset/profile-images";
import { useProfileImage } from "@/hooks/use-profile-image";

type ImageProps = { source: ProfileImageSource; alt: string; className?: string; fallback?: ReactNode };
export const ProfileImage = observer(function ProfileImage({ source, alt, className, fallback }: ImageProps) {
  const image = useProfileImage(source);
  return image.url ? (
    <img
      key={image.url}
      src={image.url}
      alt={alt}
      className={className}
      onError={image.onError}
      onLoad={image.onLoad}
    />
  ) : (
    <span
      role={alt ? "img" : undefined}
      aria-label={alt || undefined}
      aria-hidden={!alt || undefined}
      className={className}
    >
      {fallback}
    </span>
  );
});
type ProfileAvatarProps = Omit<AvatarProps, "src" | "onLoad" | "onError" | "children"> & {
  source: ProfileImageSource;
  className?: string;
};
export const ProfileAvatar = observer(function ProfileAvatar({ source, className, ...props }: ProfileAvatarProps) {
  const image = useProfileImage(source);
  // Keep the existing initials, size and tooltip; the capability image has an explicit mount lifetime.
  return (
    <span className={cn("relative inline-flex shrink-0", className)}>
      <Avatar {...props} />
      {image.url && (
        <img
          key={image.url}
          src={image.url}
          alt=""
          aria-hidden
          className="pointer-events-none absolute inset-0 size-full rounded-full object-cover"
          onError={image.onError}
          onLoad={image.onLoad}
        />
      )}
    </span>
  );
});

/** Kept outside the image so consumers can put the picture inside another button without nesting controls. */
export const ProfileImageRetry = observer(function ProfileImageRetry({ source }: { source: ProfileImageSource }) {
  const image = useProfileImage(source);
  const { t } = useTranslation();
  if (image.error === undefined) return null;
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      disabled={image.loading}
      onClick={() => {
        void image.retry().catch(() => {
          /* The reader retains the refusal for another explicit attempt. */
        });
      }}
    >
      {t("image_upload.retry_image")}
    </Button>
  );
});
