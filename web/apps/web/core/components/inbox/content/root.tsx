/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
// nerve imports
import type { IssueDetail } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { ContentWrapper } from "@nerve/ui";
// components
import { IssueDetailHost } from "@/components/issues/detail-host";
import { useIssueOperations } from "@/components/issues/issue-detail/use-issue-operations";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { useIssueSaveStatus } from "@/components/issues/use-issue-save-status";
import { PlanningReadFailure } from "@/components/planning/read-failure";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { ApiError } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useSession } from "@/lib/auth/use-session";
import { problemMessageKey } from "@/lib/error-messages";
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { DeclineIssueModal } from "../modals/decline-issue-modal";
import { DeleteInboxIssueModal } from "../modals/delete-issue-modal";
import { SelectDuplicateInboxIssueModal } from "../modals/select-duplicate";
import { InboxIssueSnoozeModal } from "../modals/snooze-issue-modal";
import { IntakeDetail } from "./detail";
import { IntakeMobileHeader } from "./header/mobile";
import { IntakeHeader } from "./header/root";
import { useIntakeActions, type IntakeActions } from "./header/use-intake-actions";

type TInboxContentRoot = {
  workspaceSlug: string;
  projectId: string;
  inboxIssueId: string;
  isMobileSidebar: boolean;
  setIsMobileSidebar: (value: boolean) => void;
};

/** Refusals of a read that say the caller cannot read the work item. */
const UNREADABLE = new Set([401, 403, 404]);

/**
 * The detail region of a project's intake page (P5a spec 2.6, 2.7): a region named so and focusable, around the
 * selected work item, one selection's at a time: each selection of a work item, in this session and by the caller's
 * permissions now, is a detail of its own, with its own read, inputs, modals and writes in flight.
 */
export const InboxContentRoot = observer(function InboxContentRoot(props: TInboxContentRoot) {
  const { workspaceSlug, projectId, inboxIssueId } = props;
  const { t } = useTranslation();
  const { rootIssueStore: root } = useIssueDetail();
  const { loginId } = useSession();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [projectId], true);
  return (
    <section
      aria-label={t("inbox_issue.a11y.detail")}
      tabIndex={-1}
      className="relative flex h-full w-full flex-col overflow-hidden outline-none"
    >
      <IntakeSelection
        key={JSON.stringify([workspaceSlug, projectId, inboxIssueId, identity, loginId])}
        identity={identity}
        {...props}
      />
    </section>
  );
});

/**
 * The selected work item of a project's intake: read as the selection begins, by a read of its own (P4 m1), never by
 * an answer SWR keeps from an earlier one, and again only by a retry. Until that read settles it says it is loading.
 * Once it settles, nerve refusing it (the work item is gone, or the caller may not read it), or answering a work item
 * not in the intake, clears the selection (replace) and says why, once (re-94 m4); any other failure is shown with
 * its retry, the selection kept. A later read that finds the work item gone, a triage's read again (R34), clears it
 * too, unless the caller is deleting it, whose clearing is the delete's. The header and the detail are drawn inside
 * the detail host, which withdraws them while a triage reads the work item again; the modals, outside it, stay.
 */
