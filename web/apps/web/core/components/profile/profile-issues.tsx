/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useSessionSWR } from "@/lib/use-session-swr";
import { useUserPermissions } from "@/hooks/store/user";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
// nerve imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
// components
import { ProfileIssuesKanBanLayout } from "@/components/issues/issue-layouts/kanban/roots/profile-issues-root";
import { ProfileIssuesListLayout } from "@/components/issues/issue-layouts/list/roots/profile-issues-root";
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { IssuesStoreContext } from "@/hooks/use-issue-layout-store";

type Props = {
  type: "assigned" | "subscribed" | "created";
};

export const ProfileIssuesPage = observer(function ProfileIssuesPage(props: Props) {
  const { type } = props;
  const { workspaceSlug, userId } = useParams();
  // store hooks
  const {
    issues: { setViewId },
    issuesFilter: { issueFilters, fetchFilters, updateFilterExpression },
  } = useIssues(EIssuesStoreType.PROFILE);
  // derived values
  const activeLayout = issueFilters?.displayFilters?.layout || undefined;

  useEffect(() => {
    if (setViewId) setViewId(type);
  }, [type, setViewId]);

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
          void updateFilterExpression(workspaceSlug, userId, expression).catch(() => {});
        }}
        workspaceSlug={workspaceSlug}
      >
        {({ filter: profileWorkItemsFilter }) => (
          <>
            <div className="flex h-full w-full flex-col">
              {profileWorkItemsFilter && <WorkItemFiltersRow filter={profileWorkItemsFilter} />}
              <div className="relative h-full w-full overflow-auto">
                {activeLayout === "list" ? (
                  <ProfileIssuesListLayout />
                ) : activeLayout === "kanban" ? (
                  <ProfileIssuesKanBanLayout />
                ) : null}
              </div>
            </div>
            {/* peek overview */}
            <IssuePeekOverview />
          </>
        )}
      </WorkspaceLevelWorkItemFiltersHOC>
    </IssuesStoreContext.Provider>
  );
});
