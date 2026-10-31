/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TProfileViews } from "@nerve/types";

/** Narrow the route once; object prototype names are not profile relations. */
export function parseProfileView(value: string | undefined): TProfileViews | undefined {
  return value === "assigned" || value === "created" || value === "subscribed" ? value : undefined;
}
