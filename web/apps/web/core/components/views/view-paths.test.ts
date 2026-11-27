/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import type { ViewContainer } from "@/store/views/view-container";
import { pageViewParam, viewPath, viewsPath } from "./view-paths";

// The addresses of the views by their container (M7/P6b spec 2.11; re-97 I5), as the app's routes have them
// (app/routes/core.ts): a project's views and a view's own page under the project; a workspace's own under its views
// list, /{slug}/workspace-views; and the route's param that names the view a page shows, viewId under a project,
// globalViewId in the workspace.

const inWeb: ViewContainer = { kind: "project", projectId: "p-web" };
const inAcme: ViewContainer = { kind: "workspace", workspaceId: "id-acme", workspaceSlug: "acme" };

it.each([
  ["a project", inWeb, "/acme/projects/p-web/views", "/acme/projects/p-web/views/v-1", "viewId"],
  ["a workspace", inAcme, "/acme/workspace-views", "/acme/workspace-views/v-1", "globalViewId"],
] as const)(
  "addresses %s's views, a view's own page, and the param its page names the view by",
  (_, container, views, view, param) => {
    expect([viewsPath("acme", container), viewPath("acme", container, "v-1"), pageViewParam(container)]).toEqual([
      views,
      view,
      param,
    ]);
  }
);
