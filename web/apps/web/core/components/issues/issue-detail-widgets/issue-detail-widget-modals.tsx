/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import React from "react";
import { observer } from "mobx-react";
// components
import {
  ExistingIssuesListModal,
  type SelectionSubmitResult,
} from "@/components/core/modals/existing-issues-list-modal";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
// local imports
import { IssueLinkCreateUpdateModal } from "../issue-detail/links/create-update-link-modal";
// helpers
import { CreateUpdateIssueModal } from "../issue-modal/modal";
import { useLinkOperations } from "./links/helper";
import { useSubIssueOperations } from "./sub-issues/helper";
import { SessionChangedError } from "@/lib/auth/token-manager";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
};

export const IssueDetailWidgetModals = observer(function IssueDetailWidgetModals(props: Props) {
  const { workspaceSlug, projectId, issueId } = props;
  // store hooks
  const { createRelation } = useIssueDetail();
  const {
    relationKey,
    issueCrudOperationState,
    isCreateIssueModalOpen,
    isIssueLinkModalOpen,
    isRelationModalOpen,
    isSubIssuesModalOpen,
    setIssueLinkData,
    toggleCreateIssueModal,
    toggleIssueLinkModal: toggleIssueLinkModalStore,
    toggleRelationModal,
    toggleSubIssuesModal,
    setRelationKey,
    setIssueCrudOperationState,
    setLastWidgetAction,
  } = useIssueDetailModals();

  // helper hooks
  const subIssueOperations = useSubIssueOperations(workspaceSlug, projectId, issueId);
  const handleLinkOperations = useLinkOperations(workspaceSlug, projectId, issueId);

  // handlers
  const handleIssueCrudState = (key: "create" | "existing", _parentIssueId: string | null) => {
    setIssueCrudOperationState({
      ...issueCrudOperationState,
      [key]: {
        toggle: !issueCrudOperationState[key].toggle,
        parentIssueId: _parentIssueId,
      },
    });
  };

  const handleExistingIssuesModalClose = () => {
    handleIssueCrudState("existing", null);
    setLastWidgetAction("sub-work-items");
    toggleSubIssuesModal(null);
  };

  const handleExistingIssuesModalOnSubmit = async (ids: readonly string[]): Promise<SelectionSubmitResult> => {
    const parent = issueCrudOperationState.existing.parentIssueId;
    if (!parent) throw new SessionChangedError();
    await subIssueOperations.addSubIssue(workspaceSlug, projectId, parent, [...ids]);
    return { confirmedIds: ids, failures: [] };
  };

  const handleCreateUpdateModalClose = () => {
    handleIssueCrudState("create", null);
    toggleCreateIssueModal(false);
    setLastWidgetAction("sub-work-items");
  };

  const handleIssueLinkModalOnClose = () => {
    toggleIssueLinkModalStore(false);
    setLastWidgetAction("links");
    setIssueLinkData(null);
  };

  const handleRelationOnClose = () => {
    setRelationKey(null);
    toggleRelationModal(null, null);
    setLastWidgetAction("relations");
  };

  const handleExistingIssueModalOnSubmit = async (ids: readonly string[]): Promise<SelectionSubmitResult> => {
    if (!relationKey) throw new SessionChangedError();
    await createRelation(workspaceSlug, projectId, issueId, relationKey, [...ids]);
    return { confirmedIds: ids, failures: [] };
  };

  // helpers
  const createUpdateModalData: Partial<IssueDetail> = {
    parent_id: issueCrudOperationState?.create?.parentIssueId,
    project_id: projectId,
  };

  const existingIssuesModalSearchParams = {
    for: "sub_issue" as const,
    issue_id: issueCrudOperationState.existing.parentIssueId ?? undefined,
  };

  // render conditions
  const shouldRenderExistingIssuesModal =
    issueCrudOperationState?.existing?.toggle &&
    issueCrudOperationState?.existing?.parentIssueId &&
    isSubIssuesModalOpen;

  const shouldRenderCreateUpdateModal =
    issueCrudOperationState?.create?.toggle && issueCrudOperationState?.create?.parentIssueId && isCreateIssueModalOpen;

  return (
    <>
      <IssueLinkCreateUpdateModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        isModalOpen={isIssueLinkModalOpen}
        handleOnClose={handleIssueLinkModalOnClose}
        linkOperations={handleLinkOperations}
      />

      {shouldRenderCreateUpdateModal && (
        <CreateUpdateIssueModal
          isOpen={issueCrudOperationState?.create?.toggle}
          data={createUpdateModalData}
          onClose={handleCreateUpdateModalClose}
          isProjectSelectionDisabled
        />
      )}

      {shouldRenderExistingIssuesModal && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isOpen={issueCrudOperationState?.existing?.toggle}
          handleClose={handleExistingIssuesModalClose}
          searchParams={existingIssuesModalSearchParams}
          handleOnSubmit={handleExistingIssuesModalOnSubmit}
        />
      )}

      <ExistingIssuesListModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isOpen={isRelationModalOpen?.issueId === issueId && isRelationModalOpen?.relationType === relationKey}
        handleClose={handleRelationOnClose}
        searchParams={{ for: "relation", issue_id: issueId }}
        handleOnSubmit={handleExistingIssueModalOnSubmit}
        workspaceLevelToggle
      />
    </>
  );
});
