/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useMemo, useState } from "react";
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
import type { IIssueDetail, TPeekIssue } from "@/store/issue/issue-details/root.store";
import { IssueView } from "./view";

type PeekOwner = {
  host: object;
  key: number;
  selection: IIssueDetail["peekIssue"];
  active: boolean;
};
let nextPeek = 0;
const peekOwners = new WeakMap<IIssueDetail, PeekOwner>();

export const IssuePeekOverview = observer(function IssuePeekOverview(
  props: IWorkItemPeekOverview & { hostScope: string }
) {
  const { hostScope, embedIssue = false, embedRemoveCurrentNotification } = props;
  const detail = useIssueDetail();
  const { peekIssue, rootIssueStore: root } = detail;
  const canRead =
    peekIssue &&
    root.active &&
    root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(
      peekIssue.workspaceSlug,
      peekIssue.projectId
    ) !== undefined;
  const host = useMemo(() => ({ detail, hostScope }), [detail, hostScope]);
  const [owner, setOwner] = useState<PeekOwner>();
  useLayoutEffect(() => {
    if (!peekIssue) return;
    const previous = peekOwners.get(detail);
    // Another mounted presentation cannot take over or erase a live owner's selection.
    if (previous?.active && previous.host !== host) return;
    if (!canRead || (previous && previous.host !== host && previous.selection === peekIssue)) {
      peekOwners.delete(detail);
      if (detail.peekIssue === peekIssue) detail.setPeekIssue(undefined);
      return;
    }
    const next = { host, key: nextPeek++, selection: peekIssue, active: true };
    peekOwners.set(detail, next);
    setOwner(next);
    return () => {
      next.active = false;
      // StrictMode reacquires before this microtask; an older cleanup never releases a new owner.
      queueMicrotask(() => {
        if (peekOwners.get(detail) !== next || next.active) return;
        peekOwners.delete(detail);
        if (detail.peekIssue === peekIssue) detail.setPeekIssue(undefined);
      });
    };
  }, [detail, peekIssue, host, canRead]);
  const ownsSelection = owner?.active && owner.host === host && owner.selection === peekIssue;
  if (!ownsSelection || !canRead || !peekIssue?.workspaceSlug || !peekIssue?.projectId || !peekIssue?.issueId)
    return null;
  return (
    <PeekContent
      key={owner.key}
      selection={peekIssue}
      embedIssue={embedIssue}
      embedRemoveCurrentNotification={embedRemoveCurrentNotification}
    />
  );
});

const PeekContent = observer(function PeekContent({
  selection: peekIssue,
  embedIssue,
  embedRemoveCurrentNotification,
}: IWorkItemPeekOverview & { selection: TPeekIssue }) {
  const { allowPermissions } = useUserPermissions();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(peekIssue.workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [peekIssue.projectId]);
  const issueOperations = useIssueOperations(peekIssue.workspaceSlug, peekIssue.projectId, peekIssue.issueId);
  const { isLoading, error, mutate } = useSessionSWR(
    ["peek-issue", identity, peekIssue.workspaceSlug, peekIssue.projectId, peekIssue.issueId],
    (_identity, ws, project, id) => issueOperations.fetch(ws, project, id)
  );

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
