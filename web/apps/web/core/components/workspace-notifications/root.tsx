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
import { InboxContentRoot } from "@/components/inbox/content";
import { useIssueOperations } from "@/components/issues/issue-detail/use-issue-operations";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { IssuePeekOverviewError } from "@/components/issues/peek-overview/error";
import { usePeekIssueRead } from "@/components/issues/peek-overview/use-peek-issue-read";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUserPermissions } from "@/hooks/store/user";
import { useProjectRead } from "@/hooks/use-project-read";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useSession } from "@/lib/auth/use-session";
import { problemMessageKey } from "@/lib/error-messages";
// store
import { inIntake } from "@/store/issue/intake/status";
// local imports
import { useNotificationLanding } from "./landing";
import type { NotificationSelection } from "./page-state";
import { useNotificationPageState } from "./page-state";
import { NotificationsLoader } from "./sidebar/loader";

/** The preview's region, focusable from script (the landing's last place): its ring inside its edge, as the list's. */
const REGION = "focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong";

/**
 * The notifications page's preview (P4 spec 2.6, R12), a region named so beside the list: that nothing is selected, or
 * the selected notification's work item (NotificationPreview), one notification's at a time. It holds the page's
 * landing (useNotificationLanding, P5b R26), which its preview's leaving never unmounts, its region the landing's last
 * place. The page leaving clears the peek it set (W6).
 */
export const NotificationsRoot = observer(function NotificationsRoot() {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const { selected } = useNotificationPageState();
  const { setPeekIssue } = useIssueDetail();
  const landing = useNotificationLanding();
  useEffect(() => () => setPeekIssue(undefined), [setPeekIssue]);

  return (
    <section
      ref={landing.preview}
      tabIndex={-1}
      aria-label={t("notification.a11y.preview")}
      className={`size-full ${REGION}`}
    >
      {!workspace || !selected ? (
        <div className="flex size-full items-center justify-center">
          <EmptyStateCompact assetKey="unknown" assetClassName="size-20" />
        </div>
      ) : (
        <NotificationPreview
          key={selected.notificationId}
          workspace={workspace}
          selected={selected}
          leave={() => landing.leave(selected.notificationId)}
        />
      )}
    </section>
  );
});

type PreviewProps = {
  workspace: { id: string; slug: string };
  selected: NotificationSelection;
  /** Leaves the notification, through the page's landing (R26). */
  leave: () => void;
};

/**
 * The selected notification's work item, scoped to the notification. It shows only once the caller's role in the work
 * item's project is known: a project he holds no role in (an archived one's, which the workspace's list leaves out) is
 * read first (useProjectRead), and the preview says it is loading meanwhile; each preview judges by its own read,
 * never by an answer SWR keeps from an earlier one (whole-branch m1). Once that read settles and he still has no role
 * (nerve refuses it: he has left the project, or nerve was not reached; or it gives the project but no role in it), the
 * selection is cleared and a toast says why: never a blank pane (re-93 I4). With his role, the work item is the
 * generation's (P5b R27): his permissions in the project and his login, a new one judging it anew (NotificationWorkItem).
 */
const NotificationPreview = observer(function NotificationPreview(props: PreviewProps) {
  const { workspace, selected } = props;
  const { t } = useTranslation();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { rootIssueStore: root } = useIssueDetail();
  const { loginId } = useSession();
  const { clearSelection } = useNotificationPageState();
  const refusalToast = useRefusalToast();
  const { notificationId, projectId } = selected;
  const readable = getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, projectId) !== undefined;
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
    if (!unreadable || said.current) return;
    said.current = true;
    if (read.error !== undefined) refusalToast(read.error);
    else
      setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(problemMessageKey("issue.not_found")) });
    clearSelection(notificationId);
  }, [unreadable, read.error, refusalToast, t, clearSelection, notificationId]);

  if (!readable) return <NotificationsLoader />;
  const identity = root.permissionReads.identity(workspace.id, [projectId]);
  return <NotificationWorkItem key={JSON.stringify([identity, loginId])} {...props} />;
});

/** How the preview shows the work item, judged once by its own read: a peek at it, or its intake's detail embedded. */
type Judgement = "peek" | "embed";

