/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export enum EViewAccess {
  PRIVATE,
  PUBLIC,
}

export type TViewFiltersSortKey = "name" | "created_at" | "updated_at";

export type TViewFiltersSortBy = "asc" | "desc";
