/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  ArchiveOutline,
  CyclesOutline,
  IntakeOutline,
  ModuleOutline,
  SettingsOutline,
  WorkItemsOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import type { Cycle, Module } from "@nerve/api-client";
import type { IProjectView } from "@nerve/types";
import { EUserProjectRoles } from "@nerve/types";
// components
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { handlePowerKNavigate } from "@/components/power-k/utils/navigation";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUser } from "@/hooks/store/user";
// lib
import { intakeEntryOn } from "@/lib/intake-entry";
import { planningEntryOn, type PlanningDomain } from "@/lib/planning-switch";

type TPowerKProjectNavigationCommandKeys =
  | "nav_project_work_items"
  | "open_project_cycle"
  | "nav_project_cycles"
  | "open_project_module"
  | "nav_project_modules"
  | "open_project_view"
  | "nav_project_views"
  | "nav_project_intake"
  | "nav_project_archives"
  | "open_project_setting"
  | "nav_project_settings";

export const baseProjectConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug && ctx.params.projectId);

/**
 * Project-level navigation commands - Navigate to the route's project's pages
 */
export const usePowerKProjectNavigationCommandsRecord = (): Record<
  TPowerKProjectNavigationCommandKeys,
  TPowerKCommandConfig
> => {
  // store hooks
  const {
    permission: { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId },
  } = useUser();
  const { getProjectById } = useProject();
  // derived values
  const hasProjectMemberLevelPermissions = (ctx: TPowerKContext) =>
    allowPermissions(
      [EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER],
      EUserPermissionsLevel.PROJECT,
      ctx.params.workspaceSlug,
      ctx.params.projectId
    );
  const getContextProject = (ctx: TPowerKContext) => getProjectById(ctx.params.projectId);
  /** The caller's role in the address's project, by its workspace and project (never the router's). */
  const getContextRole = (ctx: TPowerKContext) => {
    const { workspaceSlug, projectId } = ctx.params;
    return workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  };
  // the ways into the project's cycles and modules: the planning entry rule, by the caller's role in the address's
  // project (P7c spec 2.2, R2)
  const hasPlanningEntry = (ctx: TPowerKContext, domain: PlanningDomain) =>
    planningEntryOn(getContextProject(ctx), getContextRole(ctx), domain);
  // the way into the project's intake: the intake entry rule, as the sidebar and the tab bar offer it (P5a spec R21)
  const hasIntakeEntry = (ctx: TPowerKContext) => intakeEntryOn(getContextProject(ctx), getContextRole(ctx));

  return {
    nav_project_work_items: {
      id: "nav_project_work_items",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_work_items",
      icon: WorkItemsOutline,
      keySequence: "gi",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "issues"]),
      isEnabled: (ctx) => baseProjectConditions(ctx),
      isVisible: (ctx) => baseProjectConditions(ctx),
      closeOnSelect: true,
    },
    open_project_cycle: {
      id: "open_project_cycle",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_project_cycle",
      icon: CyclesOutline,
      keySequence: "oc",
      page: "open-project-cycle",
      onSelect: (data, ctx) => {
        const cycleDetails = data as Cycle;
        handlePowerKNavigate(ctx, [
          ctx.params.workspaceSlug,
          "projects",
          ctx.params.projectId,
          "cycles",
          cycleDetails.id,
        ]);
      },
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "cycles"),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "cycles"),
      closeOnSelect: true,
    },
    nav_project_cycles: {
      id: "nav_project_cycles",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_cycles",
      icon: CyclesOutline,
      keySequence: "gc",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "cycles"]),
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "cycles"),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "cycles"),
      closeOnSelect: true,
    },
    open_project_module: {
      id: "open_project_module",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_project_module",
      icon: ModuleOutline,
      keySequence: "om",
      page: "open-project-module",
      onSelect: (data, ctx) => {
        const moduleDetails = data as Module;
        handlePowerKNavigate(ctx, [
          ctx.params.workspaceSlug,
          "projects",
          ctx.params.projectId,
          "modules",
          moduleDetails.id,
        ]);
      },
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "modules"),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "modules"),
      closeOnSelect: true,
    },
    nav_project_modules: {
      id: "nav_project_modules",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_modules",
      icon: ModuleOutline,
      keySequence: "gm",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "modules"]),
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "modules"),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasPlanningEntry(ctx, "modules"),
      closeOnSelect: true,
    },
    open_project_view: {
      id: "open_project_view",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_project_view",
      icon: WorkItemsOutline,
      keySequence: "ov",
      page: "open-project-view",
      onSelect: (data, ctx) => {
        const viewDetails = data as IProjectView;
        handlePowerKNavigate(ctx, [
          ctx.params.workspaceSlug,
          "projects",
          ctx.params.projectId,
          "views",
          viewDetails.id,
        ]);
      },
      isEnabled: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.issue_views_view,
      isVisible: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.issue_views_view,
      closeOnSelect: true,
    },
    nav_project_views: {
      id: "nav_project_views",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_views",
      icon: WorkItemsOutline,
      keySequence: "gv",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "views"]),
      isEnabled: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.issue_views_view,
      isVisible: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.issue_views_view,
      closeOnSelect: true,
    },
    nav_project_intake: {
      id: "nav_project_intake",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_intake",
      icon: IntakeOutline,
      keySequence: "gk",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "intake"]),
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasIntakeEntry(ctx),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasIntakeEntry(ctx),
      closeOnSelect: true,
    },
    nav_project_archives: {
      id: "nav_project_archives",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_archives",
      icon: ArchiveOutline,
      keySequence: "gr",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", ctx.params.projectId, "archives", "issues"]),
      isEnabled: (ctx) => baseProjectConditions(ctx) && hasProjectMemberLevelPermissions(ctx),
      isVisible: (ctx) => baseProjectConditions(ctx) && hasProjectMemberLevelPermissions(ctx),
      closeOnSelect: true,
    },
    open_project_setting: {
      id: "open_project_setting",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_project_setting",
      icon: SettingsOutline,
      keySequence: "os",
      page: "open-project-setting",
      onSelect: (data, ctx) => {
        const settingsHref = data as string;
        handlePowerKNavigate(ctx, [
          ctx.params.workspaceSlug,
          "settings",
          "projects",
          ctx.params.projectId,
          settingsHref,
        ]);
      },
      isEnabled: (ctx) => baseProjectConditions(ctx),
      isVisible: (ctx) => baseProjectConditions(ctx),
      closeOnSelect: true,
    },
    nav_project_settings: {
      id: "nav_project_settings",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_project_settings",
      icon: SettingsOutline,
      keySequence: "gs",
      action: (ctx) =>
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "settings", "projects", ctx.params.projectId]),
      isEnabled: (ctx) => baseProjectConditions(ctx),
      isVisible: (ctx) => baseProjectConditions(ctx),
      closeOnSelect: true,
    },
  };
};
