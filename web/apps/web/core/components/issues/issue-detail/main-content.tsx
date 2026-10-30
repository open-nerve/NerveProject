// oxlint-disable no-shadow
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import { useIssueSaveStatus } from "@/components/issues/use-issue-save-status";
import { EFileAssetType } from "@nerve/types";
// components
import { IssueDescriptionVersions } from "@/components/issues/issue-detail/description-versions";
import { DescriptionInput, type DescriptionInputHandle } from "@/components/editor/rich-text/description-input";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUser } from "@/hooks/store/user";
import useReloadConfirmations from "@/hooks/use-reload-confirmation";
import useSize from "@/hooks/use-window-size";
// services
// local imports
import { IssueDetailWidgets } from "../issue-detail-widgets";
import { NameDescriptionUpdateStatus } from "../issue-update-status";
import { PeekOverviewProperties } from "../peek-overview/properties";
import { IssueTitleInput } from "../title-input";
import { IssueActivity } from "./issue-activity";
import { IssueParentDetail } from "./parent";
import { IssueReaction } from "./reactions";
import type { TIssueOperations } from "./root";
// services init

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  isEditable: boolean;
  isArchived: boolean;
};

export const IssueMainContent = observer(function IssueMainContent(props: Props) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable, isArchived } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const descriptionInputRef = useRef<DescriptionInputHandle>(null);
  // states
  const { status: isSubmitting, setTitleStatus, setDescriptionStatus } = useIssueSaveStatus();
  // hooks
  const windowSize = useSize();
  const { data: currentUser } = useUser();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const { setShowAlert } = useReloadConfirmations(isSubmitting === "submitting" || isSubmitting === "failed");
  // derived values
  const issue = issueId ? getIssueDetailById(issueId) : undefined;

  useEffect(() => {
    setShowAlert(isSubmitting === "submitting" || isSubmitting === "failed");
  }, [isSubmitting, setShowAlert]);

  if (!issue || !issue.project_id) return <></>;

  return (
    <>
      <div className="space-y-4 rounded-lg">
        {issue.parent_id && (
          <IssueParentDetail
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            issue={issue}
            issueOperations={issueOperations}
          />
        )}

        <div className="mb-2.5 flex items-center justify-between gap-4">
          <IssueIdentifier issueId={issueId} projectId={issue.project_id} size="md" enableClickToCopyIdentifier />
          <div className="flex items-center gap-3">
            <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
          </div>
        </div>

        <IssueTitleInput
          workspaceSlug={workspaceSlug}
          projectId={issue.project_id}
          issueId={issue.id}
          setIsSubmitting={setTitleStatus}
          issueOperations={issueOperations}
          disabled={isArchived || !isEditable}
          value={issue.name}
        />

        <DescriptionInput
          autosaveRef={descriptionInputRef}
          prepareSubmit={
            issueOperations.prepareUpdate
              ? () => {
                  const update = issueOperations.prepareUpdate!();
                  return (value, isMigrationUpdate) =>
                    update(workspaceSlug, projectId, issueId, {
                      description_html: value.description_html,
                      ...(isMigrationUpdate ? { skip_activity: true } : {}),
                    });
                }
              : undefined
          }
          issueSequenceId={issue.sequence_id}
          containerClassName="-ml-6 border-none p-0! pl-6!"
          disabled={isArchived || !isEditable}
          editorRef={editorRef}
          entityId={issue.id}
          fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
          initialValue={issue.description_html}
          key={issue.id}
          onSubmit={async (value, isMigrationUpdate) => {
            if (!issue.id || !issue.project_id) return;
            await issueOperations.update(workspaceSlug, issue.project_id, issue.id, {
              description_html: value.description_html,
              ...(isMigrationUpdate ? { skip_activity: true } : {}),
            });
          }}
          projectId={issue.project_id}
          setIsSubmitting={setDescriptionStatus}
          workspaceSlug={workspaceSlug}
        />

        <div className="flex items-center justify-between gap-2">
          {currentUser && (
            <IssueReaction
              className="flex-shrink-0"
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issueId={issueId}
              currentUser={currentUser}
              disabled={isArchived}
            />
          )}
          <IssueDescriptionVersions
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
            issueId={issueId}
            restoreDescription={(confirm) => descriptionInputRef.current!.restore(confirm)}
            isRestoreDisabled={!isEditable || isArchived}
          />
        </div>
      </div>

      <IssueDetailWidgets
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        disabled={!isEditable || isArchived}
      />

      {windowSize[0] < 768 && (
        <PeekOverviewProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          disabled={!isEditable || isArchived}
        />
      )}

      <IssueActivity workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} disabled={isArchived} />
    </>
  );
});
