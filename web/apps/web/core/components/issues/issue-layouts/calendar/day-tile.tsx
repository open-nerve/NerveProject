import type { IssueCreate } from "@nerve/api-client";
import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/element/adapter";
import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { ICalendarDate } from "@nerve/types";
// components
import { cn, renderFormattedPayloadDate } from "@nerve/utils";
// helpers
import { MONTHS_LIST } from "@nerve/constants";
// types
import type { IProjectIssuesFilter } from "@/store/issue/project";
import type { TRenderQuickActions } from "../list/list-view-types";
import { CalendarIssueBlocks } from "./issue-blocks";

type Props = {
  issuesFilterStore: IProjectIssuesFilter;
  date: ICalendarDate;
  issues: Record<string, Issue | IssueDetail> | undefined;
  groups: Map<string, Group>;
  loadMoreIssues: (dateString: string) => void;
  getPaginationData: (groupId: string | undefined) => Group | undefined;
  getGroupIssueCount: (groupId: string | undefined) => number | undefined;
  enableQuickIssueCreate?: boolean;
  disableIssueCreation?: boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  quickActions: TRenderQuickActions;
  handleDragAndDrop: (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => Promise<void>;
  readOnly?: boolean;
  selectedDate: Date;
  setSelectedDate: (date: Date) => void;
  canEditProperties: (projectId: string | undefined) => boolean;
};

export const CalendarDayTile = observer(function CalendarDayTile(props: Props) {
  const {
    issuesFilterStore,
    date,
    issues,
    groups,
    loadMoreIssues,
    getPaginationData,
    getGroupIssueCount,
    quickActions,
    enableQuickIssueCreate,
    disableIssueCreation,
    quickAddCallback,
    readOnly = false,
    selectedDate,
    handleDragAndDrop,
    setSelectedDate,
    canEditProperties,
  } = props;

  const [isDraggingOver, setIsDraggingOver] = useState(false);

  const calendarLayout = issuesFilterStore?.issueFilters?.displayFilters?.calendar?.layout ?? "month";

  const formattedDatePayload = renderFormattedPayloadDate(date.date);

  const { t } = useTranslation();
  const dayTileRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const element = dayTileRef.current;

    if (!element) return;

    return combine(
      dropTargetForElements({
        element,
        getData: () => ({ date: formattedDatePayload }),
        onDragEnter: () => {
          setIsDraggingOver(true);
        },
        onDragLeave: () => {
          setIsDraggingOver(false);
        },
        onDrop: ({ source, self }) => {
          setIsDraggingOver(false);
          const sourceData = source?.data as { id: string; date: string } | undefined;
          const destinationData = self?.data as { date: string } | undefined;
          if (!sourceData || !destinationData) return;

          const issueDetails = issues?.[sourceData?.id];
          if (issueDetails?.start_date && destinationData.date < issueDetails.start_date) {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: t("toast.error"),
              message: t("issue.calendar.due_before_start"),
            });
            return;
          }

          handleDragAndDrop(
            sourceData?.id,
            issueDetails?.project_id ?? undefined,
            sourceData?.date,
            destinationData?.date
          );
        },
      })
    );
  }, [formattedDatePayload, issues, handleDragAndDrop, t]);

  if (!formattedDatePayload) return null;
  const issueIds = groups?.get(formattedDatePayload)?.rows.map((row) => row.id) ?? [];

  const isToday = date.date.toDateString() === new Date().toDateString();
  const isSelectedDate = date.date.toDateString() == selectedDate.toDateString();

  const isWeekend = [0, 6].includes(date.date.getDay());
  const isMonthLayout = calendarLayout === "month";

  const normalBackground = isWeekend ? "bg-layer-1" : "bg-layer-transparent";
  const draggingOverBackground = isWeekend ? "bg-layer-1" : "bg-layer-transparent-hover";

  return (
    <>
      <div ref={dayTileRef} className="group relative flex h-full w-full flex-col">
        {/* header */}
        <div
          className={`hidden flex-shrink-0 justify-end px-2 py-1.5 text-right text-11 md:flex ${
            isMonthLayout // if month layout, highlight current month days
              ? date.is_current_month
                ? "font-medium"
                : "text-tertiary"
              : "font-medium" // if week layout, highlight all days
          } ${isWeekend ? "bg-layer-1" : "bg-layer-transparent"} `}
        >
          {date.date.getDate() === 1 && MONTHS_LIST[date.date.getMonth() + 1].shortTitle + " "}
          {isToday ? (
            <span className="flex h-5 w-5 items-center justify-center rounded-full bg-accent-primary text-on-color">
              {date.date.getDate()}
            </span>
          ) : (
            <>{date.date.getDate()}</>
          )}
        </div>

        {/* content */}
        <div className="hidden h-full w-full md:block">
          <div
            className={cn(
              `h-full w-full select-none ${isDraggingOver ? `${draggingOverBackground} opacity-70` : normalBackground}`,
              {
                "min-h-[5rem]": isMonthLayout,
              }
            )}
          >
            <CalendarIssueBlocks
              date={date.date}
              issueIdList={issueIds}
              quickActions={quickActions}
              loadMoreIssues={loadMoreIssues}
              getPaginationData={getPaginationData}
              getGroupIssueCount={getGroupIssueCount}
              isDragDisabled={readOnly}
              disableIssueCreation={disableIssueCreation}
              enableQuickIssueCreate={enableQuickIssueCreate}
              quickAddCallback={quickAddCallback}
              readOnly={readOnly}
              canEditProperties={canEditProperties}
            />
          </div>
        </div>

        {/* Mobile view content */}
        <button
          type="button"
          aria-label={formattedDatePayload}
          aria-pressed={isSelectedDate}
          onClick={() => setSelectedDate(date.date)}
          className={cn(
            "mx-auto flex h-full w-full cursor-pointer flex-col items-center justify-start py-2.5 text-13 font-medium opacity-80 md:hidden",
            {
              "bg-layer-2": !isWeekend,
            }
          )}
        >
          <span
            className={cn("flex size-6 items-center justify-center rounded-full", {
              "bg-accent-primary text-on-color": isSelectedDate,
              "bg-accent-primary/10 text-accent-primary": isToday && !isSelectedDate,
            })}
          >
            {date.date.getDate()}
          </span>
        </button>
      </div>
    </>
  );
});
