/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueUpdate } from "@nerve/api-client";
import { useState, useMemo } from "react";
import { observer } from "mobx-react";
// Nerve imports
import useSWR from "swr";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { followInSession } from "@/lib/in-session";
import type { IWorkItemPeekOverview } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
// local imports
import type { TIssueOperations } from "../issue-detail";
import { IssueView } from "./view";

export const IssuePeekOverview = observer(function IssuePeekOverview(props: IWorkItemPeekOverview) {
  const { embedIssue = false, embedRemoveCurrentNotification, storeType: issueStoreFromProps } = props;
  const { t } = useTranslation();
  // store hook
  const { allowPermissions } = useUserPermissions();

  const {
    issues: { restoreIssue },
  } = useIssues(EIssuesStoreType.ARCHIVED);
  const {
    peekIssue,
    issue: { fetchIssue },
    fetchActivities,
  } = useIssueDetail();
  const issueStoreType = useIssueStoreType();
  const storeType = issueStoreFromProps ?? issueStoreType;
  const { issues } = useIssues(storeType);

  // state
  const [error, setError] = useState(false);

  const issueOperations: TIssueOperations = useMemo(
    () => ({
      fetch: async (workspaceSlug: string, projectId: string, issueId: string) => {
        try {
          setError(false);
          await fetchIssue(workspaceSlug, projectId, issueId);
        } catch (error) {
          setError(true);
          console.error("Error fetching the parent issue", error);
        }
      },
      update: async (workspaceSlug: string, projectId: string, issueId: string, data: IssueUpdate) => {
        if (issues?.updateIssue)
          await followInSession(
            async () => {
              await issues.updateIssue(workspaceSlug, projectId, issueId, data);
            },
            {
              done: async () => {
                await fetchActivities(workspaceSlug, projectId, issueId);
              },
              failed: () =>
                setToast({
                  title: t("toast.error"),
                  type: TOAST_TYPE.ERROR,
                  message: t("entity.update.failed", { entity: t("issue.label", { count: 1 }) }),
                }),
            }
          );
      },
      restore: async (workspaceSlug: string, projectId: string, issueId: string) => {
        await followInSession(() => restoreIssue(workspaceSlug, projectId, issueId), {
          done: () =>
            setToast({
              type: TOAST_TYPE.SUCCESS,
              title: t("issue.restore.success.title"),
              message: t("issue.restore.success.message"),
            }),
          failed: () =>
            setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t("issue.restore.failed.message") }),
        });
      },
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [fetchIssue, issues, fetchActivities, restoreIssue]
  );

  const { isLoading } = useSWR(
    ["peek-issue", peekIssue?.workspaceSlug, peekIssue?.projectId, peekIssue?.issueId],
    () => peekIssue && issueOperations.fetch(peekIssue.workspaceSlug, peekIssue.projectId, peekIssue.issueId),
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  );

  if (!peekIssue?.workspaceSlug || !peekIssue?.projectId || !peekIssue?.issueId) return <></>;

  // Check if issue is editable, based on user role
  const isEditable = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    peekIssue?.workspaceSlug,
    peekIssue?.projectId
  );

  return (
    <IssueView
      workspaceSlug={peekIssue.workspaceSlug}
      projectId={peekIssue.projectId}
      issueId={peekIssue.issueId}
      isLoading={isLoading}
      isError={error}
      is_archived={!!peekIssue.isArchived}
      disabled={!isEditable}
      embedIssue={embedIssue}
      embedRemoveCurrentNotification={embedRemoveCurrentNotification}
      issueOperations={issueOperations}
    />
  );
});
