/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ViewContainer } from "@/store/views/view-container";

/**
 * The address of the container's views list (M7/P6b spec 2.11), as the app's routes have it (app/routes/core.ts): a
 * project's views, or the workspace's views list, /{slug}/workspace-views; where its page goes once it has deleted the
 * view it shows.
 */
export const viewsPath = (workspaceSlug: string, container: ViewContainer): string =>
  container.kind === "project"
    ? `/${workspaceSlug}/projects/${container.projectId}/views`
    : `/${workspaceSlug}/workspace-views`;

/** The address of the view's own page, under its container's views list: its row's link, its copy, a new view's. */
export const viewPath = (workspaceSlug: string, container: ViewContainer, viewId: string): string =>
  `${viewsPath(workspaceSlug, container)}/${viewId}`;

/**
 * The route's param that names the view a page of the container's views shows (app/routes/core.ts; re-97 I5): a
 * project's view page's viewId, the workspace's view page's globalViewId (a view's id, or a default view's key, which
 * is never one).
 */
export const pageViewParam = (container: ViewContainer): "viewId" | "globalViewId" =>
  container.kind === "project" ? "viewId" : "globalViewId";
