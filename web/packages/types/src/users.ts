/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TUserPermissions } from "./enums";

/**
 * @description The start of the week for the user
 * @enum {number}
 */
export enum EStartOfTheWeek {
  SUNDAY = 0,
  MONDAY = 1,
  TUESDAY = 2,
  WEDNESDAY = 3,
  THURSDAY = 4,
  FRIDAY = 5,
  SATURDAY = 6,
}

export interface IUserLite {
  avatar_url: string;
  display_name: string;
  email?: string;
  first_name: string;
  id: string;
  is_bot: boolean;
  last_name: string;
  joining_date?: string;
}
export interface IUser extends IUserLite {
  // only for uploading the cover image
  cover_image_asset?: string | null;
  cover_image?: string | null;
  // only for rendering the cover image
  cover_image_url: string | null;
  date_joined: string;
  email: string;
  is_active: boolean;
  is_email_verified: boolean;
  is_tour_completed: boolean;
  mobile_number: string | null;
  last_workspace_id: string;
  user_timezone: string;
  username: string;
  theme: IUserTheme;
}

export type TUserProfile = {
  id: string | undefined;
  user: string | undefined;
  role: string | undefined;
  last_workspace_id: string | undefined;
  theme: IUserTheme;
  onboarding_step: TOnboardingSteps;
  is_onboarded: boolean;
  is_tour_completed: boolean;
  use_case: string | undefined;
  language: string;
  created_at: Date | string;
  updated_at: Date | string;
  start_of_the_week: EStartOfTheWeek;
};

export interface IUserSettings {
  id: string | undefined;
  email: string | undefined;
  workspace: {
    last_workspace_id: string | undefined;
    last_workspace_slug: string | undefined;
    last_workspace_name: string | undefined;
    last_workspace_logo: string | undefined;
    fallback_workspace_id: string | undefined;
    fallback_workspace_slug: string | undefined;
    invites: number | undefined;
  };
}

export interface IUserTheme {
  theme: string | undefined; // one of THEMES, or 'system'
}

export interface IUserMemberLite extends IUserLite {
  email?: string;
}

export type UserAuth = {
  isMember: boolean;
  isOwner: boolean;
  isGuest: boolean;
};

export type TOnboardingSteps = {
  profile_complete: boolean;
  workspace_create: boolean;
  workspace_invite: boolean;
  workspace_join: boolean;
};

export interface IUserProjectsRole {
  [projectId: string]: TUserPermissions;
}

export type TProfileViews = "assigned" | "created" | "subscribed";

export type TPublicMember = {
  id: string;
  member: string;
  member__display_name: string;
  member__avatar: string;
};