/**
 * The selected notification's work item (P5b spec 2.5, R16, R17, R27): until its preview has judged how to show it,
 * the judging (NotificationJudgement); then, one in the intake embeds its intake's detail (InboxContentRoot, the
 * embed's port: its leave and its close the page's landing's), never setting the peek; any other is peeked at
 * (embedded in the page, scoped to the notification; its close the landing's too). The judgement is held here alone:
 * the judging unmounts as it is made, its read leaving the work item's key, so that nothing judges again (the work
 * item accepted in the embed stays embedded, with its unsent input) and the peek's hook, the key's only one, reads it
 * as on every other page (as the browser reconnects, and by its retry). A new generation, NotificationPreview's key
 * for it, judges anew.
 */
const NotificationWorkItem = observer(function NotificationWorkItem(props: PreviewProps) {
  const { workspace, selected, leave } = props;
  const { notificationId, issueId, projectId } = selected;
  const { setPeekIssue } = useIssueDetail();
  const region = useRef<HTMLElement>(null);
  const [judged, setJudged] = useState<Judgement>();
  useEffect(() => {
    if (judged === "peek") setPeekIssue({ workspaceSlug: workspace.slug, projectId, issueId });
  }, [judged, workspace.slug, projectId, issueId, setPeekIssue]);

  if (judged === "embed")
    return (
      <InboxContentRoot
        workspaceSlug={workspace.slug}
        projectId={projectId}
        inboxIssueId={issueId}
        selection={{ kind: "embed", leave, region, close: leave }}
      />
    );
  if (judged === "peek")
    return (
      <IssuePeekOverview
        hostScope={JSON.stringify(["notification", workspace.slug, notificationId])}
        embedIssue
        embedRemoveCurrentNotification={leave}
      />
    );
  return <NotificationJudgement {...props} judge={setJudged} />;
});

/**
 * How the preview shows the selected notification's work item, judged by a read of its own (P5b R16, R27; O-1 乙):
 * the peek's read (usePeekIssueRead, the same key, so that a peek mounting right after joins it: one read, as SWR
 * dedupes, re-95 O2), its mount and reconnect reads off, made by mutate as it mounts, so that an answer SWR keeps from
 * an earlier read is never taken for its own; until it settles, it says it is loading. At its first success, the work
 * item as the stores hold it is judged (judge): one in the intake (pending, snoozed, declined, a duplicate) embedded,
 * any other (accepted, never submitted) peeked at. A read that fails shows its own failure (IssuePeekOverviewError),
 * its close the landing's, its retry reading again, judged once settled. An answer of another session or other
 * permissions (SessionChangedError) is no failure: the work item as held is judged, the generation that follows
 * judging it anew.
 */
const NotificationJudgement = observer(function NotificationJudgement(
  props: PreviewProps & { judge: (judgement: Judgement) => void }
) {
  const { workspace, selected, leave, judge } = props;
  const { issueId, projectId } = selected;
  const { rootIssueStore: root } = useIssueDetail();
  const { fetch } = useIssueOperations(workspace.slug, projectId, issueId);
  const read = usePeekIssueRead({ workspaceSlug: workspace.slug, projectId, issueId }, fetch, {
    revalidateOnMount: false,
    revalidateOnReconnect: false,
  });
  const { mutate } = read;
  // whether this judging's own read has answered
  const [answered, setAnswered] = useState(false);
  useEffect(() => {
    let judging = true;
    void mutate().finally(() => {
      if (judging) setAnswered(true);
    });
    return () => {
      judging = false;
    };
  }, [mutate]);
  // its read settled: answered, the key no longer validating
  const settled = answered && !read.isValidating;
  const failure = settled && !(read.error instanceof SessionChangedError) ? read.error : undefined;
  const judgement =
    settled && failure === undefined
      ? inIntake(root.issues.getIssueDetailById(issueId))
        ? "embed"
        : "peek"
      : undefined;
  useEffect(() => {
    if (judgement !== undefined) judge(judgement);
  }, [judgement, judge]);

  if (failure !== undefined)
    return (
      <IssuePeekOverviewError
        removeRoutePeekId={leave}
        error={failure}
        retry={() => mutate()}
        workspaceSlug={workspace.slug}
        projectId={projectId}
        issueId={issueId}
      />
    );
  return <NotificationsLoader />;
});
