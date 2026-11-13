/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useLayoutEffect, useMemo } from "react";
import type { ICommandPaletteStore } from "@/store/command-palette.store";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { observer } from "mobx-react";
// nerve imports

import { EIssuesStoreType } from "@nerve/types";
// components
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useParams, useNavigate } from "react-router";
import { useIssuesActions } from "@/hooks/use-issues-actions";

export type TWorkItemLevelModalsProps = {
  workItemIdentifier: string | undefined;
};

const commandOwners = new WeakMap<ICommandPaletteStore, object>();

export const WorkItemLevelModals = observer(function WorkItemLevelModals(props: TWorkItemLevelModalsProps) {
  const { workItemIdentifier } = props;
  // router
  const { workspaceSlug, projectId: routerProjectId } = useParams();
  const navigate = useNavigate();
  // store hooks
  const {
    issue: { getIssueById, getIssueIdByIdentifier },
    rootIssueStore: root,
  } = useIssueDetail();
  // derived values
  const workItemId = workItemIdentifier ? getIssueIdByIdentifier(workItemIdentifier) : undefined;
  const workItemDetails = workItemId ? getIssueById(workItemId) : undefined;

  const { removeIssue: removeWorkItem } = useIssuesActions(EIssuesStoreType.PROJECT);

  const commands = useCommandPalette();
  const {
    isCreateIssueModalOpen,
    toggleCreateIssueModal,
    isDeleteIssueModalOpen,
    toggleDeleteIssueModal,
    createIssueStoreType,
    createWorkItemAllowedProjectIds,
  } = commands;
  // derived values
  const { fetchSubIssues: fetchSubWorkItems } = useIssueDetail();

  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug ?? "");
  const identity = root.permissionReads.identity(workspace?.id, routerProjectId ? [routerProjectId] : [], true);
  const scope = JSON.stringify([workspaceSlug, routerProjectId ?? workItemIdentifier, workItemIdentifier, identity]);
  const owner = useMemo(() => ({ active: false, scope, commands, root }), [commands, root, scope]);
  // The numbered route survives a confirmed deletion even though its canonical issue is gone.
  // Scope this host by its route and root-owned permission watermark, not by the entity's presence.
  const capture = useDetailScope(workspaceSlug ?? "", routerProjectId, workItemIdentifier ?? "commands", true);
  useLayoutEffect(() => {
    if (commandOwners.has(commands)) {
      toggleCreateIssueModal(false);
      toggleDeleteIssueModal(false);
    }
    owner.active = true;
    commandOwners.set(commands, owner);
    return () => {
      owner.active = false;
      if (commandOwners.get(commands) !== owner) return;
      commandOwners.delete(commands);
      toggleCreateIssueModal(false);
      toggleDeleteIssueModal(false);
    };
  }, [commands, owner, toggleCreateIssueModal, toggleDeleteIssueModal]);
  const owns = () => owner.active && commandOwners.get(commands) === owner;
  const handleDeleteIssue = async (slug: string, project: string, id: string) => {
    const current = capture();
    await removeWorkItem(project, id);
    if (owns() && current()) navigate(`/${slug}/projects/${project}/issues`);
  };

  const handleCreateIssueSubmit = async (newIssue: Issue) => {
    if (
      !owns() ||
      !capture()() ||
      !workspaceSlug ||
      !newIssue.project_id ||
      !newIssue.id ||
      newIssue.parent_id !== workItemDetails?.id
    )
      return;

    await fetchSubWorkItems(workspaceSlug, newIssue.project_id, workItemDetails.id);
  };

  return (
    <>
      <CreateUpdateIssueModal
        key={`create:${scope}`}
        isOpen={isCreateIssueModalOpen}
        onClose={() => {
          if (owns()) toggleCreateIssueModal(false);
        }}
        onSubmit={handleCreateIssueSubmit}
        allowedProjectIds={createWorkItemAllowedProjectIds}
        // the cycle page's own entries open it for a work item of the cycle, of its project (P5b spec 2.5); any other
        // opening, the app's own included, lets the project be chosen
        isProjectSelectionDisabled={createIssueStoreType === EIssuesStoreType.CYCLE}
      />
      {workspaceSlug && workItemId && workItemDetails && workItemDetails.project_id && (
        <DeleteIssueModal
          key={`delete:${scope}`}
          handleClose={() => {
            if (owns()) toggleDeleteIssueModal(false);
          }}
          isOpen={isDeleteIssueModalOpen}
          data={workItemDetails}
          onSubmit={() => handleDeleteIssue(workspaceSlug, workItemDetails.project_id!, workItemId)}
        />
      )}
    </>
  );
});
