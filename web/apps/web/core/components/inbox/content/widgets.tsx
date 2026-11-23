/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { LinkOutline } from "@makeplane/propel/icons";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { IssueLinkCreateUpdateModal } from "@/components/issues/issue-detail/links/create-update-link-modal";
import { IssueAttachments } from "@/components/issues/issue-detail-widgets/attachments/root";
import { IssueLinksActionButton, LinksCollapsible } from "@/components/issues/issue-detail-widgets/links";
import { useLinkOperations } from "@/components/issues/issue-detail-widgets/links/helper";
import { IssueDetailWidgetButton } from "@/components/issues/issue-detail-widgets/widget-button";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  /** Whether the caller changes the work item by the intake's rule (intakeEditableBy), the detail's. */
  editable: boolean;
};

/**
 * The attachments and links of the selected work item of a project's intake (P5b spec 2.3, R10, R11), inside its
 * detail host. Its attachments take files by the intake's rule, which the detail gives (the content view's canWrite
 * is the second layer). Its links follow the contract's issue.update: an admin or a member adds, changes and deletes
 * them, whatever the work item's status and whether the intake is on, while neither the project nor the work item is
 * archived; a guest reads them. The work item detail's sub-work item and relation buttons are not here: nerve refuses
 * them for a work item in the intake.
 */
export const IntakeWidgets = observer(function IntakeWidgets(props: Props) {
  const { workspaceSlug, projectId, issueId, editable } = props;
  const { t } = useTranslation();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const { isIssueLinkModalOpen, toggleIssueLinkModal, setIssueLinkData, setLastWidgetAction } = useIssueDetailModals();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const linkOperations = useLinkOperations(workspaceSlug, projectId, issueId);

  const issue = getIssueDetailById(issueId);
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const linksWritable =
    !!project &&
    !project.archived_at &&
    !issue?.archived_at &&
    (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER);

  return (
    <div className="flex flex-col space-y-4">
      {linksWritable && (
        <div className="flex flex-wrap items-center gap-2">
          <IssueLinksActionButton
            customButton={
              <IssueDetailWidgetButton
                title={t("issue.add.link")}
                icon={<LinkOutline className="h-3.5 w-3.5 flex-shrink-0" />}
              />
            }
          />
        </div>
      )}
      <div className="flex flex-col">
        <IssueAttachments issueId={issueId} disabled={!editable} />
        {!!issue?.link_count && (
          <LinksCollapsible
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            disabled={!linksWritable}
          />
        )}
      </div>
      <IssueLinkCreateUpdateModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        isModalOpen={isIssueLinkModalOpen}
        handleOnClose={() => {
          toggleIssueLinkModal(false);
          setLastWidgetAction("links");
          setIssueLinkData(null);
        }}
        linkOperations={linkOperations}
      />
    </div>
  );
});
