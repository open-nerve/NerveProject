// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentType } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project, ProjectRole } from "@nerve/api-client";
import { EViewAccess, type IProjectView } from "@nerve/types";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import { InstanceStore } from "@/store/instance.store";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RootStore } from "@/store/root.store";
import { RouterStore } from "@/store/router.store";
import { ThemeStore } from "@/store/theme.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import Page from "./page";

// A project's views list page at the route's address (design §8, M3 handoff §4, P7c spec §2.9): whether the project's
// views are turned off, and what the caller may do on the page and its rows, are the address's project's, never those
// of the project the router store still holds (it follows the address only after the render, M3/P10 F-2). The stores
// are a RootStore's over nerve's lists, the caller acme's member and each project's member as its member_role says;
// the views' store holds what a test puts in it (nothing reads views from nerve yet, M7).

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// a view's star on its row (favorites are M7's), by a name the test reads
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  FavoriteStar: () => <button type="button">the view's star</button>,
}));

/** A route module's component, given its route's params as the router gives them. */
type RouteProps = { params: Readonly<Record<string, string | undefined>> };
function RouteOf({ Component }: { Component: ComponentType<RouteProps> }) {
  const params = useParams();
  return <Component params={params} />;
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>, store: RootStore | undefined;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  store?.dispose();
  store = undefined;
  vi.unstubAllGlobals();
});

const acme = workspaceOf("acme");
const button = (name: string) => [...host.querySelectorAll("button")].find((one) => one.textContent === name);

/** A view of web's, named name. */
function viewOf(name: string): IProjectView {
  return {
    id: "v-release",
    name,
    access: EViewAccess.PUBLIC,
    created_at: new Date(),
    updated_at: new Date(),
    is_favorite: false,
    created_by: "u",
    updated_by: "u",
    description: "",
    rich_filters: {},
    display_filters: {},
    display_properties: {},
    query: {},
    query_data: {},
    project: "p-web",
    workspace: acme.id,
    logo_props: undefined,
    is_locked: false,
    owned_by: "u",
  };
}

/**
 * The stores over acme and its projects web and ops as nerve lists them, the router store holding ops (the project
 * before web, say: it follows the address only after the render).
 */
async function storesAtOps(web: Project, ops: Project) {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  const stores = new RootStore(nerve.client(), { router, instance: new InstanceStore(), theme: new ThemeStore() });
  vi.useFakeTimers();
  await loadWorkspaces(nerve, stores.workspaceRoot, [acme]);
  await loadProjects(nerve, stores.projectRoot.project, acme, [web, ops]);
  vi.useRealTimers();
  router.setQuery({ workspaceSlug: acme.slug, projectId: ops.id });
  return stores;
}

/**
 * The caller, of role inWeb in web and inOps in ops (null: no member of it), at web's views list, web being as fields
 * say, while the router store still holds ops; web's views listed as views says (none: not read).
 */
async function opened(
  inWeb: ProjectRole | null,
  inOps: ProjectRole | null,
  fields: Partial<Project> = {},
  views?: IProjectView[]
) {
  const held = await storesAtOps(
    projectOf("WEB", acme.id, { member_role: inWeb, ...fields }),
    projectOf("OPS", acme.id, { member_role: inOps })
  );
  store = held;
  if (views)
    runInAction(() => {
      held.projectView.fetchedMap["p-web"] = true;
      views.forEach((view) => (held.projectView.viewMap[view.id] = view));
    });
  await act(async () =>
    root.render(
      <StoreContext.Provider value={held}>
        <MemoryRouter initialEntries={["/acme/projects/p-web/views"]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/views"
              element={<RouteOf Component={Page as unknown as ComponentType<RouteProps>} />}
            />
          </Routes>
        </MemoryRouter>
      </StoreContext.Provider>
    )
  );
}

it.each([
  { who: "an admin of web, member of ops", inWeb: 20, inOps: 15, may: true },
  { who: "a member of web, admin of ops", inWeb: 15, inOps: 20, may: false },
] as const)(
  "shows $who web's views turned off, ops' being on, whose features web's admin alone manages",
  async ({ inWeb, inOps, may }) => {
    await opened(inWeb, inOps, { issue_views_view: false });
    expect(host.textContent).toContain("disabled_project.empty_state.view.title");
    expect(button("disabled_project.empty_state.view.primary_button.text")?.disabled).toBe(!may);
  }
);

it.each([
  { who: "a guest of web, no member of ops", inWeb: 5, inOps: null, may: true },
  { who: "no member of web, an admin of ops", inWeb: null, inOps: 20, may: false },
] as const)("offers $who web's first view as it does web's members", async ({ inWeb, inOps, may }) => {
  await opened(inWeb, inOps, {}, []);
  expect(host.textContent).toContain("project_empty_state.views.title");
  expect(button("project_empty_state.views.cta_primary")?.disabled).toBe(!may);
});

it.each([
  { who: "a member of web, guest of ops", inWeb: 15, inOps: 5, may: true },
  { who: "a guest of web, admin of ops", inWeb: 5, inOps: 20, may: false },
] as const)("gives $who the star of web's view as web's admins and members have it", async ({ inWeb, inOps, may }) => {
  await opened(inWeb, inOps, {}, [viewOf("Release view")]);
  expect(host.textContent).toContain("Release view");
  expect(button("the view's star") !== undefined).toBe(may);
});
