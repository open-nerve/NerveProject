/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import { useCallback } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { EIssueFilterType, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { EIssuesStoreType, IIssueDisplayFilterOptions } from "@nerve/types";
import { EIssueLayoutTypes } from "@nerve/types";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListRefresh, IssueListRetry } from "@/hooks/use-issue-list-refresh";
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
// local imports
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { IQuickActionProps, TRenderQuickActions } from "../list/list-view-types";
import { SpreadsheetView } from "./spreadsheet-view";

type SpreadsheetStoreType = EIssuesStoreType.PROJECT;

interface IBaseSpreadsheetRoot {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  viewId?: string | undefined;
}

export const BaseSpreadsheetRoot = observer(function BaseSpreadsheetRoot(props: IBaseSpreadsheetRoot) {
  const { QuickActions, canEditPropertiesBasedOnProject, viewId } = props;
  // router
  const { workspaceSlug, projectId } = useParams();
  // store hooks
  const storeType = useIssueStoreType() as SpreadsheetStoreType;
  const { allowPermissions } = useUserPermissions();
  const { issues, issuesFilter } = useIssues(storeType);
  const {
    fetchIssues,
    fetchNextIssues,
    createIssue,
    updateIssue,
    removeIssue,
    archiveIssue,
    restoreIssue,
    updateFilters,
  } = useIssuesActions(storeType);
  // derived values
  const { enableInlineEditing, enableQuickAdd, enableIssueCreation } = issues?.viewFlags || {};
  // user role validation
  const isEditingAllowed =
    !!workspaceSlug &&
    !!projectId &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );

  useIssueListMount(storeType, issues.rootIssueStore, fetchIssues, { limit: 100 }, viewId);

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && targetProjectId
          ? canEditPropertiesBasedOnProject(targetProjectId)
          : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  const staleRefresh = useIssueListRefresh(issues.currentList, [{ group: "" }]);
  const issueIds = issues.getIssueIds() ?? [];
  const nextCursor = issues.getPaginationData(undefined)?.nextCursor;

  const handleDisplayFiltersUpdate = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      void updateFilters(projectId ?? "", EIssueFilterType.DISPLAY_FILTERS, {
        ...updatedDisplayFilter,
      }).catch(() => {});
    },
    [projectId, updateFilters]
  );

  const renderQuickActions: TRenderQuickActions = useCallback(
    ({ issue, parentRef, customActionButton, placement, portalElement }) => (
      <QuickActions
        parentRef={parentRef}
        customActionButton={customActionButton}
        issue={issue}
        handleDelete={async () => removeIssue(issue.project_id, issue.id)}
        handleUpdate={async (data) => updateIssue && updateIssue(issue.project_id, issue.id, data)}
        handleArchive={async () => archiveIssue && archiveIssue(issue.project_id, issue.id)}
        handleRestore={async () => restoreIssue && restoreIssue(issue.project_id, issue.id)}
        portalElement={portalElement}
        readOnly={!canEditProperties(issue.project_id ?? undefined)}
        placements={placement}
      />
    ),
    [QuickActions, canEditProperties, removeIssue, updateIssue, archiveIssue, restoreIssue]
  );

  if (!Array.isArray(issueIds)) return null;

  return (
    <>
      <IssueListRetry {...staleRefresh} />
      <IssueLayoutHOC layout={EIssueLayoutTypes.SPREADSHEET}>
        <SpreadsheetView
          displayProperties={issuesFilter.issueFilters?.displayProperties ?? {}}
          displayFilters={issuesFilter.issueFilters?.displayFilters ?? {}}
          handleDisplayFilterUpdate={handleDisplayFiltersUpdate}
          issueIds={issueIds}
          quickActions={renderQuickActions}
          updateIssue={updateIssue}
          canEditProperties={canEditProperties}
          quickAddCallback={createIssue}
          enableQuickCreateIssue={enableQuickAdd}
          disableIssueCreation={!enableIssueCreation || !isEditingAllowed}
          canLoadMoreIssues={!!nextCursor}
          loadMoreIssues={() => {
            void fetchNextIssues().catch(() => {});
          }}
        />
      </IssueLayoutHOC>
    </>
  );
});
