/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { createRoutesStub } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { IProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import AutomationsPage from "./automations/page";
import CyclesPage from "./features/cycles/page";
import IntakePage from "./features/intake/page";
import ModulesPage from "./features/modules/page";
import ViewsPage from "./features/views/page";

// Whom a project's features and automations pages show their controls (M3 design 3.4, 9.2): those nerve lets change
// the project (project.update: its admins, and its members who are the workspace's admins, its guests among them); the
// others are told they may not. Each page renders on the server at its own route, through React Router's stub, which
// gives it the route's params, with stand-ins for its parts that write what they show; the caller's permissions and
// projects are the stores' own, over acme and its project web as nerve lists them to him (fake-permissions.ts), while
// the address the router store holds names no project: a page reads its project from its route.

const caller = vi.hoisted((): { permissions?: IUserPermissionStore; projects?: IProjectStore } => ({}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => caller.projects }));
vi.mock("@/components/auth-screens/not-authorized-view", () => ({ NotAuthorizedView: () => "not authorized" }));
vi.mock("@/components/core/page-title", () => ({ PageHead: ({ title }: { title?: string }) => `[${title}]` }));
vi.mock("@/components/settings/content-wrapper", () => ({
  SettingsContentWrapper: ({ children }: { children: unknown }) => children,
}));
vi.mock("@/components/settings/heading", () => ({ SettingsHeading: () => null }));
vi.mock("@/components/settings/project/content/feature-control-item", () => ({
  ProjectSettingsFeatureControlItem: ({ value }: { value: boolean }) => `switch ${value}`,
}));
vi.mock("@/components/automation", () => ({ AutoArchiveAutomation: () => "auto-archive" }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** A page, at its route under the project's settings, and what it shows those it lets change web. */
type Page = { page: string; route: string; Component: StubRoute["Component"]; shows: (web: Project) => string };
/** A route of React Router's stub, whose component gets the route's props as a route module's does. */
type StubRoute = Parameters<typeof createRoutesStub>[0][number];

const pages: Page[] = [
  {
    page: "the cycles feature",
    route: "features/cycles",
    Component: CyclesPage,
    shows: (web) => `[${web.name} settings - project_settings.features.cycles.short_title]switch ${web.cycle_view}`,
  },
  {
    page: "the intake feature",
    route: "features/intake",
    Component: IntakePage,
    shows: (web) => `[${web.name} settings - project_settings.features.intake.short_title]switch ${web.intake_view}`,
  },
  {
    page: "the modules feature",
    route: "features/modules",
    Component: ModulesPage,
    shows: (web) => `[${web.name} settings - project_settings.features.modules.short_title]switch ${web.module_view}`,
  },
  {
    page: "the views feature",
    route: "features/views",
    Component: ViewsPage,
    shows: (web) =>
      `[${web.name} settings - project_settings.features.views.short_title]switch ${web.issue_views_view}`,
  },
  {
    page: "the automations",
    route: "automations",
    Component: AutomationsPage,
    shows: (web) => `[${web.name} - Automations]auto-archive`,
  },
];

/**
 * Signs the caller in as nerve lists acme and web to him (callerInWeb), moves the router store's address to acme with
 * no project, and opens the page at web's route: the text it shows him, and web.
 */
async function openedBy({ workspaceRole, projectRole }: Caller, { route, Component }: Page) {
  const { permissions, router, projects, acme, web } = await callerInWeb(workspaceRole, projectRole);
  Object.assign(caller, { permissions, projects });
  router.setQuery({ workspaceSlug: acme.slug });
  const Routes = createRoutesStub([{ path: `/:workspaceSlug/settings/projects/:projectId/${route}`, Component }]);
  const markup = renderToStaticMarkup(
    <Routes initialEntries={[`/${acme.slug}/settings/projects/${web.id}/${route}`]} />
  );
  return { text: markup.replace(/<[^>]*>/g, ""), web };
}

describe.each(pages)("$page page", (page) => {
  it.each([its.admin, its.memberAdmin, its.guestAdmin])("shows $who its controls", async (row) => {
    const { text, web } = await openedBy(row, page);
    expect(text).toBe(page.shows(web));
  });

  it.each([its.member, its.guest])("tells $who he may not change it", async (row) => {
    const { text } = await openedBy(row, page);
    expect(text).toBe("not authorized");
  });
});
