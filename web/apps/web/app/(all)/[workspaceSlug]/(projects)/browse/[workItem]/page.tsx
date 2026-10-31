/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
// nerve imports
import { useTranslation } from "@nerve/i18n";

import { Loader } from "@nerve/ui";
// assets
import emptyIssueDark from "@/app/assets/empty-state/search/issues-dark.webp?url";
import emptyIssueLight from "@/app/assets/empty-state/search/issues-light.webp?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useNavigate, useOutletContext } from "react-router";
// layouts
import { ProjectAuthWrapper } from "@/layouts/auth-layout/project-wrapper";
// components
import { WorkItemDetailRoot } from "@/components/browse/workItem-detail";

import type { Route } from "./+types/page";
import type { IssueRouteRead } from "./layout";

export const IssueDetailsPage = observer(function IssueDetailsPage({ params }: Pick<Route.ComponentProps, "params">) {
  // router
  const navigate = useNavigate();
  const { workspaceSlug, workItem } = params;
  // hooks
  const { resolvedTheme } = useTheme();
  // store hooks
  const { t } = useTranslation();
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getProjectById } = useProject();
  const { toggleIssueDetailSidebar, issueDetailSidebarCollapsed } = useAppTheme();

  const { data, isLoading, error, mutate } = useOutletContext<IssueRouteRead>();

  // The identifier is parsed only by the server; the proven response chooses the project.
  const issueId = data?.id;
  const projectId = data?.project_id ?? "";
  const issue = getIssueById(issueId || "") || undefined;
  const project = (issue?.project_id && getProjectById(issue?.project_id)) || undefined;
  const issueLoader = !issue || isLoading;
  const pageTitle = project && issue ? `${project?.identifier}-${issue?.sequence_id} ${issue?.name}` : undefined;

  useEffect(() => {
    const handleToggleIssueDetailSidebar = () => {
      if (window && window.innerWidth < 768) {
        toggleIssueDetailSidebar(true);
      }
      if (window && issueDetailSidebarCollapsed && window.innerWidth >= 768) {
        toggleIssueDetailSidebar(false);
      }
    };
    window.addEventListener("resize", handleToggleIssueDetailSidebar);
    handleToggleIssueDetailSidebar();
    return () => window.removeEventListener("resize", handleToggleIssueDetailSidebar);
  }, [issueDetailSidebarCollapsed, toggleIssueDetailSidebar]);

  if ((error || data === undefined) && !isLoading) {
    return (
      <>
        {error && (
          <DetailResourceError
            error={error}
            retry={() => mutate()}
            workspaceSlug={workspaceSlug}
            projectId={projectId || undefined}
            issueId={workItem}
          />
        )}
        <EmptyState
          image={resolvedTheme === "dark" ? emptyIssueDark : emptyIssueLight}
          title={error ? t("common.error.label") : t("issue.empty_state.issue_detail.title")}
          description={error ? undefined : t("issue.empty_state.issue_detail.description")}
          primaryButton={{
            text: t("issue.empty_state.issue_detail.primary_button.text"),
            onClick: () => navigate(`/${workspaceSlug}/projects`),
          }}
        />
      </>
    );
  }

  if (issueLoader) {
    return (
      <Loader className="flex h-full gap-5 p-5">
        <div className="basis-2/3 space-y-2">
          <Loader.Item height="30px" width="40%" />
          <Loader.Item height="15px" width="60%" />
          <Loader.Item height="15px" width="60%" />
          <Loader.Item height="15px" width="40%" />
        </div>
        <div className="basis-1/3 space-y-3">
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
        </div>
      </Loader>
    );
  }

  return (
    <>
      <PageHead title={pageTitle} />
      {workspaceSlug && projectId && issueId && (
        <ProjectAuthWrapper projectId={projectId}>
          <WorkItemDetailRoot workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} issue={issue} />
        </ProjectAuthWrapper>
      )}
    </>
  );
});

export default IssueDetailsPage;
