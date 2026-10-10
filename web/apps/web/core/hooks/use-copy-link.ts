/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyUrlToClipboard } from "@nerve/utils";

/**
 * Copies a link to the clipboard, and says whether it could (M3 design 7.7): copied names what was copied, the
 * success's message; the browser may not let the page write the clipboard (no permission, or plain http), which the
 * failure says. The link is a path of this origin, or a whole address.
 */
export function useCopyLink(): (link: string, copied: string) => Promise<void> {
  const { t } = useTranslation();
  return async (link, copied) => {
    try {
      await copyUrlToClipboard(link);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.link_copied"), message: copied });
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

/** Copies the link of a project of the address's workspace, its work items' page, as useCopyLink. */
export function useCopyProjectLink(): (projectId: string) => Promise<void> {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (projectId) =>
    copyLink(`/${workspaceSlug}/projects/${projectId}/issues`, t("project_link_copied_to_clipboard"));
}
