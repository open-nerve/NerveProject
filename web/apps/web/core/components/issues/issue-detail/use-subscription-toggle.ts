/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { followInSession } from "@/lib/in-session";

/**
 * Changes the caller's subscription to a work item (P4 spec 2.6, R13), as its detail's button, its peek's and the
 * palette's command do: unsubscribes him if its detail says he subscribes, else subscribes him, through the
 * subscription store (the work item's queue, then nerve's read of it again); nothing while he may not
 * (subscribable) or a change of it is pending. Says it once nerve has made it, or why nerve refused (useRefusalToast),
 * never both, only in the session it was sent in (followInSession); nothing for a change the work item's scope dropped
 * while it waited (SessionChangedError: its project's read changed, its issues retired).
 */
export function useSubscriptionToggle(): (workspaceSlug: string, projectId: string, issueId: string) => Promise<void> {
  const { t } = useTranslation();
  const { subscription } = useIssueDetail();
  const refusalToast = useRefusalToast();
  return async (workspaceSlug, projectId, issueId) => {
    const detail = subscription.subscribable(workspaceSlug, projectId, issueId);
    if (!detail || subscription.isPending(issueId)) return;
    const subscribed = detail.is_subscribed;
    const change = subscribed ? subscription.unsubscribe : subscription.subscribe;
    await followInSession(() => change(workspaceSlug, projectId, issueId), {
      done: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: subscribed
            ? t("issue.subscription.actions.unsubscribed")
            : t("issue.subscription.actions.subscribed"),
        }),
      failed: (error) => {
        if (!(error instanceof SessionChangedError)) refusalToast(error);
      },
    });
  };
}
