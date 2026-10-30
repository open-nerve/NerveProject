/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { Collapsible } from "@makeplane/propel/components/collapsible";
import { useTranslation } from "@nerve/i18n";
import type { TIssueRelationTypes } from "@nerve/types";
import { cn } from "@nerve/utils";
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import { ISSUE_RELATION_OPTIONS } from "@/components/relations";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
// helper
import { DeleteIssueModal } from "../../delete-issue-modal";
import { RelationIssueList } from "../../relations/issue-list";
import { useRelationOperations } from "./helper";
import { DetailResourceError } from "../resource-error";

type Props = {
  workspaceSlug: string;
  issueId: string;
  disabled: boolean;
};

type TIssueCrudState = { toggle: boolean; issueId: string | undefined; issue: Issue | undefined };

export type TRelationObject = {
  key: TIssueRelationTypes;
  i18n_label: string;
  className: string;
  icon: (size: number) => React.ReactElement;
  placeholder: string;
};

export const RelationsCollapsibleContent = observer(function RelationsCollapsibleContent(props: Props) {
  const { workspaceSlug, issueId, disabled = false } = props;
  // nerve hooks
  const { t } = useTranslation();
  // state
  const [issueCrudState, setIssueCrudState] = useState<{
    update: TIssueCrudState;
    delete: TIssueCrudState;
  }>({
    update: {
      toggle: false,
      issueId: undefined,
      issue: undefined,
    },
    delete: {
      toggle: false,
      issueId: undefined,
      issue: undefined,
    },
  });

  // store hooks
  const {
    relation: { getRelationsByIssueId, getError, retry, getProjectId },
  } = useIssueDetail();
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailModals();

  // helper
  const issueOperations = useRelationOperations(workspaceSlug, issueId);

  // derived values
  const relations = getRelationsByIssueId(issueId);

  const handleIssueCrudState = (key: "update" | "delete", _issueId: string | null, issue: Issue | null = null) => {
    setIssueCrudState((prevState) => ({
      ...prevState,
      [key]: {
        toggle: !prevState[key].toggle,
        issueId: _issueId,
        issue: issue,
      },
    }));
  };

  // if relations are not available, return null
  const projectId = getProjectId(issueId);
  const error = (
    <DetailResourceError
      error={getError(issueId)}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
      retry={() => (projectId ? retry(workspaceSlug, projectId, issueId) : Promise.resolve())}
    />
  );
  if (!relations) return error;

  // map relations to array
  const relationsArray = (Object.keys(relations) as TIssueRelationTypes[])
    .filter((relationKey) => !!ISSUE_RELATION_OPTIONS[relationKey])
    .map((relationKey) => {
      const issueIds = relations[relationKey];
      const issueRelationOption = ISSUE_RELATION_OPTIONS[relationKey];
      return {
        relationKey: relationKey,
        issueIds: issueIds,
        icon: issueRelationOption?.icon,
        label: issueRelationOption?.i18n_label ? t(issueRelationOption?.i18n_label) : "",
        className: issueRelationOption?.className,
      };
    });

  // filter out relations with no issues
  const filteredRelationsArray = relationsArray.filter((relation) => relation.issueIds.length > 0);

  const shouldRenderIssueDeleteModal =
    issueCrudState?.delete?.toggle &&
    issueCrudState?.delete?.issue &&
    issueCrudState.delete.issueId &&
    issueCrudState.delete.issue.id;

  const shouldRenderIssueUpdateModal = issueCrudState?.update?.toggle && issueCrudState?.update?.issue;

  return (
    <>
      {error}
      <div className="gap- flex flex-col">
        {filteredRelationsArray.map((relation) => (
          <div key={relation.relationKey}>
            <Collapsible
              defaultOpen
              icon={
                relation.icon ? (
                  <span className={cn("flex items-center justify-center rounded p-0.5", relation.className)}>
                    {relation.icon(14)}
                  </span>
                ) : undefined
              }
              trigger={<span className="text-13 leading-5 font-medium">{relation.label}</span>}
            >
              {/* Cancels this nested Collapsible's own panel inset so rows realign under
                  the outer Relations collapsible's single inset instead of stacking two. */}
              <div className="-mx-3">
                <RelationIssueList
                  workspaceSlug={workspaceSlug}
                  issueId={issueId}
                  relationKey={relation.relationKey}
                  issueIds={relation.issueIds}
                  disabled={disabled}
                  handleIssueCrudState={handleIssueCrudState}
                />
              </div>
            </Collapsible>
          </div>
        ))}
      </div>

      {shouldRenderIssueDeleteModal && (
        <DeleteIssueModal
          isOpen={issueCrudState?.delete?.toggle}
          handleClose={() => {
            handleIssueCrudState("delete", null, null);
            toggleDeleteIssueModal(null);
          }}
          data={issueCrudState?.delete?.issue as Issue}
          onSubmit={async () => {
            if (
              issueCrudState.delete.issue &&
              issueCrudState.delete.issue.id &&
              issueCrudState.delete.issue.project_id
            ) {
              await issueOperations.remove(
                workspaceSlug,
                issueCrudState.delete.issue?.project_id,
                issueCrudState?.delete?.issue?.id
              );
            }
          }}
        />
      )}

      {shouldRenderIssueUpdateModal && (
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
