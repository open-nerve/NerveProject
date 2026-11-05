/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export enum EUserPermissions {
  ADMIN = 20,
  MEMBER = 15,
  GUEST = 5,
}

export type TUserPermissions = EUserPermissions.ADMIN | EUserPermissions.MEMBER | EUserPermissions.GUEST;

// workspace notifications
export enum ENotificationFilterType {
  CREATED = "created",
  ASSIGNED = "assigned",
  SUBSCRIBED = "subscribed",
}
