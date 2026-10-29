/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLayoutEffect, useMemo, type ReactNode } from "react";
import { observer } from "mobx-react";
import { IssueDetailModalContext, useIssueDetail } from "@/hooks/store/use-issue-detail";
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
  const scope = JSON.stringify([workspaceSlug, projectId, issueId, identity]);
  const owner = useMemo(
    () => ({ root, scope, key: nextHost++, modals: new IssueDetailModals(setLastWidgetAction) }),
    [root, scope, setLastWidgetAction]
  );
  const { modals } = owner;
  useLayoutEffect(() => {
    modals.activate();
    return modals.retire;
  }, [modals]);
  // Remount editor/dropdowns and child-local edit/delete payloads at the same boundary.
  return (
    <IssueDetailModalContext.Provider key={owner.key} value={modals}>
      {children}
    </IssueDetailModalContext.Provider>
  );
});
