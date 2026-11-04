/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
import { ProfileImage } from "@/components/assets/profile-image";

type PublicPreview = { kind: "public-preview"; url: string | null; expiresAt: string | null };
type Props = {
  source: { kind: "workspace"; workspaceID: string | undefined } | PublicPreview;
  name: string | undefined;
  classNames?: string;
};

export const WorkspaceLogo = observer(function WorkspaceLogo({ source, name, classNames }: Props) {
  const { t } = useTranslation();
  const className = cn(
    "relative grid h-6 w-6 shrink-0 place-items-center rounded-md bg-accent-primary object-cover text-on-color uppercase",
    classNames
  );
  const alt = t("aria_labels.projects_sidebar.workspace_logo");
  const fallback = name?.[0] ?? "...";
  if (source.kind === "public-preview")
    return <PublicLogo source={source} alt={alt} className={className} fallback={fallback} />;
  return source.workspaceID ? (
    <ProfileImage
      source={{ kind: "workspace", workspaceID: source.workspaceID }}
      alt={alt}
      className={className}
      fallback={fallback}
    />
  ) : (
    <span role="img" aria-label={alt} className={className}>
      {fallback}
    </span>
  );
});

/** Explicit invitation capabilities have expiry, but no dependency on a signed-in RootStore. */
function PublicLogo({
  source,
  alt,
  className,
  fallback,
}: {
  source: PublicPreview;
  alt: string;
  className: string;
  fallback: string;
}) {
  const stamp = JSON.stringify([source.url, source.expiresAt]);
  const [failed, setFailed] = useState<string>();
  const [, redraw] = useState(0);
  const expiry = source.expiresAt ? Date.parse(source.expiresAt) : NaN;
  const valid = !!source.url && Number.isFinite(expiry) && expiry > Date.now() && stamp !== failed;
  useEffect(() => {
    if (!valid) return;
    let timer: ReturnType<typeof setTimeout>;
    const check = () => {
      clearTimeout(timer);
      if (expiry <= Date.now()) redraw((value) => value + 1);
      else timer = setTimeout(check, Math.min(expiry - Date.now(), 2_147_483_647));
    };
    check();
    document.addEventListener("visibilitychange", check);
    window.addEventListener("focus", check);
    window.addEventListener("online", check);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", check);
      window.removeEventListener("focus", check);
      window.removeEventListener("online", check);
    };
  }, [stamp, expiry, valid]);
  return valid && source.url ? (
    <img key={stamp} src={source.url} alt={alt} className={className} onError={() => setFailed(stamp)} />
  ) : (
    <span role="img" aria-label={alt} className={className}>
      {fallback}
    </span>
  );
}
