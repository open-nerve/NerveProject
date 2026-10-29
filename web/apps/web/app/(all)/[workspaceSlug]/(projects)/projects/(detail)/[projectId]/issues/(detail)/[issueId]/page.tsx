/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTheme } from "next-themes";
import { useEffect } from "react";
import { observer } from "mobx-react";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useNavigate } from "react-router";
import { useTranslation } from "@nerve/i18n";
// assets
import emptyIssueDark from "@/app/assets/empty-state/search/issues-dark.webp?url";
import emptyIssueLight from "@/app/assets/empty-state/search/issues-light.webp?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { LogoSpinner } from "@/components/common/logo-spinner";
import type { Route } from "./+types/page";

const IssueDetailsPage = observer(function IssueDetailsPage({ params }: Pick<Route.ComponentProps, "params">) {
  const { workspaceSlug, projectId, issueId } = params;
  const detail = useIssueDetail();
  const issueRoot = detail.rootIssueStore;
  const workspaceID = issueRoot.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  const identity = issueRoot.permissionReads.identity(workspaceID, [projectId]);
  const { data, error, isLoading, mutate } = useSessionSWR(
    ["issue-id-entry", workspaceSlug, projectId, issueId, identity],
    (slug, project, id) => detail.issue.fetchIssueCore(slug, project, id),
    { dedupingInterval: 0 }
  );
  const issue = data?.id === issueId ? detail.issue.getIssueDetailById(issueId) : undefined;
  const project = issue ? issueRoot.rootStore.projectRoot.project.getProjectById(issue.project_id) : undefined;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    if (issue && project)
      navigate(`/${workspaceSlug}/browse/${project.identifier}-${issue.sequence_id}`, { replace: true });
  }, [navigate, workspaceSlug, issue, project]);

  if (error || (!isLoading && !issue)) {
    return (
      <div className="flex size-full flex-col items-center justify-center">
        {error && (
          <DetailResourceError
            error={error}
            retry={() => mutate()}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
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
      </div>
    );
  }

  return (
    <div className="flex size-full items-center justify-center">
      <LogoSpinner />
    </div>
  );
});

export default IssueDetailsPage;
