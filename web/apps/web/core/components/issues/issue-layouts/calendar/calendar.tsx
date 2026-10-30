import type { IssueCreate } from "@nerve/api-client";
import type { Group } from "@/lib/paged-groups";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import { useEffect, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
// nerve constants
import type { TSupportedFilterTypeForUpdate } from "@nerve/constants";
// types
import type { ICalendarWeek, TSupportedFilterForUpdate } from "@nerve/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@nerve/types";
// ui
import { Spinner } from "@nerve/ui";
import { renderFormattedPayloadDate, cn } from "@nerve/utils";
// constants
import { MONTHS_LIST } from "@nerve/constants";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import useSize from "@/hooks/use-window-size";
// store
import type { ICalendarStore } from "@/store/issue/issue_calendar_view.store";
import type { IProjectIssuesFilter } from "@/store/issue/project";
// local imports
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { TRenderQuickActions } from "../list/list-view-types";
import { CalendarHeader } from "./header";
import { CalendarIssueBlocks } from "./issue-blocks";
import { CalendarWeekDays } from "./week-days";
import { CalendarWeekHeader } from "./week-header";

type Props = {
  issuesFilterStore: IProjectIssuesFilter;
  issues: Record<string, Issue | IssueDetail> | undefined;
  groups: Map<string, Group>;
  layout: "month" | "week" | undefined;
  showWeekends: boolean;
  issueCalendarView: ICalendarStore;
  loadMoreIssues: (dateString: string) => void;
  getPaginationData: (groupId: string | undefined) => Group | undefined;
  getGroupIssueCount: (groupId: string | undefined) => number | undefined;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  quickActions: TRenderQuickActions;
  handleDragAndDrop: (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => Promise<void>;
  readOnly?: boolean;
  updateFilters?: (
    projectId: string,
    filterType: TSupportedFilterTypeForUpdate,
    filters: TSupportedFilterForUpdate
  ) => Promise<void>;
  canEditProperties: (projectId: string | undefined) => boolean;
};

export const CalendarChart = observer(function CalendarChart(props: Props) {
  const {
    issuesFilterStore,
    issues,
    groups,
    layout,
    showWeekends,
    issueCalendarView,
    loadMoreIssues,
    handleDragAndDrop,
    quickActions,
    quickAddCallback,
    getPaginationData,
    getGroupIssueCount,
    updateFilters,
    canEditProperties,
    readOnly = false,
  } = props;
  // states
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  //refs
  const [scrollableContainer, setScrollableContainer] = useState<HTMLDivElement | null>(null);
  // store hooks
  const {
    issues: { viewFlags },
  } = useIssues(EIssuesStoreType.PROJECT);

  const [windowWidth] = useSize();

  const { enableIssueCreation, enableQuickAdd } = viewFlags || {};

  const calendarPayload = issueCalendarView.calendarPayload;

  const allWeeksOfActiveMonth = issueCalendarView.allWeeksOfActiveMonth;

  const formattedDatePayload = renderFormattedPayloadDate(selectedDate) ?? undefined;

  // Enable Auto Scroll for calendar
  useEffect(() => {
    const element = scrollableContainer;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, [scrollableContainer]);

  if (!calendarPayload || !formattedDatePayload)
    return (
      <div className="grid h-full w-full place-items-center">
        <Spinner />
      </div>
    );

  const issueIdList = groups ? (groups.get(formattedDatePayload)?.rows.map((row) => row.id) ?? []) : [];

  return (
    <>
      <div className="flex h-full w-full flex-col overflow-hidden">
        <CalendarHeader
          setSelectedDate={setSelectedDate}
          issuesFilterStore={issuesFilterStore}
          updateFilters={updateFilters}
        />

        <IssueLayoutHOC layout={EIssueLayoutTypes.CALENDAR}>
          <div
            className={cn("flex w-full flex-col overflow-y-auto md:h-full", {
              "vertical-scrollbar scrollbar-lg": windowWidth > 768,
            })}
            ref={setScrollableContainer}
          >
            <CalendarWeekHeader isLoading={!issues} showWeekends={showWeekends} />
            <div className="h-full w-full">
              {layout === "month" && (
                <div className="grid h-full w-full grid-cols-1 divide-y-[0.5px] divide-subtle-1">
                  {allWeeksOfActiveMonth &&
                    Object.entries(allWeeksOfActiveMonth).map(([weekKey, week]: [string, ICalendarWeek]) => (
                      <CalendarWeekDays
                        selectedDate={selectedDate}
                        setSelectedDate={setSelectedDate}
                        handleDragAndDrop={handleDragAndDrop}
                        issuesFilterStore={issuesFilterStore}
                        key={weekKey}
                        week={week}
                        issues={issues}
                        groups={groups}
                        loadMoreIssues={loadMoreIssues}
                        getPaginationData={getPaginationData}
                        getGroupIssueCount={getGroupIssueCount}
                        enableQuickIssueCreate={enableQuickAdd}
                        disableIssueCreation={!enableIssueCreation}
                        quickActions={quickActions}
                        quickAddCallback={quickAddCallback}
                        readOnly={readOnly}
                        canEditProperties={canEditProperties}
                      />
                    ))}
                </div>
              )}
              {layout === "week" && (
                <CalendarWeekDays
                  selectedDate={selectedDate}
                  setSelectedDate={setSelectedDate}
                  handleDragAndDrop={handleDragAndDrop}
                  issuesFilterStore={issuesFilterStore}
                  week={issueCalendarView.allDaysOfActiveWeek}
                  issues={issues}
                  groups={groups}
                  loadMoreIssues={loadMoreIssues}
                  getPaginationData={getPaginationData}
                  getGroupIssueCount={getGroupIssueCount}
                  enableQuickIssueCreate={enableQuickAdd}
                  disableIssueCreation={!enableIssueCreation}
                  quickActions={quickActions}
                  quickAddCallback={quickAddCallback}
                  readOnly={readOnly}
                  canEditProperties={canEditProperties}
                />
              )}
            </div>

            {/* mobile view */}
            <div className="md:hidden">
              <p className="p-4 text-18 font-semibold">
                {`${selectedDate.getDate()} ${
                  MONTHS_LIST[selectedDate.getMonth() + 1].title
                }, ${selectedDate.getFullYear()}`}
              </p>
              <CalendarIssueBlocks
                date={selectedDate}
                issueIdList={issueIdList}
                loadMoreIssues={loadMoreIssues}
                getPaginationData={getPaginationData}
                getGroupIssueCount={getGroupIssueCount}
                quickActions={quickActions}
                enableQuickIssueCreate={enableQuickAdd}
                disableIssueCreation={!enableIssueCreation}
                quickAddCallback={quickAddCallback}
                readOnly={readOnly}
                canEditProperties={canEditProperties}
                isDragDisabled
                isMobileView
              />
            </div>
          </div>
        </IssueLayoutHOC>
      </div>
    </>
  );
});