const IntakeSelection = observer(function IntakeSelection(props: TInboxContentRoot & { identity: string }) {
  const { workspaceSlug, projectId, inboxIssueId: issueId, identity, isMobileSidebar, setIsMobileSidebar } = props;
  const { t } = useTranslation();
  const { rootIssueStore: root } = useIssueDetail();
  const refusalToast = useRefusalToast();
  const [, setSearchParams] = useSearchParams();
  const operations = useIssueOperations(workspaceSlug, projectId, issueId);
  const { status: isSubmitting, setTitleStatus, setDescriptionStatus } = useIssueSaveStatus();
  const read = useSessionSWR(
    ["intake-issue", identity, workspaceSlug, projectId, issueId],
    async (_identity, ws, project, id) => {
      await operations.fetch(ws, project, id);
      return root.issues.getIssueDetailById(id) ?? null;
    },
    { revalidateOnMount: false }
  );
  const { mutate } = read;
  // whether this selection's own read has answered: what SWR keeps of an earlier selection's is not its own
  const [answered, setAnswered] = useState(false);
  useEffect(() => {
    let selected = true;
    // mutate reads the work item anew, never joining an earlier read, and settles once SWR holds its answer
    void mutate().finally(() => {
      if (selected) setAnswered(true);
    });
    return () => {
      selected = false;
    };
  }, [mutate]);
  const clearSelection = useCallback(
    () =>
      setSearchParams(
        (params) => {
          const next = new URLSearchParams(params);
          next.delete("inboxIssueId");
          return next;
        },
        { replace: true }
      ),
    [setSearchParams]
  );
  // whether the caller is deleting the work item, whose absence is then the delete's to follow
  const leaving = useRef(false);
  const actions = useIntakeActions(workspaceSlug, projectId, issueId, { clearSelection, leaving });

  const settled = answered && !read.isValidating;
  const error = settled && !(read.error instanceof SessionChangedError) ? read.error : undefined;
  const refused = error instanceof ApiError && UNREADABLE.has(error.status);
  const outOfIntake = settled && error === undefined && read.data?.intake === null;
  // a later read's conclusion, as the detail host draws it: not there, nor being read again
  const later = root.verification.getError(issueId);
  const gone =
    settled &&
    error === undefined &&
    !leaving.current &&
    (root.verification.isMissing(issueId) ||
      (later instanceof ApiError && UNREADABLE.has(later.status)) ||
      (!root.issues.getIssueById(issueId) && !root.verification.isUncertain(issueId)));
  // whether the selection has said why it cannot show the work item, which it says once
  const said = useRef(false);
  useEffect(() => {
    if (said.current || !(refused || outOfIntake || gone)) return;
    said.current = true;
    const refusal = refused ? error : later instanceof ApiError ? later : undefined;
    if (refusal) refusalToast(refusal);
    else
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t(problemMessageKey(outOfIntake ? "issue.intake_not_found" : "issue.not_found")),
      });
    clearSelection();
  }, [refused, outOfIntake, gone, error, later, refusalToast, t, clearSelection]);

  return (
    <>
      {settled && error === undefined ? (
        <IssueDetailHost workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId}>
          <div className="z-[11] min-h-[52px] flex-shrink-0">
            <IntakeHeader projectId={projectId} issueId={issueId} actions={actions} isSubmitting={isSubmitting} />
            <IntakeMobileHeader
              issueId={issueId}
              actions={actions}
              isSubmitting={isSubmitting}
              isMobileSidebar={isMobileSidebar}
              setIsMobileSidebar={setIsMobileSidebar}
            />
          </div>
          <ContentWrapper className="divide-y-2 divide-subtle-1">
            <IntakeDetail
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issueId={issueId}
              isSubmitting={isSubmitting}
              setTitleStatus={setTitleStatus}
              setDescriptionStatus={setDescriptionStatus}
            />
          </ContentWrapper>
        </IssueDetailHost>
      ) : error !== undefined && !refused ? (
        <PlanningReadFailure error={error} onRetry={() => void mutate()} />
      ) : (
        <div role="status" className="p-4 text-13 text-secondary">
          {t("common.loading")}
        </div>
      )}
      <IntakeModals projectId={projectId} actions={actions} />
    </>
  );
});

/**
 * The modals of the selected work item's actions (P5a spec 2.8), outside the detail host, so that a triage's read
 * again does not withdraw them; each is given the work item as held when it opened.
 */
const IntakeModals = observer(function IntakeModals({
  projectId,
  actions,
}: {
  projectId: string;
  actions: IntakeActions;
}) {
  const { t } = useTranslation();
  const { getProjectById } = useProject();
  const { modal, close } = actions;
  if (!modal) return null;
  const detail: IssueDetail = modal.detail;
  if (modal.name === "accept")
    return (
      <CreateUpdateIssueModal
        data={detail}
        isOpen
        onClose={close}
        beforeFormSubmit={actions.accept}
        withDraftIssueWrapper={false}
        fetchIssueDetails={false}
        showActionItemsOnUpdate
        modalTitle={t("inbox_issue.actions.move", {
          value: `${getProjectById(projectId)?.identifier}-${detail.sequence_id}`,
        })}
        primaryButtonText={{ default: t("add_to_project"), loading: t("adding") }}
      />
    );
  if (modal.name === "decline")
    return (
      <DeclineIssueModal data={detail} isOpen onClose={close} onSubmit={() => actions.triage({ status: "declined" })} />
    );
  if (modal.name === "snooze")
    return (
      <InboxIssueSnoozeModal
        isOpen
        handleClose={close}
        value={detail.intake?.snoozed_till ? new Date(detail.intake.snoozed_till) : undefined}
        onConfirm={(date) => void actions.triage({ status: "snoozed", snoozed_till: date.toISOString() })}
      />
    );
  if (modal.name === "duplicate")
    return (
      <SelectDuplicateInboxIssueModal
        isOpen
        onClose={close}
        value={detail.intake?.duplicate_to_id}
        onSubmit={(duplicateToId) => void actions.triage({ status: "duplicate", duplicate_to_id: duplicateToId })}
      />
    );
  return <DeleteInboxIssueModal data={detail} isOpen onClose={close} onSubmit={actions.remove} />;
});
