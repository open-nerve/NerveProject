/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useMemo } from "react";
import { observer } from "mobx-react";
import { matchPath, useParams, Link, useLocation } from "react-router";
import { EUserPermissionsLevel, EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CyclesOutline, IntakeOutline, ModuleOutline, ViewsOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import type { EUserProjectRoles } from "@nerve/types";
// components
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useIntake } from "@/hooks/store/use-intake";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { intakeEntryOn } from "@/lib/intake-entry";

export type TNavigationItem = {
  name: string;
  href: string;
  icon: React.ElementType;
  access: EUserPermissions[] | EUserProjectRoles[];
  shouldRender: boolean;
  sortOrder: number;
  i18n_key: string;
  key: string;
};

type TProjectItemsProps = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectNavigation = observer(function ProjectNavigation(props: TProjectItemsProps) {
  const { workspaceSlug, projectId } = props;
  const { workItem: workItemIdentifierFromRoute } = useParams();
  // store hooks
  const { t } = useTranslation();
  const { isExtendedProjectSidebarOpened, toggleExtendedProjectSidebar, toggleSidebar } = useAppTheme();
  const { getProjectById } = useProject();
  const { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { pendingCount } = useIntake();
  const {
    issue: { getIssueIdByIdentifier, getIssueById },
  } = useIssueDetail();
  // pathname
  const { pathname } = useLocation();
  // derived values
  const workItemId = workItemIdentifierFromRoute ? getIssueIdByIdentifier(workItemIdentifierFromRoute) : undefined;
  const workItem = workItemId ? getIssueById(workItemId) : undefined;
  const project = getProjectById(projectId);
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  // handlers
  const handleProjectClick = () => {
    if (window.innerWidth < 768) {
      toggleSidebar();
    }
    // close the extended sidebar if it is open
    if (isExtendedProjectSidebarOpened) {
      toggleExtendedProjectSidebar(false);
    }
  };

  const baseNavigation = useCallback(
    (): TNavigationItem[] => [
      {
        i18n_key: "sidebar.work_items",
        key: "work_items",
        name: "Work items",
        href: `/${workspaceSlug}/projects/${projectId}/issues`,
        icon: WorkItemsOutline,
        access: [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST],
        shouldRender: true,
        sortOrder: 1,
      },
      {
        i18n_key: "sidebar.cycles",
        key: "cycles",
        name: "Cycles",
        href: `/${workspaceSlug}/projects/${projectId}/cycles`,
        icon: CyclesOutline,
        access: [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
        shouldRender: project?.cycle_view ?? false,
        sortOrder: 2,
      },
      {
        i18n_key: "sidebar.modules",
        key: "modules",
        name: "Modules",
        href: `/${workspaceSlug}/projects/${projectId}/modules`,
        icon: ModuleOutline,
        access: [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
        shouldRender: project?.module_view ?? false,
        sortOrder: 3,
      },
      {
        i18n_key: "sidebar.views",
        key: "views",
        name: "Views",
        href: `/${workspaceSlug}/projects/${projectId}/views`,
        icon: ViewsOutline,
        access: [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST],
        shouldRender: project?.issue_views_view ?? false,
        sortOrder: 4,
      },
      {
        i18n_key: "sidebar.intake",
        key: "intake",
        name: "Intake",
        href: `/${workspaceSlug}/projects/${projectId}/intake`,
        icon: IntakeOutline,
        access: [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST],
        shouldRender: intakeEntryOn(project, role),
        sortOrder: 6,
      },
    ],
    [project, role, workspaceSlug, projectId]
  );

  // memoized navigation items, sorted by sortOrder
  const navigationItemsMemo = useMemo(
    () => baseNavigation().sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0)),
    [baseNavigation]
  );

  const isActive = useCallback(
    (item: TNavigationItem) => {
      // work item condition
      const workItemCondition = workItemId && workItem && workItem?.project_id === projectId;
      // is active
      const isWorkItemActive = item.key === "work_items" && workItemCondition;
      // pathname condition
      const isPathnameActive = matchPath({ path: item.href, end: false }, pathname) !== null;
      // return
      return isWorkItemActive || isPathnameActive;
    },
    [pathname, workItem, workItemId, projectId]
  );

  if (!project) return null;

  return (
    <>
      {navigationItemsMemo.map((item) => {
        if (!item.shouldRender) return;

        const hasAccess = allowPermissions(item.access, EUserPermissionsLevel.PROJECT, workspaceSlug, project.id);
        if (!hasAccess) return null;

        // the intake's pending work items as nerve counted them, beside its name and in it, while there are some (P5a
        // spec R7): the projects list fetches them (useIntakeCounts)
        const pending = item.key === "intake" ? (pendingCount(workspaceSlug, project.id) ?? 0) : 0;
        const shownPending = pending > 0 ? pending : undefined;

        return (
          <Link
            key={item.key}
            to={item.href}
            onClick={handleProjectClick}
            aria-label={
              shownPending === undefined
                ? undefined
                : t("inbox_issue.a11y.pending_count", { label: t(item.i18n_key), count: shownPending })
            }
          >
            <SidebarNavItem isActive={!!isActive(item)}>
              <div className="flex w-full items-center justify-between gap-1.5 py-[1px]">
                <div className="flex items-center gap-1.5">
                  <item.icon
                    className={`size-4 flex-shrink-0 ${item.name === "Intake" ? "stroke-1" : "stroke-[1.5]"}`}
                  />
                  <span className="text-11 font-medium">{t(item.i18n_key)}</span>
                </div>
                {shownPending !== undefined && (
                  <span className="text-11 font-medium text-tertiary">{shownPending}</span>
                )}
              </div>
            </SidebarNavItem>
          </Link>
        );
      })}
    </>
  );
});
