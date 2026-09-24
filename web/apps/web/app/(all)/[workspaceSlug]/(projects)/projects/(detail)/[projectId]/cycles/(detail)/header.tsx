/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useRef } from "react";
import { observer } from "mobx-react";
// icons
import { CyclesOutline, PreferencesOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
// plane imports
import {
  EIssueFilterType,
  EUserPermissions,
  EUserPermissionsLevel,
  ISSUE_DISPLAY_FILTERS_BY_PAGE,
} from "@nerve/constants";
import { usePlatformOS } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { IconButton } from "@nerve/propel/icon-button";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { ICustomSearchSelectOption, IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@nerve/types";
import { Breadcrumbs, BreadcrumbNavigationSearchDropdown, Header } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { CycleQuickActions } from "@/components/cycles/quick-actions";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  LayoutSelection,
  MobileLayoutSelection,
} from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useCycle } from "@/hooks/store/use-cycle";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useNavigate } from "react-router";
import useLocalStorage from "@/hooks/use-local-storage";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
};

export const CycleIssuesHeader = observer(function CycleIssuesHeader(props: TProps) {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, cycleId } = props;
  // i18n
  const { t } = useTranslation();
  // store hooks
  const {
    issuesFilter: { issueFilters, updateFilters },
    issues: { getGroupIssueCount },
  } = useIssues(EIssuesStoreType.CYCLE);
  const { currentProjectCycleIds, getCycleById } = useCycle();
  const { toggleCreateIssueModal } = useCommandPalette();
  const { currentProjectDetails, loader } = useProject();
  const { isMobile } = usePlatformOS();
  const { allowPermissions } = useUserPermissions();

  const activeLayout = issueFilters?.displayFilters?.layout;

  const { setValue, storedValue } = useLocalStorage("cycle_sidebar_collapsed", false);

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => {
    setValue(!isSidebarCollapsed);
  };

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_FILTERS, { layout: layout }, cycleId);
    },
    [workspaceSlug, projectId, cycleId, updateFilters]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter, cycleId);
    },
    [workspaceSlug, projectId, cycleId, updateFilters]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      updateFilters(workspaceSlug, projectId, EIssueFilterType.DISPLAY_PROPERTIES, property, cycleId);
    },
    [workspaceSlug, projectId, cycleId, updateFilters]
  );

  // derived values
  const cycleDetails = getCycleById(cycleId);
  const isCompletedCycle = cycleDetails?.status?.toLocaleLowerCase() === "completed";
  const canUserCreateIssue = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT
  );

  const switcherOptions = currentProjectCycleIds
    ?.map((id) => {
      const _cycle = id === cycleId ? cycleDetails : getCycleById(id);
      if (!_cycle) return;
      return {
        value: _cycle.id,
        query: _cycle.name,
        content: <SwitcherLabel name={_cycle.name} LabelIcon={CyclesOutline} />,
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  const workItemsCount = getGroupIssueCount(undefined, undefined, false);

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex items-center gap-2">
          <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Cycles"
                  href={`/${workspaceSlug}/projects/${projectId}/cycles`}
                  icon={<CyclesOutline className="h-4 w-4 text-tertiary" />}
                />
              }
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={cycleId}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    navigate(`/${workspaceSlug}/projects/${projectId}/cycles/${value}`);
                  }}
                  title={cycleDetails?.name}
                  icon={
                    <Breadcrumbs.Icon>
                      <CyclesOutline className="size-4 flex-shrink-0 text-tertiary" />
                    </Breadcrumbs.Icon>
                  }
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
          {workItemsCount && workItemsCount > 0 ? (
            <Tooltip
              label={`There are ${workItemsCount} ${workItemsCount > 1 ? "work items" : "work item"} in this cycle`}
              layout="stacked"
              side="bottom"
              disabled={isMobile}
            >
              <span className="flex flex-shrink-0 cursor-default items-center justify-center rounded-xl bg-accent-primary/20 px-2 text-center text-11 font-semibold text-accent-primary">
                {workItemsCount}
              </span>
            </Tooltip>
          ) : null}
        </div>
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <div className="hidden items-center gap-2 md:flex">
          <div className="hidden @4xl:flex">
            <LayoutSelection
              layouts={[
                EIssueLayoutTypes.LIST,
                EIssueLayoutTypes.KANBAN,
                EIssueLayoutTypes.CALENDAR,
                EIssueLayoutTypes.SPREADSHEET,
              ]}
              onChange={(layout) => handleLayoutChange(layout)}
              selectedLayout={activeLayout}
            />
          </div>
          <div className="flex @4xl:hidden">
            <MobileLayoutSelection
              layouts={[
                EIssueLayoutTypes.LIST,
                EIssueLayoutTypes.KANBAN,
                EIssueLayoutTypes.CALENDAR,
                EIssueLayoutTypes.SPREADSHEET,
              ]}
              onChange={(layout) => handleLayoutChange(layout)}
              activeLayout={activeLayout}
            />
          </div>
          <WorkItemFiltersToggle entityType={EIssuesStoreType.CYCLE} entityId={cycleId} />
          <FiltersDropdown
            title={t("common.display")}
            placement="bottom-end"
            miniIcon={<PreferencesOutline className="size-3.5" />}
          >
            <DisplayFiltersSelection
              layoutDisplayFiltersOptions={
                activeLayout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions[activeLayout] : undefined
              }
              displayFilters={issueFilters?.displayFilters ?? {}}
              handleDisplayFiltersUpdate={handleDisplayFilters}
              displayProperties={issueFilters?.displayProperties ?? {}}
              handleDisplayPropertiesUpdate={handleDisplayProperties}
              ignoreGroupedFilters={["cycle"]}
              cycleViewDisabled={!currentProjectDetails?.cycle_view}
              moduleViewDisabled={!currentProjectDetails?.module_view}
            />
          </FiltersDropdown>

          {canUserCreateIssue && !isCompletedCycle && (
            <Button
              variant="primary"
              size="lg"
              onClick={() => {
                toggleCreateIssueModal(true, EIssuesStoreType.CYCLE);
              }}
            >
              {t("issue.add.label")}
            </Button>
          )}
          <IconButton
            variant="tertiary"
            size="lg"
            icon={RightSidePaneOutline}
            onClick={toggleSidebar}
            className={cn({
              "bg-accent-subtle text-accent-primary": !isSidebarCollapsed,
            })}
          />
          <CycleQuickActions
            parentRef={parentRef}
            cycleId={cycleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            customClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
          />
        </div>
      </Header.RightItem>
    </Header>
  );
});
