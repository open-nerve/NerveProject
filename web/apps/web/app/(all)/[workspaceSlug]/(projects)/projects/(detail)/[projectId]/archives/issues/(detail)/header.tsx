/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import useSWR from "swr";
// ui
import { ArchiveOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { IssueDetailQuickActions } from "@/components/issues/issue-detail/issue-detail-quick-actions";
// constants
import { ISSUE_DETAILS } from "@nerve/constants";
// hooks
import { useProject } from "@/hooks/store/use-project";
// components
import { ProjectBreadcrumb } from "@/components/breadcrumbs/project";
// services
import { IssueService } from "@/services/issue";

const issueService = new IssueService();

type TProps = {
  workspaceSlug: string;
  projectId: string;
  archivedIssueId: string;
};

export const ProjectArchivedIssueDetailsHeader = observer(function ProjectArchivedIssueDetailsHeader(props: TProps) {
  // router
  const { workspaceSlug, projectId, archivedIssueId } = props;
  // store hooks
  const { currentProjectDetails, loader } = useProject();

  const { data: issueDetails } = useSWR(ISSUE_DETAILS(archivedIssueId), () =>
    issueService.retrieve(workspaceSlug, projectId, archivedIssueId)
  );

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
                label={
                  currentProjectDetails && issueDetails
                    ? `${currentProjectDetails.identifier}-${issueDetails.sequence_id}`
                    : ""
                }
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <IssueDetailQuickActions workspaceSlug={workspaceSlug} projectId={projectId} issueId={archivedIssueId} />
      </Header.RightItem>
    </Header>
  );
});
