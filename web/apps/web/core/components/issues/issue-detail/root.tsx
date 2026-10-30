/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueUpdate } from "@nerve/api-client";
import { IssueDetailHost } from "../detail-host";
import { useIssueOperations } from "./use-issue-operations";
import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// assets
import emptyIssue from "@/app/assets/empty-state/issue.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useUserPermissions } from "@/hooks/store/user";
import { useNavigate } from "react-router";
// local components
import { IssuePeekOverview } from "../peek-overview";
import { IssueMainContent } from "./main-content";
import { IssueDetailsSidebar } from "./sidebar";
import { DetailResourceError } from "../issue-detail-widgets/resource-error";

export type TIssueOperations<TUpdate = IssueUpdate> = {
  fetch: (workspaceSlug: string, projectId: string, issueId: string, loader?: boolean) => Promise<void>;
  update: (workspaceSlug: string, projectId: string, issueId: string, data: TUpdate) => Promise<void>;
  /** Capture persistence authorization while the input's host is alive, for its final unmount flush. */
  prepareUpdate?: () => TIssueOperations<TUpdate>["update"];
  restore?: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
};

export type TIssueDetailRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  is_archived?: boolean;
};

export const IssueDetailRoot = observer(function IssueDetailRoot(props: TIssueDetailRoot) {
  const { t } = useTranslation();
  const { workspaceSlug, projectId, issueId, is_archived = false } = props;
  // router
  const navigate = useNavigate();
  // hooks
  const {
    issue: { getIssueById },
    rootIssueStore,
    subIssues,
    relation,
    link,
  } = useIssueDetail();
  const { allowPermissions } = useUserPermissions();
  const { issueDetailSidebarCollapsed } = useAppTheme();

  const issueOperations = useIssueOperations(workspaceSlug, projectId, issueId);

  // issue details
  const issue = getIssueById(issueId);
  const uncertain = rootIssueStore.verification.isUncertain(issueId);
  const readError =
    subIssues.getError(issueId) ??
    relation.getError(issueId) ??
    link.getError(issueId) ??
    rootIssueStore.verification.getError(issueId);
  const retryRead = async () => {
    const resources = [subIssues, relation, link].filter((resource) => resource.getError(issueId));
    if (resources.length)
      await Promise.all(resources.map((resource) => resource.retry(workspaceSlug, projectId, issueId)));
    else await rootIssueStore.issues.refreshAuthoritative(issueId);
  };
  // checking if issue is editable, based on user role
  const isEditable = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  return (
    <>
      <IssueDetailHost workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId}>
        {!issue && uncertain ? (
          readError ? (
            <DetailResourceError
              error={readError}
              retry={retryRead}
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              issueId={issueId}
            />
          ) : (
            <div role="status">{t("common.loading")}</div>
          )
        ) : !issue ? (
          <EmptyState
            image={emptyIssue}
            title={t("issue.empty_state.issue_detail.title")}
            description={t("issue.empty_state.issue_detail.description")}
            primaryButton={{
              text: t("issue.empty_state.issue_detail.primary_button.text"),
              onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/issues`),
            }}
          />
        ) : (
          <div className="flex h-full w-full overflow-hidden">
            <div className="h-full w-full space-y-6 overflow-y-auto px-9 py-5">
              <IssueMainContent
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                issueOperations={issueOperations}
                isEditable={isEditable}
                isArchived={is_archived}
              />
            </div>
            <div
              className="fixed right-0 z-[5] h-full w-full min-w-[300px] border-l border-subtle bg-surface-1 sm:w-1/2 md:relative md:w-1/4 lg:min-w-80 xl:min-w-96"
              style={issueDetailSidebarCollapsed ? { right: `-${window?.innerWidth || 0}px` } : {}}
            >
              <IssueDetailsSidebar
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                issueOperations={issueOperations}
                isEditable={!is_archived && isEditable}
              />
            </div>
          </div>
        )}
      </IssueDetailHost>
      {/* peek overview */}
      <IssuePeekOverview hostScope={JSON.stringify(["detail", workspaceSlug, projectId, issueId])} />
    </>
  );
});
