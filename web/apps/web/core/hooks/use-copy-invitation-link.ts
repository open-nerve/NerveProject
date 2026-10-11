/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useCopyLink } from "@/hooks/use-copy-link";
// lib
import { invitationLink } from "@/lib/invitation-link";

/** Copies an invitation's link (invitationLink, at this origin) to the clipboard, and says whether it could. */
export function useCopyInvitationLink(): (invitation: Pick<WorkspaceInvitation, "id" | "token">) => Promise<void> {
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (invitation) =>
    copyLink(
      invitationLink(window.location.origin, invitation),
      t("entity.link_copied_to_clipboard", { entity: t("common.invite") })
    );
}
