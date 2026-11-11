/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// icons
import { CyclesOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { IconButton } from "@nerve/propel/icon-button";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { Breadcrumbs, BreadcrumbNavigationSearchDropdown, Header } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { CountChip } from "@/components/common/count-chip";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { CycleQuickActions } from "@/components/cycles/quick-actions";
import { HeaderFilters } from "@/components/issues/filters";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useCycles } from "@/hooks/store/use-cycles";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useCycleListState } from "@/hooks/use-cycle-list-state";
import useLocalStorage from "@/hooks/use-local-storage";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  cycleId: string;
};

/**
 * The header of a cycle's page (P5a spec 2.5), of the route's project and cycle: its crumbs (the project, its cycles,
 * and the cycle, whose crumb switches to another of the project's cycles that are not archived, in nerve's order; an
 * archived cycle is not among them, its project's archive lists it), the toggle of the progress sidebar, and the
 * cycle's quick actions, under the host of its page (the layout's CycleActionsHost). For those who see the page's
 * work items while the project's cycles are on (P5b spec 2.6), their count as nerve gives it, the page's layouts,
 * filters and display, and, for those who may create a work item, a new one of the cycle while it takes them.
 */
export const CycleIssuesHeader = observer(function CycleIssuesHeader(props: TProps) {
  const { workspaceSlug, projectId, cycleId } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { getCycleById, getProjectCycleIds } = useCycles();
  const { getProjectById, loader } = useProject();
  const { issues } = useIssues(EIssuesStoreType.CYCLE);
  const { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { toggleCreateIssueModal } = useCommandPalette();
  const { joinable } = useCycleListState();
  const { isMobile } = usePlatformOS();

  const { setValue, storedValue } = useLocalStorage("cycle_sidebar_collapsed", false);

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => {
    setValue(!isSidebarCollapsed);
  };

  // derived values
  const cycleDetails = getCycleById(cycleId);
  // the page's work items show for the project's members while its cycles are on (the layout root's own condition)
  const listShown =
    getProjectById(projectId)?.cycle_view === true &&
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined;
  const workItemsCount = listShown ? issues.pageTotal({ projectID: projectId, cycleID: cycleId }) : undefined;
  const canCreateWorkItem =
    joinable &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      projectId
    );
  const switcherOptions = (getProjectCycleIds(projectId) ?? []).flatMap((id): ICustomSearchSelectOption[] => {
    const listed = getCycleById(id);
    if (!listed) return [];
    return [
      {
        value: listed.id,
        query: listed.name,
        content: <SwitcherLabel name={listed.name} LabelIcon={CyclesOutline} />,
      },
    ];
  });

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex items-center gap-2">
          <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label={t("cycles")}
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
          {workItemsCount ? (
            <Tooltip
              label={t("cycle.work_items_count", { count: workItemsCount })}
              layout="stacked"
              side="bottom"
              disabled={isMobile}
            >
              <CountChip count={workItemsCount} />
            </Tooltip>
          ) : null}
        </div>
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <div className="hidden items-center gap-2 md:flex">
          {listShown && <HeaderFilters storeType={EIssuesStoreType.CYCLE} entityId={cycleId} />}
          {canCreateWorkItem && (
            <Button variant="primary" size="lg" onClick={() => toggleCreateIssueModal(true, EIssuesStoreType.CYCLE)}>
              {t("issue.add.label")}
            </Button>
          )}
          <IconButton
            variant="tertiary"
            size="lg"
            icon={RightSidePaneOutline}
            aria-label={t("aria_labels.toggle_details")}
            aria-pressed={!isSidebarCollapsed}
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
