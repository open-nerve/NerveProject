/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  ActivityOutline,
  ArchiveOutline,
  EditOutline,
  HomeOutline,
  InboxOutline,
  ProjectsOutline,
  SettingsOutline,
  WorkItemsOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { EUserPermissionsLevel } from "@nerve/constants";
import type { Workspace, Project } from "@nerve/api-client";
import { EUserWorkspaceRoles } from "@nerve/types";
// components
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { handlePowerKNavigate } from "@/components/power-k/utils/navigation";
// hooks
import { useUser } from "@/hooks/store/user";
// local imports
import { baseProjectConditions } from "./project-commands";

type TPowerKWorkspaceNavigationCommandKeys =
  | "open_workspace"
  | "nav_home"
  | "nav_inbox"
  | "nav_your_work"
  | "nav_account_settings"
  | "open_project"
  | "nav_projects_list"
  | "nav_all_workspace_work_items"
  | "nav_assigned_workspace_work_items"
  | "nav_created_workspace_work_items"
  | "nav_subscribed_workspace_work_items"
  | "nav_workspace_drafts"
  | "nav_workspace_archives"
  | "open_workspace_setting"
  | "nav_workspace_settings";

const baseWorkspaceConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug);

/**
 * Workspace-level navigation commands - Navigate to the workspace's and the account's pages
 */
export const usePowerKWorkspaceNavigationCommandsRecord = (): Record<
  TPowerKWorkspaceNavigationCommandKeys,
  TPowerKCommandConfig
> => {
  // store hooks
  const {
    data: currentUser,
    permission: { allowPermissions },
  } = useUser();
  // derived values
  const hasWorkspaceMemberLevelPermissions = (ctx: TPowerKContext) =>
    allowPermissions(
      [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
      EUserPermissionsLevel.WORKSPACE,
      ctx.params.workspaceSlug
    );

  return {
    open_workspace: {
      id: "open_workspace",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_workspace",
      icon: ProjectsOutline,
      keySequence: "ow",
      page: "open-workspace",
      onSelect: (data, ctx) => {
        const workspaceDetails = data as Workspace;
        handlePowerKNavigate(ctx, [workspaceDetails.slug]);
      },
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_home: {
      id: "nav_home",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_home",
      icon: HomeOutline,
      keySequence: "gh",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_inbox: {
      id: "nav_inbox",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_inbox",
      icon: InboxOutline,
      keySequence: "gx",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "notifications"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_your_work: {
      id: "nav_your_work",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_your_work",
      icon: ActivityOutline,
      keySequence: "gy",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "profile", currentUser?.id]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      closeOnSelect: true,
    },
    nav_account_settings: {
      id: "nav_account_settings",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_account_settings",
      icon: SettingsOutline,
      action: (ctx) => handlePowerKNavigate(ctx, ["settings", "profile", "general"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    open_project: {
      id: "open_project",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_project",
      icon: ProjectsOutline,
      keySequence: "op",
      page: "open-project",
      onSelect: (data, ctx) => {
        const projectDetails = data as Project;
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", projectDetails.id, "issues"]);
      },
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_projects_list: {
      id: "nav_projects_list",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_projects_list",
      icon: ProjectsOutline,
      keySequence: "gp",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_all_workspace_work_items: {
      id: "nav_all_workspace_work_items",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_all_workspace_work_items",
      icon: WorkItemsOutline,
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "workspace-views", "all-issues"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_assigned_workspace_work_items: {
      id: "nav_assigned_workspace_work_items",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_assigned_workspace_work_items",
      icon: WorkItemsOutline,
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "workspace-views", "assigned"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_created_workspace_work_items: {
      id: "nav_created_workspace_work_items",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_created_workspace_work_items",
      icon: WorkItemsOutline,
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "workspace-views", "created"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_subscribed_workspace_work_items: {
      id: "nav_subscribed_workspace_work_items",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_subscribed_workspace_work_items",
      icon: WorkItemsOutline,
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "workspace-views", "subscribed"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx),
      closeOnSelect: true,
    },
    nav_workspace_drafts: {
      id: "nav_workspace_drafts",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_workspace_drafts",
      icon: EditOutline,
      keySequence: "gj",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "drafts"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      closeOnSelect: true,
    },
    nav_workspace_archives: {
      id: "nav_workspace_archives",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_workspace_archives",
      icon: ArchiveOutline,
      keySequence: "gr",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "projects", "archives"]),
      isEnabled: (ctx) =>
        baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      isVisible: (ctx) =>
        baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx) && hasWorkspaceMemberLevelPermissions(ctx),
      closeOnSelect: true,
    },
    open_workspace_setting: {
      id: "open_workspace_setting",
      type: "change-page",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.open_workspace_setting",
      icon: SettingsOutline,
      keySequence: "os",
      page: "open-workspace-setting",
      onSelect: (data, ctx) => {
        const settingsHref = data as string;
        handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, settingsHref]);
      },
      isEnabled: (ctx) => baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx),
      closeOnSelect: true,
    },
    nav_workspace_settings: {
      id: "nav_workspace_settings",
      type: "action",
      group: "navigation",
      i18n_title: "power_k.navigation_actions.nav_workspace_settings",
      icon: SettingsOutline,
      keySequence: "gs",
      action: (ctx) => handlePowerKNavigate(ctx, [ctx.params.workspaceSlug, "settings"]),
      isEnabled: (ctx) => baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx),
      isVisible: (ctx) => baseWorkspaceConditions(ctx) && !baseProjectConditions(ctx),
      closeOnSelect: true,
    },
  };
};
