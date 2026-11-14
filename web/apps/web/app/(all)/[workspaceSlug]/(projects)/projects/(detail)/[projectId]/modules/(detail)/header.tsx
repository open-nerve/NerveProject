/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// icons
import { ModuleOutline, RightSidePaneOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
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
import { HeaderFilters } from "@/components/issues/filters";
import { usePlanningPageHeader } from "@/components/issues/issue-layouts/use-planning-page-header";
import { ModuleQuickActions } from "@/components/modules/quick-actions";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";
import useLocalStorage from "@/hooks/use-local-storage";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TProps = {
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
};

/**
 * The header of a module's page (P6a spec 2.6), of the route's project and module: its crumbs (the project, its
 * modules, and the module, whose crumb switches to another of the project's modules that are not archived, in nerve's
 * order; an archived module is not among them, its project's archive lists it), the toggle of the progress sidebar, and
 * the module's quick actions, under the host of its page (the layout's ModuleActionsHost). For those who see the page's
 * work items while the project's modules are on (P6b spec 2.7), their count as nerve gives it; the page's layouts,
 * filters and display (never by module, R12) once nerve has given the page's preferences, which they change: not before
 * its first answer, nor when that first read failed (R3); and, for the project's admins and members, a new work item of
 * the module while it takes them (R6), whose form keeps the module's project. The list's controls and the new work item
 * show from a medium width, as the cycle's page's (its mobile header comes with P7).
 */
export const ModuleIssuesHeader = observer(function ModuleIssuesHeader(props: TProps) {
  const { workspaceSlug, projectId, moduleId } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // router
  const navigate = useNavigate();
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { getModuleById, getProjectModuleIds } = useModules();
  const { loader } = useProject();
  const { toggleCreateIssueModal } = useCommandPalette();
  const { isMobile } = usePlatformOS();
  const { setValue, storedValue } = useLocalStorage("module_sidebar_collapsed", false);

  const isSidebarCollapsed = storedValue === true;
  const toggleSidebar = () => {
    setValue(!isSidebarCollapsed);
  };

  // derived values: the route's project's module alone
  const held = getModuleById(moduleId);
  const moduleDetails = held?.project_id === projectId ? held : undefined;
  const {
    count: workItemsCount,
    preferencesShown,
    canCreate: canCreateWorkItem,
  } = usePlanningPageHeader(EIssuesStoreType.MODULE, { workspaceSlug, projectId, entityId: moduleId });
  const switcherOptions = (getProjectModuleIds(projectId) ?? []).flatMap((id): ICustomSearchSelectOption[] => {
    const listed = getModuleById(id);
    if (!listed) return [];
    return [
      {
        value: listed.id,
        query: listed.name,
        content: <SwitcherLabel name={listed.name} LabelIcon={ModuleOutline} />,
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
                  label={t("modules")}
                  href={`/${workspaceSlug}/projects/${projectId}/modules`}
                  icon={<ModuleOutline className="h-4 w-4 text-tertiary" />}
                />
              }
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
              isLast
            />
          </Breadcrumbs>
          {workItemsCount ? (
            <Tooltip
              label={t("module.work_items_count", { count: workItemsCount })}
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
          {preferencesShown && <HeaderFilters storeType={EIssuesStoreType.MODULE} entityId={moduleId} />}
          {canCreateWorkItem && (
            <Button variant="primary" size="lg" onClick={() => toggleCreateIssueModal(true, EIssuesStoreType.MODULE)}>
              {t("issue.add.label")}
            </Button>
          )}
        </div>
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
        {moduleDetails && (
          <ModuleQuickActions
            parentRef={parentRef}
            moduleId={moduleId}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
            customClassName="flex-shrink-0 flex items-center justify-center bg-layer-1/70 rounded-sm size-[26px]"
          />
        )}
      </Header.RightItem>
    </Header>
  );
});
