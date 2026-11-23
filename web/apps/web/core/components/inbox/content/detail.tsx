/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import type { EditorRefApi } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
// components
import { DescriptionInput, type DescriptionInputHandle } from "@/components/editor/rich-text/description-input";
import { IssueDescriptionVersions } from "@/components/issues/issue-detail/description-versions";
import { IssueActivity } from "@/components/issues/issue-detail/issue-activity";
import { IssueReaction } from "@/components/issues/issue-detail/reactions";
import { useIssueOperations } from "@/components/issues/issue-detail/use-issue-operations";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
import { IssueTitleInput } from "@/components/issues/title-input";
// hooks
import { useIssueImages } from "@/hooks/editor/use-issue-images";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import useReloadConfirmations from "@/hooks/use-reload-confirmation";
// store
import { intakeEditableBy } from "@/store/issue/intake/status";
// local imports
import { InboxIssueContentProperties } from "./issue-properties";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isSubmitting: TNameDescriptionLoader;
  setTitleStatus: (value: TNameDescriptionLoader) => void;
  setDescriptionStatus: (value: TNameDescriptionLoader) => void;
};

/**
 * The selected work item of a project's intake, drawn inside the detail host (P5a spec 2.7): its name and description,
 * its reactions and description versions (shown to admins and members alone, IssueDescriptionVersions), its
 * properties, and its activity and comments. Each is the work item detail's own control, writing through the work
 * item's queue (useIssueOperations), which sends a work item in the intake to its intake (R3). Who changes it is the
 * intake's rule (intakeEditableBy), which its description's images (useIssueImages, the issue host) take uploads by
 * too (P5b spec 2.2). A guest who submitted it changes its name and description alone. While the work item is not
 * held, as a triage reads it again (R34), it says that it is loading, or why the read again failed with its retry.
 */
export const IntakeDetail = observer(function IntakeDetail(props: Props) {
  const { workspaceSlug, projectId, issueId, isSubmitting, setTitleStatus, setDescriptionStatus } = props;
  const { t } = useTranslation();
  const editorRef = useRef<EditorRefApi>(null);
  const descriptionInputRef = useRef<DescriptionInputHandle>(null);
  const { rootIssueStore: root, issue: detail } = useIssueDetail();
  const { data: currentUser } = useUser();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const issueOperations = useIssueOperations(workspaceSlug, projectId, issueId);
  const { images } = useIssueImages(workspaceSlug, projectId, issueId);
  const { setShowAlert } = useReloadConfirmations(isSubmitting === "submitting" || isSubmitting === "failed");
  useEffect(() => {
    setShowAlert(isSubmitting === "submitting" || isSubmitting === "failed");
  }, [isSubmitting, setShowAlert]);

  const issue = detail.getIssueDetailById(issueId);
  if (!issue) {
    if (!root.verification.isUncertain(issueId) || root.verification.isMissing(issueId)) return null;
    const error = root.verification.getError(issueId);
    return error ? (
      <DetailResourceError
        error={error}
        retry={() => root.issues.refreshAuthoritative(issueId)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
      />
    ) : (
      <div role="status">{t("common.loading")}</div>
    );
  }
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const editable = intakeEditableBy(issue, project, role, currentUser?.id);
  const archived = !!project?.archived_at;

  return (
    <>
      <div className="space-y-4 pb-4">
        <IssueTitleInput
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          setIsSubmitting={setTitleStatus}
          issueOperations={issueOperations}
          disabled={!editable}
          value={issue.name}
        />
        <DescriptionInput
          autosaveRef={descriptionInputRef}
          images={images}
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
          disabled={!editable}
          editorRef={editorRef}
          entityId={issueId}
          initialValue={issue.description_html}
          key={issueId}
          onSubmit={(value, isMigrationUpdate) =>
            issueOperations.update(workspaceSlug, projectId, issueId, {
              description_html: value.description_html,
              ...(isMigrationUpdate ? { skip_activity: true } : {}),
            })
          }
          projectId={projectId}
          setIsSubmitting={setDescriptionStatus}
          workspaceSlug={workspaceSlug}
        />
        <div className="flex items-center justify-between gap-2">
          {currentUser && (
            <IssueReaction
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issueId={issueId}
              currentUser={currentUser}
              disabled={archived}
            />
          )}
          <IssueDescriptionVersions
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            restoreDescription={(confirm) => descriptionInputRef.current!.restore(confirm)}
            isRestoreDisabled={!editable}
          />
        </div>
      </div>
      <div className="py-4">
        <InboxIssueContentProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          isEditable={editable && role !== EUserPermissions.GUEST}
        />
      </div>
      <div className="pt-4">
        <IssueActivity
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={archived}
          isIntakeIssue
        />
      </div>
    </>
  );
});
