/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLayoutEffect, useMemo, type ReactNode } from "react";
import { observer } from "mobx-react";
import { IssueDetailModalContext, useIssueDetail } from "@/hooks/store/use-issue-detail";
import { ApiError } from "@/lib/api-error";
import { useTranslation } from "@nerve/i18n";
import { IssueDetailInputContext, useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import { IssueDetailInputState } from "./detail-input-state";
import { IssueDetailModals } from "@/store/issue/issue-details/modals";

let nextHost = 0;

export const IssueDetailHost = observer(function IssueDetailHost({
  workspaceSlug,
  projectId,
  issueId,
  children,
}: {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  children: ReactNode;
}) {
  const { rootIssueStore: root, setLastWidgetAction } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [projectId], true);
  const error = root.verification.getError(issueId);
  // A terminal canonical conclusion starts a new empty owner; temporary verification preserves this owner.
  const unavailable =
    root.verification.isMissing(issueId) ||
    (error instanceof ApiError && [401, 403, 404].includes(error.status)) ||
    (!root.issues.getIssueById(issueId) && !root.verification.isUncertain(issueId));
  const workspaceID = workspace?.id ?? "";
  const scope = JSON.stringify([workspaceSlug, workspaceID, projectId, issueId, identity, unavailable]);
  const owner = useMemo(
    () => ({
      root,
      scope,
      key: nextHost++,
      modals: new IssueDetailModals(setLastWidgetAction),
      inputs: new IssueDetailInputState(root, workspaceID, projectId, issueId),
    }),
    [root, scope, setLastWidgetAction, workspaceID, projectId, issueId]
  );
  const { modals, inputs } = owner;
  useLayoutEffect(() => {
    modals.activate();
    inputs.activate();
    return () => {
      modals.retire();
      inputs.deactivate();
    };
  }, [modals, inputs]);
  // Remount editor/dropdowns and child-local edit/delete payloads at the same boundary.
  return (
    <IssueDetailModalContext.Provider key={owner.key} value={modals}>
      <IssueDetailInputContext.Provider value={inputs}>{children}</IssueDetailInputContext.Provider>
    </IssueDetailModalContext.Provider>
  );
});

export const ConfirmedContentNotice = observer(function ConfirmedContentNotice() {
  const input = useIssueDetailInput();
  const { t } = useTranslation();
  const receipt = input?.receipts.at(-1);
  return receipt ? (
    <p role="status" className="px-4 py-2 text-13">
      {t(receipt.kind === "uploaded" ? "content_files.confirmed_upload" : "content_files.confirmed_delete")}
    </p>
  ) : null;
});
