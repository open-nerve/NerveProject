/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { OrganizationSize } from "@nerve/api-client";
import type { TStaticViewTypes, IWorkspaceSearchResults } from "@nerve/types";
import { EUserWorkspaceRoles } from "@nerve/types";

export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];

export const ROLE = {
  [EUserWorkspaceRoles.GUEST]: "Guest",
  [EUserWorkspaceRoles.MEMBER]: "Member",
  [EUserWorkspaceRoles.ADMIN]: "Admin",
};

export const ROLE_DETAILS = {
  [EUserWorkspaceRoles.GUEST]: {
    i18n_title: "role_details.guest.title",
    i18n_description: "role_details.guest.description",
  },
  [EUserWorkspaceRoles.MEMBER]: {
    i18n_title: "role_details.member.title",
    i18n_description: "role_details.member.description",
  },
  [EUserWorkspaceRoles.ADMIN]: {
    i18n_title: "role_details.admin.title",
    i18n_description: "role_details.admin.description",
  },
};

export const DEFAULT_GLOBAL_VIEWS_LIST: {
  key: TStaticViewTypes;
  i18n_label: string;
}[] = [
  {
    key: "all-issues",
    i18n_label: "default_global_view.all_issues",
  },
  {
    key: "assigned",
    i18n_label: "default_global_view.assigned",
  },
  {
    key: "created",
    i18n_label: "default_global_view.created",
  },
  {
    key: "subscribed",
    i18n_label: "default_global_view.subscribed",
  },
];

export interface IWorkspaceSidebarNavigationItem {
  key: string;
  labelTranslationKey: string;
  href: string;
  access: EUserWorkspaceRoles[];
  /** Current only on this exact address, not below it. */
  end?: boolean;
}

export const WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS: IWorkspaceSidebarNavigationItem[] = [
  {
    key: "home",
    labelTranslationKey: "home.title",
    href: "",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER, EUserWorkspaceRoles.GUEST],
    end: true,
  },
  {
    key: "your_work",
    labelTranslationKey: "your_work",
    href: "/profile",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
  },
  {
    key: "drafts",
    labelTranslationKey: "drafts",
    href: "/drafts",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
  },
];

export const WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS: IWorkspaceSidebarNavigationItem[] = [
  {
    key: "projects",
    labelTranslationKey: "projects",
    href: "/projects",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER, EUserWorkspaceRoles.GUEST],
    end: true,
  },
  {
    key: "views",
    labelTranslationKey: "views",
    href: "/workspace-views/all-issues",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER, EUserWorkspaceRoles.GUEST],
  },
  {
    key: "archives",
    labelTranslationKey: "archives",
    href: "/projects/archives",
    access: [EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER],
  },
];

export const IS_FAVORITE_MENU_OPEN = "is_favorite_menu_open";
export const WORKSPACE_DEFAULT_SEARCH_RESULT: IWorkspaceSearchResults = {
  results: {
    workspace: [],
    project: [],
    issue: [],
    cycle: [],
    module: [],
    issue_view: [],
  },
};
