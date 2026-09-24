/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TWorkspaceSettingsItem, TWorkspaceSettingsTabs } from "@nerve/types";
import { EUserWorkspaceRoles } from "@nerve/types";

export enum WORKSPACE_SETTINGS_CATEGORY {
  ADMINISTRATION = "administration",
  DEVELOPER = "developer",
}

export const WORKSPACE_SETTINGS_CATEGORIES: WORKSPACE_SETTINGS_CATEGORY[] = [
  WORKSPACE_SETTINGS_CATEGORY.ADMINISTRATION,
  WORKSPACE_SETTINGS_CATEGORY.DEVELOPER,
];

export const WORKSPACE_SETTINGS_CATEGORY_LABELS: Record<WORKSPACE_SETTINGS_CATEGORY, string> = {
  [WORKSPACE_SETTINGS_CATEGORY.ADMINISTRATION]: "common.administration",
  [WORKSPACE_SETTINGS_CATEGORY.DEVELOPER]: "common.developer",
};

export const WORKSPACE_SETTINGS: Record<TWorkspaceSettingsTabs, TWorkspaceSettingsItem> = {
  general: {
    key: "general",
    i18n_label: "workspace_settings.settings.general.title",
    href: `/settings`,
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
  },
  members: {
    key: "members",
    i18n_label: "workspace_settings.settings.members.title",
    href: `/settings/members`,
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
  },
  webhooks: {
    key: "webhooks",
    i18n_label: "workspace_settings.settings.webhooks.title",
    href: `/settings/webhooks`,
    access: [EUserWorkspaceRoles.ADMIN],
  },
};

export const WORKSPACE_SETTINGS_ACCESS = Object.fromEntries(
  Object.entries(WORKSPACE_SETTINGS).map(([_, { href, access }]) => [href, access])
);

export const GROUPED_WORKSPACE_SETTINGS: Record<WORKSPACE_SETTINGS_CATEGORY, TWorkspaceSettingsItem[]> = {
  [WORKSPACE_SETTINGS_CATEGORY.ADMINISTRATION]: [WORKSPACE_SETTINGS["general"], WORKSPACE_SETTINGS["members"]],
  [WORKSPACE_SETTINGS_CATEGORY.DEVELOPER]: [WORKSPACE_SETTINGS["webhooks"]],
};
