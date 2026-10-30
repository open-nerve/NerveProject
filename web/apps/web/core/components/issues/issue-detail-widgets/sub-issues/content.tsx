/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import React, { useEffect, useState, useCallback } from "react";
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
import { useDetailScope } from "../use-detail-scope";

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
    create: TIssueCrudState;
    existing: TIssueCrudState;
    update: TIssueCrudState;
    delete: TIssueCrudState;
  }>({
    create: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
    existing: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
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
    fetchSubIssues,
    subIssues: { getError, retry },
  } = useIssueDetail();
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailModals();

  // helpers
  const subIssueOperations = useSubIssueOperations(workspaceSlug, projectId, parentIssueId);
  const captureScope = useDetailScope(workspaceSlug, projectId, parentIssueId);
  const [visible, setVisible] = useState(false);

  // handler
  const handleIssueCrudState = useCallback(
    (key: "create" | "existing" | "update" | "delete", _parentIssueId: string | null, issue: Issue | null = null) => {
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

  useEffect(() => {
    const current = captureScope();
    let active = true;
    void fetchSubIssues(workspaceSlug, projectId, parentIssueId).then(
      () => {
        if (active && current()) setVisible(true);
        return undefined;
      },
      () => {
        /* The resource retains its refusal for the read-only retry. */
      }
    );
    return () => {
      active = false;
    };
  }, [captureScope, fetchSubIssues, workspaceSlug, projectId, parentIssueId]);

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
        retry={async () => {
          const current = captureScope();
          await retry(workspaceSlug, projectId, parentIssueId);
          if (current() && !getError(parentIssueId)) setVisible(true);
        }}
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
