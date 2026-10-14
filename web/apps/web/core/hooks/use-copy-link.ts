/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyTextToClipboard, copyUrlToClipboard } from "@nerve/utils";

/**
 * Writes a text to the clipboard by write, and says whether it could (M3 design 7.7): the success under the title of
 * titleKey, its message copied, which names what was copied; the browser may not let the page write the clipboard (no
 * permission, or plain http), which the failure says. Every copy of the web app is one of this module's (the keyword
 * rule copy-hook).
 */
function useCopy(
  write: (text: string) => Promise<void>,
  titleKey: string
): (text: string, copied: string) => Promise<void> {
  const { t } = useTranslation();
  return async (text, copied) => {
    try {
      await write(text);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t(titleKey), message: copied });
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

/** Copies a link, a path of this origin or a whole address, as useCopy does, under "Link copied". */
export function useCopyLink(): (link: string, copied: string) => Promise<void> {
  return useCopy(copyUrlToClipboard, "common.link_copied");
}

/** Copies a text as it is (a token, say), as useCopy does, under "Success". */
export function useCopyText(): (text: string, copied: string) => Promise<void> {
  return useCopy(copyTextToClipboard, "toast.success");
}

/** Copies the link of a project of the address's workspace, its work items' page, as useCopyLink. */
export function useCopyProjectLink(): (projectId: string) => Promise<void> {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (projectId) =>
    copyLink(`/${workspaceSlug}/projects/${projectId}/issues`, t("project_link_copied_to_clipboard"));
}
