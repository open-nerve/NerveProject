/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect } from "react";
import { observer } from "mobx-react";
// Nerve imports
import { useSessionSWR } from "@/lib/use-session-swr";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { IWorkItemPeekOverview } from "@nerve/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { useIssueOperations } from "../issue-detail/use-issue-operations";
import { IssueDetailHost } from "../detail-host";
import type { IIssueDetail } from "@/store/issue/issue-details/root.store";
import { IssueView } from "./view";

const peekOwners = new WeakMap<
  IIssueDetail,
  { scope: string; selection: IIssueDetail["peekIssue"]; active: boolean }
>();

export const IssuePeekOverview = observer(function IssuePeekOverview(
  props: IWorkItemPeekOverview & { hostScope: string }
) {
  const { hostScope, embedIssue = false, embedRemoveCurrentNotification } = props;
  // store hook
  const { allowPermissions } = useUserPermissions();

  const detail = useIssueDetail();
  const { peekIssue, rootIssueStore: root } = detail;
  useLayoutEffect(() => {
    if (!peekIssue) return;
    const previous = peekOwners.get(detail);
    if (previous && previous.scope !== hostScope && previous.selection === peekIssue) {
      peekOwners.delete(detail);
      detail.setPeekIssue(undefined);
      return;
    }
    const owner = { scope: hostScope, selection: peekIssue, active: true };
    peekOwners.set(detail, owner);
    return () => {
      owner.active = false;
      // StrictMode immediately reacquires. A real unmount releases after layout setups;
      // a different route is rejected synchronously by its next setup above.
      queueMicrotask(() => {
        if (peekOwners.get(detail) !== owner || owner.active) return;
        peekOwners.delete(detail);
        if (detail.peekIssue === peekIssue) detail.setPeekIssue(undefined);
      });
    };
  }, [detail, peekIssue, hostScope]);
  const issueOperations = useIssueOperations(
    peekIssue?.workspaceSlug ?? "",
    peekIssue?.projectId,
    peekIssue?.issueId ?? ""
  );

  const workspace = peekIssue ? root.rootStore.workspaceRoot.getWorkspaceBySlug(peekIssue.workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(workspace?.id, peekIssue ? [peekIssue.projectId] : []);
  const canRead =
    peekIssue &&
    root.active &&
    root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(
      peekIssue.workspaceSlug,
      peekIssue.projectId
    ) !== undefined;
  const { isLoading, error, mutate } = useSessionSWR(
    canRead ? ["peek-issue", identity, peekIssue.workspaceSlug, peekIssue.projectId, peekIssue.issueId] : null,
    (_identity, ws, project, id) => issueOperations.fetch(ws, project, id)
  );

  if (!peekIssue?.workspaceSlug || !peekIssue?.projectId || !peekIssue?.issueId) return <></>;

  // Check if issue is editable, based on user role
  const isEditable = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    peekIssue?.workspaceSlug,
    peekIssue?.projectId
  );

  return (
    <IssueDetailHost
      workspaceSlug={peekIssue.workspaceSlug}
      projectId={peekIssue.projectId}
      issueId={peekIssue.issueId}
    >
      <IssueView
        workspaceSlug={peekIssue.workspaceSlug}
        projectId={peekIssue.projectId}
        issueId={peekIssue.issueId}
        isLoading={isLoading}
        error={error instanceof SessionChangedError ? undefined : error}
        retry={() => mutate()}
        is_archived={!!peekIssue.isArchived}
        disabled={!isEditable}
        embedIssue={embedIssue}
        embedRemoveCurrentNotification={embedRemoveCurrentNotification}
        issueOperations={issueOperations}
      />
    </IssueDetailHost>
  );
});
