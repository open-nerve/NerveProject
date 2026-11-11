/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
import { observer } from "mobx-react";
import { CalendarOutline, MembersOutline, ShowOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import type { CycleListItem } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
import { AvatarGroupOverflow } from "@/components/common/avatar-group-overflow";
import { MergedDateDisplay } from "@/components/dropdowns/merged-date";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// local imports
import { CycleQuickActions } from "../quick-actions";
import { TransferIssuesButton } from "../transfer-issues";

type Props = {
  workspaceSlug: string;
  projectId: string;
  row: CycleListItem;
  parentRef: React.RefObject<HTMLDivElement | null>;
  /** Whether the page previews a cycle (?peekCycle). */
  isPeekOpen: boolean;
  /** Previews the cycle, or closes its preview. */
  onPeek: () => void;
};

/**
 * The actions and the facts of a cycle's row (P5a spec 2.5): "more details", which previews it; its work items, while
 * a draft or upcoming; the transfer of its unfinished work items, once it has ended (TransferIssuesButton says when);
 * its days, days of its project's calendar ("YYYY-MM-DD", M6 design 3.1, R6), never shifted by the browser's time
 * zone; its creator; its members, the assignees of its work items (but in the card of the cycle in progress, whose
 * members it shows itself); and its quick actions. People show as the project's members, one who has left it as such.
 */
export const CycleListItemAction = observer(function CycleListItemAction(props: Props) {
  const { workspaceSlug, projectId, row, parentRef, isPeekOpen, onPeek } = props;
  // hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const imageWorkspace = useRouteWorkspace();
  const {
    project: { getProjectMemberDetails },
  } = useMember();

  const isActive = row.status === "current";
  const showIssueCount = row.status === "draft" || row.status === "upcoming";
  const memberOf = (accountId: string) => getProjectMemberDetails(accountId, projectId)?.member;
  const avatarOf = (accountId: string) => (
    <ProfileAvatar
      key={accountId}
      alt={memberOf(accountId)?.display_name ?? t("distribution.former_member")}
      fallback={memberOf(accountId)?.display_name[0]?.toUpperCase()}
      source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: accountId }}
      size="xs"
    />
  );

  const handlePeek = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    onPeek();
  };

  return (
    <>
      <button
        type="button"
        onClick={handlePeek}
        className={cn("z-[1] flex-shrink-0 gap-1 text-11 text-accent-secondary", {
          flex: isMobile || (isActive && !isPeekOpen),
          "hidden group-hover:flex": !(isMobile || (isActive && !isPeekOpen)),
        })}
      >
        <ShowOutline className="my-auto h-4 w-4 text-accent-secondary" aria-hidden="true" />
        <span>{t("project_cycles.more_details")}</span>
      </button>
      {showIssueCount && (
        <div className="flex items-center gap-1 text-11 text-tertiary" title={t("work_items")}>
          <WorkItemsOutline className="h-4 w-4" aria-hidden="true" />
          <span>{row.total_issues}</span>
        </div>
      )}
      <TransferIssuesButton row={row} workspaceSlug={workspaceSlug} projectId={projectId} />
      {row.start_date !== null && (
        <div className="flex items-center gap-1 text-11 font-medium text-tertiary">
          <CalendarOutline className="my-auto h-3 w-3 flex-shrink-0" aria-hidden="true" />
          <MergedDateDisplay startDate={row.start_date} endDate={row.end_date} />
        </div>
      )}
      {row.created_by_id !== null && (
        <span className="flex" title={t("common.created_by")}>
          {avatarOf(row.created_by_id)}
        </span>
      )}
      {!isActive && (
        <Tooltip
          label={t("project_cycles.member_count", { count: row.assignee_ids.length })}
          layout="stacked"
          disabled={isMobile}
        >
          <div className="flex w-min cursor-default items-center justify-center">
            {row.assignee_ids.length > 0 ? (
              <AvatarGroupOverflow size="xs">
                {row.assignee_ids.map((assignee) => avatarOf(assignee))}
              </AvatarGroupOverflow>
            ) : (
              <MembersOutline className="h-4 w-4 text-tertiary" aria-hidden="true" />
            )}
          </div>
        </Tooltip>
      )}
      <CycleQuickActions parentRef={parentRef} cycleId={row.id} projectId={projectId} workspaceSlug={workspaceSlug} />
    </>
  );
});
