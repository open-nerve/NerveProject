/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { TIssue } from "@nerve/types";
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

export const WorkItemLevelModals = observer(function WorkItemLevelModals(props: TWorkItemLevelModalsProps) {
  const { workItemIdentifier } = props;
  // router
  const { workspaceSlug, cycleId, moduleId } = useParams();
  const navigate = useNavigate();
  // store hooks
  const {
    issue: { getIssueById, getIssueIdByIdentifier },
  } = useIssueDetail();
  // derived values
  const workItemId = workItemIdentifier ? getIssueIdByIdentifier(workItemIdentifier) : undefined;
  const workItemDetails = workItemId ? getIssueById(workItemId) : undefined;

  const { removeIssue: removeWorkItem } = useIssuesActions(EIssuesStoreType.PROJECT);

  const {
    isCreateIssueModalOpen,
    toggleCreateIssueModal,
    isDeleteIssueModalOpen,
    toggleDeleteIssueModal,
    createWorkItemAllowedProjectIds,
  } = useCommandPalette();
  // derived values
  const { fetchSubIssues: fetchSubWorkItems } = useIssueDetail();

  // oxlint-disable-next-line no-shadow
  const handleDeleteIssue = async (workspaceSlug: string, projectId: string, issueId: string) => {
    try {
      await removeWorkItem(projectId, issueId);
      navigate(`/${workspaceSlug}/projects/${projectId}/issues`);
    } catch (error) {
      console.error("Failed to delete issue:", error);
    }
  };

  const handleCreateIssueSubmit = async (newIssue: TIssue) => {
    if (!workspaceSlug || !newIssue.project_id || !newIssue.id || newIssue.parent_id !== workItemDetails?.id) return;

    await fetchSubWorkItems(workspaceSlug, newIssue.project_id, workItemDetails.id);
  };

  const getCreateIssueModalData = () => {
    if (cycleId) return { cycle_id: cycleId };
    if (moduleId) return { module_ids: [moduleId] };
    return undefined;
  };

  return (
    <>
      <CreateUpdateIssueModal
        isOpen={isCreateIssueModalOpen}
        onClose={() => toggleCreateIssueModal(false)}
        data={getCreateIssueModalData()}
        onSubmit={handleCreateIssueSubmit}
        allowedProjectIds={createWorkItemAllowedProjectIds}
      />
      {workspaceSlug && workItemId && workItemDetails && workItemDetails.project_id && (
        <DeleteIssueModal
          handleClose={() => toggleDeleteIssueModal(false)}
          isOpen={isDeleteIssueModalOpen}
          data={workItemDetails}
          onSubmit={() => handleDeleteIssue(workspaceSlug, workItemDetails.project_id!, workItemId)}
        />
      )}
    </>
  );
});
