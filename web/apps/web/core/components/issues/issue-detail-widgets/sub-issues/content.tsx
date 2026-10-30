/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import React, { useState, useCallback } from "react";
import { observer } from "mobx-react";

import { EIssuesStoreType } from "@nerve/types";
// components
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
// local imports
import { CreateUpdateIssueModal } from "../../issue-modal/modal";
import { useSubIssueOperations } from "./helper";
import { SubIssuesListRoot } from "./issues-list/root";
import { DetailResourceError } from "../resource-error";

type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  disabled: boolean;
};

type TIssueCrudState = { toggle: boolean; parentIssueId: string | undefined; issue: Issue | undefined };

export function SubIssuesCollapsibleContent(props: Props) {
  return (
    <SubIssuesContent key={JSON.stringify([props.workspaceSlug, props.projectId, props.parentIssueId])} {...props} />
  );
}
const SubIssuesContent = observer(function SubIssuesContent(props: Props) {
  const { workspaceSlug, projectId, parentIssueId, disabled } = props;
  // state
  const [issueCrudState, setIssueCrudState] = useState<{
    update: TIssueCrudState;
    delete: TIssueCrudState;
  }>({
    update: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
    delete: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
  });
  // store hooks
  const {
    subIssues: { getError, retry, subIssuesByIssueId },
  } = useIssueDetail();
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailModals();

  // helpers
  const subIssueOperations = useSubIssueOperations(workspaceSlug, projectId, parentIssueId);
  // The detail loader owns the initial read. This getter also enforces the resource's current read proof.
  const visible = subIssuesByIssueId(parentIssueId) !== undefined;

  // handler
  const handleIssueCrudState = useCallback(
    (key: "update" | "delete", _parentIssueId: string | null, issue: Issue | null = null) => {
      setIssueCrudState({
        ...issueCrudState,
        [key]: {
          toggle: !issueCrudState[key].toggle,
          parentIssueId: _parentIssueId,
          issue,
        },
      });
    },
    [issueCrudState]
  );

  // render conditions
  const shouldRenderDeleteIssueModal =
    issueCrudState?.delete?.toggle &&
    issueCrudState?.delete?.issue &&
    issueCrudState.delete.parentIssueId &&
    issueCrudState.delete.issue.id;

  const shouldRenderUpdateIssueModal = issueCrudState?.update?.toggle && issueCrudState?.update?.issue;

  return (
    <>
      <DetailResourceError
        error={getError(parentIssueId)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={parentIssueId}
        retry={() => retry(workspaceSlug, projectId, parentIssueId)}
      />
      {visible && (
        <SubIssuesListRoot
          storeType={EIssuesStoreType.PROJECT}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          parentIssueId={parentIssueId}
          rootIssueId={parentIssueId}
          spacingLeft={6}
          canEdit={!disabled}
          handleIssueCrudState={handleIssueCrudState}
          subIssueOperations={subIssueOperations}
        />
      )}

      {shouldRenderDeleteIssueModal && (
        <DeleteIssueModal
          isOpen={issueCrudState?.delete?.toggle}
          handleClose={() => {
            handleIssueCrudState("delete", null, null);
            toggleDeleteIssueModal(null);
          }}
          data={issueCrudState?.delete?.issue as Issue}
          onSubmit={async () =>
            await subIssueOperations.deleteSubIssue(
              workspaceSlug,
              projectId,
              issueCrudState?.delete?.parentIssueId as string,
              issueCrudState?.delete?.issue?.id as string
            )
          }
          isSubIssue
        />
      )}

      {shouldRenderUpdateIssueModal && (
        <CreateUpdateIssueModal
          isOpen={issueCrudState?.update?.toggle}
          onClose={() => {
            handleIssueCrudState("update", null, null);
            toggleCreateIssueModal(false);
          }}
          data={issueCrudState?.update?.issue ?? undefined}
        />
      )}
    </>
  );
});
