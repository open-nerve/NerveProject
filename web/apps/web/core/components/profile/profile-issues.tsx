/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useUserPermissions } from "@/hooks/store/user";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType, type TProfileViews } from "@nerve/types";
// components
import { ProfileIssuesKanBanLayout } from "@/components/issues/issue-layouts/kanban/roots/profile-issues-root";
import { ProfileIssuesListLayout } from "@/components/issues/issue-layouts/list/roots/profile-issues-root";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";
import { useLocalIssueFilterUpdate } from "@/hooks/use-local-issue-filter-update";

type Props = {
  type: TProfileViews;
};

export const ProfileIssuesPage = observer(function ProfileIssuesPage(props: Props) {
  const { type } = props;
  const { workspaceSlug, userId } = useParams();
  // store hooks
  const {
    issuesFilter: { getIssueFilters, fetchFilters, updateFilterExpression },
  } = useIssues(EIssuesStoreType.PROFILE);
  const issueFilters = workspaceSlug && userId ? getIssueFilters(userId, workspaceSlug) : undefined;
  const follow = useLocalIssueFilterUpdate(workspaceSlug ?? "", undefined, userId ?? "", true);
  // derived values
  const activeLayout = issueFilters?.displayFilters?.layout || undefined;

  const permission = useUserPermissions();
  const role = workspaceSlug ? permission.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
  const allowed = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  const { t } = useTranslation();
  const { error } = useSessionSWR(allowed ? ["PROFILE_ISSUE_FILTERS", workspaceSlug, userId] : null, (slug, user) =>
    fetchFilters(slug, user)
  );
  if (error) return <div role="alert">{t(errorMessageKey(error))}</div>;
  if (!allowed) return null;

  if (!workspaceSlug || !userId) return null;

  return (
    <IssuesStoreContext.Provider value={EIssuesStoreType.PROFILE}>
      <WorkspaceLevelWorkItemFiltersHOC
        entityId={userId}
        entityType={EIssuesStoreType.PROFILE}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.profile_issues.filters}
        initialWorkItemFilters={issueFilters}
        updateFilters={(expression) => {
          void follow(() => updateFilterExpression(workspaceSlug, userId, expression));
        }}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: profileWorkItemsFilter }) => (
          <>
            <div className="flex h-full w-full flex-col">
              {profileWorkItemsFilter && <WorkItemFiltersRow filter={profileWorkItemsFilter} />}
              <div className="relative h-full w-full overflow-auto">
                {activeLayout === "list" ? (
                  <ProfileIssuesListLayout viewId={type} />
                ) : activeLayout === "kanban" ? (
                  <ProfileIssuesKanBanLayout viewId={type} />
                ) : null}
              </div>
            </div>
            {/* peek overview */}
            <IssuePeekOverview hostScope={JSON.stringify(["profile", workspaceSlug, userId, type])} />
          </>
        )}
      </WorkspaceLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
