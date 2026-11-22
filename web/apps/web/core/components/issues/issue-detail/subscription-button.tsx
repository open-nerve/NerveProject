/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { SubscribeOutline, UnsubscribeOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useSubscriptionToggle } from "./use-subscription-toggle";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
};

/**
 * Subscribe or unsubscribe, in a work item's detail header and its peek's, an embedded peek's too (P4 spec 2.6, R13,
 * R14): shown while the caller may change his subscription to it (the subscription store's subscribable: a role in its
 * project, a guest's too; its detail held; neither it nor its project archived), saying which by its detail's
 * is_subscribed as nerve last read it. Out of use and saying Loading while a change of it is pending, so that it never
 * shows the change before nerve's read of the work item does. Its name is its text at every width, never an icon alone
 * (W7). It sends the change as the palette's command does (useSubscriptionToggle).
 */
export const IssueSubscriptionButton = observer(function IssueSubscriptionButton(props: Props) {
  const { workspaceSlug, projectId, issueId } = props;
  const { t } = useTranslation();
  const { subscription } = useIssueDetail();
  const toggle = useSubscriptionToggle();
  const detail = subscription.subscribable(workspaceSlug, projectId, issueId);
  if (!detail) return null;
  const pending = subscription.isPending(issueId);
  return (
    <Button
      variant="secondary"
      size="lg"
      prependIcon={detail.is_subscribed ? <UnsubscribeOutline /> : <SubscribeOutline />}
      disabled={pending}
      onClick={() => void toggle(workspaceSlug, projectId, issueId)}
    >
      {pending
        ? t("common.loading")
        : detail.is_subscribed
          ? t("common.actions.unsubscribe")
          : t("common.actions.subscribe")}
    </Button>
  );
});
