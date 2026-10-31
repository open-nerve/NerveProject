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
import { EFileAssetType } from "@nerve/types";
import type { TNameDescriptionLoader } from "@nerve/types";
// components
import { IssueDescriptionVersions } from "@/components/issues/issue-detail/description-versions";
import { DescriptionInput, type DescriptionInputHandle } from "@/components/editor/rich-text/description-input";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUser } from "@/hooks/store/user";
import useReloadConfirmations from "@/hooks/use-reload-confirmation";
// services
// local components
import type { TIssueOperations } from "../issue-detail";
import { IssueParentDetail } from "../issue-detail/parent";
import { IssueReaction } from "../issue-detail/reactions";
import { IssueTitleInput } from "../title-input";
// services init

type Props = {
  editorRef: React.RefObject<EditorRefApi | null>;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  disabled: boolean;
  isArchived: boolean;
  isSubmitting: TNameDescriptionLoader;
  setTitleStatus: (value: TNameDescriptionLoader) => void;
  setDescriptionStatus: (value: TNameDescriptionLoader) => void;
};

export const PeekOverviewIssueDetails = observer(function PeekOverviewIssueDetails(props: Props) {
  const {
    editorRef,
    workspaceSlug,
    projectId,
    issueId,
    issueOperations,
    disabled,
    isArchived,
    isSubmitting,
    setTitleStatus,
    setDescriptionStatus,
  } = props;
  const descriptionInputRef = useRef<DescriptionInputHandle>(null);
  // store hooks
  const { data: currentUser } = useUser();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();

  // reload confirmation
  const { setShowAlert } = useReloadConfirmations(isSubmitting === "submitting" || isSubmitting === "failed");

  useEffect(() => {
    setShowAlert(isSubmitting === "submitting" || isSubmitting === "failed");
  }, [isSubmitting, setShowAlert]);

  // derived values
  const issue = issueId ? getIssueDetailById(issueId) : undefined;

  if (!issue || !issue.project_id) return <></>;

  const issueDescription = issue.description_html === "" ? "<p></p>" : issue.description_html;

  return (
    <div className="space-y-2">
      {issue.parent_id && (
        <IssueParentDetail
          workspaceSlug={workspaceSlug}
          projectId={issue.project_id}
          issueId={issueId}
          issue={issue}
          issueOperations={issueOperations}
        />
      )}
      <div className="flex items-center justify-between gap-2">
        <IssueIdentifier issueId={issueId} projectId={issue.project_id} size="md" enableClickToCopyIdentifier />
      </div>
      <IssueTitleInput
        workspaceSlug={workspaceSlug}
        projectId={issue.project_id}
        issueId={issue.id}
        setIsSubmitting={setTitleStatus}
        issueOperations={issueOperations}
        disabled={disabled || isArchived}
        value={issue.name}
      />

      <DescriptionInput
        autosaveRef={descriptionInputRef}
        disabledExtensions={["image"]}
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
        containerClassName="-ml-3 border-none"
        disabled={disabled || isArchived}
        editorRef={editorRef}
        entityId={issue.id}
        fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
        initialValue={issueDescription}
        key={issue.id}
        onSubmit={async (value, isMigrationUpdate) => {
          if (!issue.id || !issue.project_id) return;
          await issueOperations.update(workspaceSlug, issue.project_id, issue.id, {
            description_html: value.description_html,
            ...(isMigrationUpdate ? { skip_activity: true } : {}),
          });
        }}
        setIsSubmitting={setDescriptionStatus}
        projectId={issue.project_id}
        workspaceSlug={workspaceSlug}
      />

      <div className="flex items-center justify-between gap-2">
        {currentUser && (
          <IssueReaction
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
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
          isRestoreDisabled={disabled || isArchived}
        />
      </div>
    </div>
  );
});
