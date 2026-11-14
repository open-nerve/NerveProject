/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Project } from "@nerve/api-client";

/**
 * The cycle or module a planning page has (P6b spec 5, review re-71 M2): the one the tab holds, while it is of the
 * page's project once the page has that. One of another project is not the page's, as its body and crumb say: the
 * page's list state takes it as not yet there (read-only, joining nothing) and its header shows nothing of it.
 */
export function pageEntity<T extends { project_id: string }>(
  entity: T | undefined,
  project: Project | undefined
): T | undefined {
  return project === undefined || entity?.project_id === project.id ? entity : undefined;
}
