/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "next/navigation";
// plane imports
import type { TIssue } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
// components
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useNavigate } from "react-router";
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

    await fetchSubWorkItems(workspaceSlug?.toString(), newIssue.project_id, workItemDetails.id);
  };

  const getCreateIssueModalData = () => {
    if (cycleId) return { cycle_id: cycleId.toString() };
    if (moduleId) return { module_ids: [moduleId.toString()] };
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
          onSubmit={() =>
            handleDeleteIssue(workspaceSlug.toString(), workItemDetails.project_id!, workItemId?.toString())
          }
        />
      )}
    </>
  );
});
