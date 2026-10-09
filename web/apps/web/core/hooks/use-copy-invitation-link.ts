/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyTextToClipboard } from "@nerve/utils";
// lib
import { invitationLink } from "@/lib/invitation-link";

/** Copies an invitation's link (invitationLink, at this origin) to the clipboard, and says whether it could. */
export function useCopyInvitationLink(): (invitation: Pick<WorkspaceInvitation, "id" | "token">) => Promise<void> {
  const { t } = useTranslation();
  return async (invitation) => {
    try {
      await copyTextToClipboard(invitationLink(window.location.origin, invitation));
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("entity.link_copied_to_clipboard", { entity: t("common.invite") }),
      });
    } catch {
      // the browser did not let the page write the clipboard
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t("something_went_wrong_please_try_again"),
      });
    }
  };
}
