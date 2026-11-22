/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateCompact } from "@nerve/propel/empty-state";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
// components
import { IssuePeekOverview } from "@/components/issues/peek-overview";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUserPermissions } from "@/hooks/store/user";
import { useProjectRead } from "@/hooks/use-project-read";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { problemMessageKey } from "@/lib/error-messages";
// local imports
import type { NotificationSelection } from "./page-state";
import { useNotificationPageState } from "./page-state";
import { NotificationsLoader } from "./sidebar/loader";

/**
 * The notifications page's preview (P4 spec 2.6, R12), a region named so beside the list: that nothing is selected, or
 * the selected notification's work item (NotificationPreview), one notification's at a time. The page leaving clears
 * the peek it set (W6).
 */
export const NotificationsRoot = observer(function NotificationsRoot() {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const { selected } = useNotificationPageState();
  const { setPeekIssue } = useIssueDetail();
  useEffect(() => () => setPeekIssue(undefined), [setPeekIssue]);

  return (
    <section aria-label={t("notification.a11y.preview")} className="size-full">
      {!workspace || !selected ? (
        <div className="flex size-full items-center justify-center">
          <EmptyStateCompact assetKey="unknown" assetClassName="size-20" />
        </div>
      ) : (
        <NotificationPreview key={selected.notificationId} workspaceSlug={workspace.slug} selected={selected} />
      )}
    </section>
  );
});

type PreviewProps = { workspaceSlug: string; selected: NotificationSelection };

/**
 * The selected notification's work item, the peek embedded in the page and scoped to the notification, which closes by
 * clearing the selection; an intake item's as any other's (its triage is P5's). The peek shows only once the caller's
 * role in the work item's project is known (IssuePeekOverview): a project he holds no role in (an archived one's, which
 * the workspace's list leaves out) is read first (useProjectRead), and the preview says it is loading meanwhile; each
 * preview judges by its own read, never by an answer SWR keeps from an earlier one (whole-branch m1). Once that read
 * settles and he still has no role (nerve refuses it: he has left the project, or nerve was not reached; or it gives
 * the project but no role in it), the selection is cleared and a toast says why: never a blank pane (re-93 I4).
 */
const NotificationPreview = observer(function NotificationPreview({ workspaceSlug, selected }: PreviewProps) {
  const { t } = useTranslation();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { setPeekIssue } = useIssueDetail();
  const { clearSelection } = useNotificationPageState();
  const refusalToast = useRefusalToast();
  const { notificationId, issueId, projectId } = selected;
  const readable = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined;
  const read = useProjectRead(readable ? undefined : projectId, { revalidateOnMount: false });
  const { mutate } = read;
  // whether this preview's own read of the project has answered: what SWR keeps of an earlier preview's is not its own
  const [answered, setAnswered] = useState(false);
  useEffect(() => {
    if (readable) return;
    let previewing = true;
    // mutate reads the project anew, never joining an earlier read, and settles once SWR holds its answer
    void mutate().finally(() => {
      if (previewing) setAnswered(true);
    });
    return () => {
      previewing = false;
    };
  }, [readable, mutate]);
  // this preview's read settled and the caller has no role still: refused, or answered without one
  const unreadable =
    !readable && answered && !read.isValidating && (read.error !== undefined || read.data !== undefined);
  // whether the preview has said why it cannot show the work item, which it says once
  const said = useRef(false);

  useEffect(() => {
    if (readable) setPeekIssue({ workspaceSlug, projectId, issueId });
  }, [readable, workspaceSlug, projectId, issueId, setPeekIssue]);

  useEffect(() => {
    if (!unreadable || said.current) return;
    said.current = true;
    if (read.error !== undefined) refusalToast(read.error);
    else
      setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(problemMessageKey("issue.not_found")) });
    clearSelection(notificationId);
  }, [unreadable, read.error, refusalToast, t, clearSelection, notificationId]);

  if (!readable) return <NotificationsLoader />;
  return (
    <IssuePeekOverview
      hostScope={JSON.stringify(["notification", workspaceSlug, notificationId])}
      embedIssue
      embedRemoveCurrentNotification={() => clearSelection(notificationId)}
    />
  );
});
