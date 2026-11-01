/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTheme } from "next-themes";
import { observer } from "mobx-react";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
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
  const navigate = useNavigate();
  const capture = useDetailScope(workspaceSlug, projectId, issueId);
  const issueRoot = detail.rootIssueStore;
  const workspaceID = issueRoot.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id;
  const identity = issueRoot.permissionReads.identity(workspaceID, [projectId]);
  const { data, error, isLoading, isValidating, mutate } = useSessionSWR(
    ["issue-id-entry", workspaceSlug, projectId, issueId, identity],
    async (slug, project, id) => {
      const current = capture();
      const answer = await detail.issue.fetchIssueCore(slug, project, id);
      const directory = answer ? issueRoot.rootStore.projectRoot.project.getProjectById(answer.project_id) : undefined;
      // A cached SWR snapshot can precede revalidation's loading state. Only this
      // entry's completed read may replace its URL, while its UI scope still owns it.
      if (current() && answer && directory)
        navigate(`/${slug}/browse/${directory.identifier}-${answer.sequence_id}`, { replace: true });
      return answer;
    },
    { dedupingInterval: 0 }
  );
  const pending = isLoading || isValidating;
  const issue = !pending && !error && data?.id === issueId ? detail.issue.getIssueDetailById(issueId) : undefined;
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();

  if (error || (!pending && !issue)) {
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
