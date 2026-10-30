/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useMemo } from "react";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "../use-detail-scope";
import type { TLinkOperations } from "../../issue-detail/links";

export function useLinkOperations(workspaceSlug: string, projectId: string, issueId: string): TLinkOperations {
  const { createLink, updateLink, removeLink } = useIssueDetail();
  const capture = useDetailScope(workspaceSlug, projectId, issueId);
  const refusal = useRefusalToast();
  const { t } = useTranslation();
  return useMemo(
    () => ({
      // The form owns success/close/refusal; preserve the actual command's outcome.
      create: async (data) => {
        await createLink(workspaceSlug, projectId, issueId, data);
      },
      update: async (linkId, data) => {
        await updateLink(workspaceSlug, projectId, issueId, linkId, data);
      },
      remove: async (linkId) => {
        const current = capture();
        if (!current()) return;
        await followInSession(() => removeLink(workspaceSlug, projectId, issueId, linkId), {
          done: () => {
            if (current())
              setToast({
                type: TOAST_TYPE.SUCCESS,
                title: t("links.toasts.removed.title"),
                message: t("links.toasts.removed.message"),
              });
          },
          failed: (error) => {
            if (current() && !(error instanceof SessionChangedError)) refusal(error);
          },
        });
      },
    }),
    [workspaceSlug, projectId, issueId, createLink, updateLink, removeLink, capture, refusal, t]
  );
}
