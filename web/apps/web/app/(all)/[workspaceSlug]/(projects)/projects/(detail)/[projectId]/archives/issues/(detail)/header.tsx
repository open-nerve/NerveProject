/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
// ui
import { ArchiveOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { IssueDetailQuickActions } from "@/components/issues/issue-detail/issue-detail-quick-actions";
// hooks
import { useProject } from "@/hooks/store/use-project";
// components
import { ProjectBreadcrumb } from "@/components/breadcrumbs/project";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  archivedIssueId: string;
};

export const ProjectArchivedIssueDetailsHeader = observer(function ProjectArchivedIssueDetailsHeader(props: TProps) {
  // router
  const { workspaceSlug, projectId, archivedIssueId } = props;
  // store hooks
  const { getProjectById, loader } = useProject();
  const detail = useIssueDetail();
  const root = detail.rootIssueStore;
  const workspaceID = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  const identity = root.permissionReads.identity(workspaceID, [projectId]);
  const readable =
    root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined;
  const { error, mutate } = useSessionSWR(
    readable ? ["archive-issue-header", workspaceSlug, projectId, archivedIssueId, identity] : null,
    (slug, project, id) => detail.issue.fetchIssueCore(slug, project, id),
    { dedupingInterval: 0 }
  );
  const issueDetails = readable ? detail.issue.getIssueDetailById(archivedIssueId) : undefined;
  const project = getProjectById(projectId);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={loader === "init-loader"}>
          <ProjectBreadcrumb workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                href={`/${workspaceSlug}/projects/${projectId}/archives/issues`}
                label="Archives"
                icon={<ArchiveOutline className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                href={`/${workspaceSlug}/projects/${projectId}/archives/issues`}
                label="Work items"
                icon={<WorkItemsOutline className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={project && issueDetails ? `${project.identifier}-${issueDetails.sequence_id}` : ""}
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <DetailResourceError
          error={error}
          retry={() => mutate()}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={archivedIssueId}
        />
        <IssueDetailQuickActions workspaceSlug={workspaceSlug} projectId={projectId} issueId={archivedIssueId} />
      </Header.RightItem>
    </Header>
  );
});
