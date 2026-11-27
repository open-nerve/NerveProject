/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * Where a view is made (M7/P6b spec 2.6, R9): in a project, by its id; or in a workspace, one of its own, of no
 * project, by its id, which holds it, and its slug, which nerve's address names it by.
 */
export type ViewContainer =
  | { kind: "project"; projectId: string }
  | { kind: "workspace"; workspaceId: string; workspaceSlug: string };

/**
 * The key the views' store holds the container's views by: its project's id, or its workspace's; a view's own is its
 * project_id, else its workspace_id.
 */
export const containerKey = (container: ViewContainer): string =>
  container.kind === "project" ? container.projectId : container.workspaceId;
