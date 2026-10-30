/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useEffect, useRef, useState, forwardRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import { Popover } from "@nerve/propel/popover";

import { ControlLink } from "@nerve/ui";
import { cn, generateWorkItemLink } from "@nerve/utils";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// local components
import { WorkItemPreviewCard } from "../../preview-card";
import type { TRenderQuickActions } from "../list/list-view-types";
import type { CalendarStoreType } from "./base-calendar-root";

type Props = {
  issue: Issue;
  quickActions: TRenderQuickActions;
  isDragging?: boolean;
};

export const CalendarIssueBlock = observer(
  forwardRef(function CalendarIssueBlock(props: Props, ref: React.ForwardedRef<HTMLAnchorElement>) {
    const { issue, quickActions, isDragging = false } = props;
    const [previewOpen, setPreviewOpen] = useState(false);
    useEffect(() => {
      if (isDragging) setPreviewOpen(false);
    }, [isDragging]);
    // refs
    const blockRef = useRef(null);
    const menuActionRef = useRef<HTMLSpanElement | null>(null);
    // hooks
    const { workspaceSlug } = useParams();
    const { getProjectStates } = useProjectState();
    const { getIsIssuePeeked } = useIssueDetail();
    const { handleRedirection } = useIssuePeekOverviewRedirection();
    const { isMobile } = usePlatformOS();
    const storeType = useIssueStoreType() as CalendarStoreType;
    const { issuesFilter } = useIssues(storeType);
    const { getProjectIdentifierById } = useProject();

    const stateColor = getProjectStates(issue?.project_id)?.find((state) => state?.id == issue?.state_id)?.color || "";
    const projectIdentifier = getProjectIdentifierById(issue?.project_id);

    // handlers
    const handleIssuePeekOverview = (peekIssue: Issue, trigger: HTMLElement) =>
      handleRedirection(workspaceSlug, peekIssue, isMobile, undefined, trigger);

    const customActionButton = (
      <span ref={menuActionRef} className="block w-full rounded-sm p-1">
        <MoreHorizontalOutline className="h-3.5 w-3.5" />
      </span>
    );

    const isMenuActionRefAboveScreenBottom =
      typeof window !== "undefined" &&
      menuActionRef?.current &&
      menuActionRef?.current?.getBoundingClientRect().bottom < window.innerHeight - 220;

    const placement = isMenuActionRefAboveScreenBottom ? "bottom-end" : "top-end";

    const workItemLink = generateWorkItemLink({
      workspaceSlug,
      projectId: issue?.project_id,
      issueId: issue?.id,
      projectIdentifier,
      sequenceId: issue?.sequence_id,
      isArchived: !!issue?.archived_at,
    });

    return (
      <Popover
        delay={100}
        openOnHover={!isDragging}
        open={previewOpen && !isDragging}
        onOpenChange={(open) => setPreviewOpen(open && !isDragging)}
      >
        <Popover.Button
          nativeButton={false}
          className="w-full"
          render={
            <ControlLink
              id={`issue-${issue.id}`}
              href={workItemLink}
              onClick={(event) => handleIssuePeekOverview(issue, event.currentTarget)}
              className="block w-full rounded-sm border-b border-subtle text-13 text-primary hover:border-subtle-1 md:border-[1px]"
              disabled={isMobile}
              ref={ref}
            >
              <>
                <div
                  ref={blockRef}
                  className={cn(
                    "group/calendar-block flex h-10 w-full items-center justify-between gap-1.5 rounded-sm px-4 py-1.5 md:h-8 md:px-1",
                    {
                      "border-accent-strong bg-surface-2 shadow-raised-200": isDragging,
                      "bg-surface-1 hover:bg-surface-2": !isDragging,
                      "border border-accent-strong hover:border-accent-strong": getIsIssuePeeked(issue.id),
                    }
                  )}
                >
                  <div className="flex h-full items-center gap-1.5 truncate">
                    <span
                      className="h-full w-0.5 flex-shrink-0 rounded-sm"
                      style={{
                        backgroundColor: stateColor,
                      }}
                    />
                    {issue.project_id && (
                      <IssueIdentifier
                        issueId={issue.id}
                        projectId={issue.project_id}
                        size="xs"
                        variant="tertiary"
                        displayProperties={issuesFilter?.issueFilters?.displayProperties}
                      />
                    )}
                    <div className="truncate text-13 font-medium md:text-11 md:font-regular">{issue.name}</div>
                  </div>
                  <div
                    className={cn("size-5 flex-shrink-0 rounded-sm has-[[aria-expanded=true]]:bg-layer-1-active", {
                      "opacity-0 group-hover/calendar-block:opacity-100 focus-within:opacity-100 has-[[aria-expanded=true]]:opacity-100":
                        !isMobile,
                    })}
                  >
                    {quickActions({
                      issue,
                      parentRef: blockRef,
                      customActionButton,
                      placement,
                    })}
                  </div>
                </div>
              </>
            </ControlLink>
          }
        />
        <Popover.Panel side="bottom" align="start">
          <>
            {issue.project_id && (
              <WorkItemPreviewCard
                projectId={issue.project_id}
                stateDetails={{
                  id: issue.state_id ?? undefined,
                }}
                workItem={issue}
              />
            )}
          </>
        </Popover.Panel>
      </Popover>
    );
  })
);

CalendarIssueBlock.displayName = "CalendarIssueBlock";
