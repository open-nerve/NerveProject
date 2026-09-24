/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useRef } from "react";
import { observer } from "mobx-react";
// icons
import { ModuleOutline, PreferencesOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
// plane imports
import {
  EIssueFilterType,
  ISSUE_DISPLAY_FILTERS_BY_PAGE,
  EUserPermissions,
  EUserPermissionsLevel,
} from "@nerve/constants";
import { Button } from "@nerve/propel/button";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { ICustomSearchSelectOption, IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@nerve/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@nerve/types";
import { Breadcrumbs, Header, BreadcrumbNavigationSearchDropdown } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { SwitcherLabel } from "@/components/common/switcher-label";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  LayoutSelection,
  MobileLayoutSelection,
} from "@/components/issues/issue-layouts/filters";
import { ModuleQuickActions } from "@/components/modules";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useIssues } from "@/hooks/store/use-issues";
import { useModule } from "@/hooks/store/use-module";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useNavigate } from "react-router";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import useLocalStorage from "@/hooks/use-local-storage";
import { usePlatformOS } from "@/hooks/use-platform-os";
// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { IconButton } from "@nerve/propel/icon-button";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
};

export const ModuleIssuesHeader = observer(function ModuleIssuesHeader(props: TProps) {
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  const { workspaceSlug, projectId, moduleId } = props;
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const {
    issuesFilter: { issueFilters },
    issues: { getGroupIssueCount },
  } = useIssues(EIssuesStoreType.MODULE);
  const { updateFilters } = useIssuesActions(EIssuesStoreType.MODULE);
  const { projectModuleIds, getModuleById } = useModule();
  const { toggleCreateIssueModal } = useCommandPalette();
  const { allowPermissions } = useUserPermissions();
  const { currentProjectDetails, loader } = useProject();
  // local storage
  const { setValue, storedValue } = useLocalStorage("module_sidebar_collapsed", "false");
  // derived values
  const isSidebarCollapsed = storedValue ? storedValue === "true" : false;
  const activeLayout = issueFilters?.displayFilters?.layout;
  const moduleDetails = getModuleById(moduleId);
  const canUserCreateIssue = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT
  );
  const workItemsCount = getGroupIssueCount(undefined, undefined, false);

  const toggleSidebar = () => {
    setValue(`${!isSidebarCollapsed}`);
  };

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      updateFilters(projectId, EIssueFilterType.DISPLAY_FILTERS, { layout: layout });
    },
    [projectId, updateFilters]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      updateFilters(projectId, EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
    },
    [projectId, updateFilters]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      updateFilters(projectId, EIssueFilterType.DISPLAY_PROPERTIES, property);
    },
    [projectId, updateFilters]
  );

  const switcherOptions = projectModuleIds
    ?.map((id) => {
      const _module = id === moduleId ? moduleDetails : getModuleById(id);
      if (!_module) return;
      return {
        value: _module.id,
        query: _module.name,
        content: <SwitcherLabel name={_module.name} LabelIcon={ModuleOutline} />,
      };
    })
    .filter((option) => option !== undefined) as ICustomSearchSelectOption[];

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex items-center gap-2">
          <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label="Modules"
                  href={`/${workspaceSlug}/projects/${projectId}/modules`}
                  icon={<ModuleOutline className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
            <Breadcrumbs.Item
              component={
                <BreadcrumbNavigationSearchDropdown
                  selectedItem={moduleId}
                  navigationItems={switcherOptions}
                  onChange={(value: string) => {
                    navigate(`/${workspaceSlug}/projects/${projectId}/modules/${value}`);
                  }}
                  title={moduleDetails?.name}
                  icon={<ModuleOutline className="size-3.5 flex-shrink-0 text-tertiary" />}
                  isLast
                />
              }
            />
          </Breadcrumbs>
          {workItemsCount && workItemsCount > 0 ? (
            <Tooltip
              label={`There are ${workItemsCount} ${workItemsCount > 1 ? "work items" : "work item"} in this module`}
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
        <div className="hidden gap-2 md:flex">
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
          <WorkItemFiltersToggle entityType={EIssuesStoreType.MODULE} entityId={moduleId} />
          <FiltersDropdown
            title="Display"
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
              ignoreGroupedFilters={["module"]}
              cycleViewDisabled={!currentProjectDetails?.cycle_view}
              moduleViewDisabled={!currentProjectDetails?.module_view}
            />
          </FiltersDropdown>
        </div>

        {canUserCreateIssue && (
          <Button
            variant="primary"
            size="lg"
            className="hidden sm:flex"
            onClick={() => {
              toggleCreateIssueModal(true, EIssuesStoreType.MODULE);
            }}
          >
            Add work item
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
        <ModuleQuickActions
          parentRef={parentRef}
          moduleId={moduleId}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
          customClassName="flex-shrink-0 flex items-center justify-center bg-layer-1/70 rounded-sm size-[26px]"
        />
      </Header.RightItem>
    </Header>
  );
});
