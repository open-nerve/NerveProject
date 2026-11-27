/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { OrganizationSize } from "@nerve/api-client";
import { EUserWorkspaceRoles } from "@nerve/types";

export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];

/** The key of each organization size's label in the page's language: nerve keeps the size itself. */
export const ORGANIZATION_SIZE_I18N_LABEL: Record<OrganizationSize, string> = {
  "Just myself": "workspace_creation.form.organization_size.sizes.just_myself",
  "2-10": "workspace_creation.form.organization_size.sizes.2_10",
  "11-50": "workspace_creation.form.organization_size.sizes.11_50",
  "51-200": "workspace_creation.form.organization_size.sizes.51_200",
  "201-500": "workspace_creation.form.organization_size.sizes.201_500",
  "500+": "workspace_creation.form.organization_size.sizes.500_plus",
};

export const ROLE_DETAILS = {
  [EUserWorkspaceRoles.GUEST]: {
    i18n_title: "role_details.guest.title",
  },
  [EUserWorkspaceRoles.MEMBER]: {
    i18n_title: "role_details.member.title",
  },
  [EUserWorkspaceRoles.ADMIN]: {
    i18n_title: "role_details.admin.title",
  },
};

/** The workspace's default views, by the key its address names (M7/P6b spec 2.5). */
export type DefaultWorkspaceViewKey = "all-issues" | "assigned" | "created" | "subscribed";

/**
 * The workspace's four default views (M7/P6b spec 2.5, R4): each one's own term, `me` in it, is fixed in its query and
 * left out of the words its page filters by, so that the page never sends that key twice (nerve refuses it) and no
 * filter of the page replaces it.
 */
export const DEFAULT_GLOBAL_VIEWS_LIST: {
  key: DefaultWorkspaceViewKey;
  i18n_label: string;
  own?: "assignee_id" | "created_by_id" | "subscriber_id";
}[] = [
  {
    key: "all-issues",
    i18n_label: "default_global_view.all_issues",
  },
  {
    key: "assigned",
    i18n_label: "default_global_view.assigned",
    own: "assignee_id",
  },
  {
    key: "created",
    i18n_label: "default_global_view.created",
    own: "created_by_id",
  },
  {
    key: "subscribed",
    i18n_label: "default_global_view.subscribed",
    own: "subscriber_id",
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
