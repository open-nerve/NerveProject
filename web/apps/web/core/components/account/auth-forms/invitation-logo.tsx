/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { InvitationPreview } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { WorkspaceLogo } from "@/components/workspace/logo";

type Props = {
  invitation: InvitationPreview;
  error: unknown;
  reload(): Promise<InvitationPreview | undefined>;
};

/** The public link owns its retry budget; no account, workspace resource or asset ID is required. */
export function InvitationLogo({ invitation, error, reload }: Props) {
  const { t } = useTranslation();
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [loading, setLoading] = useState(false);
  const life = useRef({ active: false, automatic: true, pending: false, invitation, attempt });
  useLayoutEffect(() => {
    life.current.invitation = invitation;
    life.current.attempt = attempt;
  }, [invitation, attempt]);
  useLayoutEffect(() => {
    const owner = life.current;
    owner.active = true;
    return () => {
      owner.active = false;
    };
  }, []);
  const current = useCallback(
    () => life.current.active && life.current.invitation === invitation && life.current.attempt === attempt,
    [invitation, attempt]
  );
  const refresh = useCallback(async () => {
    if (!current() || life.current.pending) return;
    life.current.pending = true;
    setLoading(true);
    try {
      await reload();
      if (life.current.active) {
        setFailed(false);
        setAttempt((value) => value + 1);
      }
    } catch {
      // SWR exposes the public refusal. Only an explicit retry may ask again.
    } finally {
      life.current.pending = false;
      if (life.current.active) setLoading(false);
    }
  }, [current, reload]);
  const invalid = useCallback(() => {
    if (!current()) return;
    setFailed(true);
    if (life.current.automatic) {
      life.current.automatic = false;
      void refresh();
    }
  }, [current, refresh]);
  const loaded = () => {
    if (current() && !failed && error === undefined) life.current.automatic = true;
  };
  return (
    <>
      <WorkspaceLogo
        key={attempt}
        source={{
          kind: "public-preview",
          url: failed || error !== undefined ? null : invitation.workspace_logo_url,
          expiresAt: invitation.workspace_logo_url_expires_at,
          onExpire: invalid,
          onError: invalid,
          onLoad: loaded,
        }}
        name={invitation.workspace_name}
        classNames="size-9 flex-shrink-0"
      />
      {(failed || error !== undefined) && (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          disabled={loading}
          onClick={() => {
            life.current.automatic = false;
            void refresh();
          }}
        >
          {t("image_upload.retry_image")}
        </Button>
      )}
    </>
  );
}
