// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentType } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation, useParams, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import { listedModules } from "@/components/modules/fake-module-lists";
import { membersAtAnotherRouter } from "@/components/modules/fake-module-members";
import { useModuleDisplay } from "@/components/modules/use-module-display";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { ModuleFilterStore } from "@/store/planning/module_filter.store";
import { moduleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "./layout";
import Page from "./page";

// A project's archive of modules (P6a spec 2.4, 2.6, R3, R4, R12, R14): nerve's archived modules of the project, read
// for any of its members while the project is not archived (usePlanningReads' readable: whether its modules are turned
// on or not); a wait for them, why nerve could not read them with a retry (never a wait that does not end), or that
// there are none yet. Searched and filtered by the archive's own search and filters (its lead and members, the members
// of the address's workspace, and its days; no status), ordered as the project's list is (the browser's one order of
// the project, useModuleDisplay); each row opens the module's preview, and its quick actions restore it. The caller's
// role is the permission store's own; the modules a stand-in store's, the filters a session's, the members the member
// store's own, whose router names another workspace; the archives' header, the avatars and the preview's sidebar
// stand-ins.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("@/components/modules/fake-module-lists").listedModules>,
  filters: undefined as unknown as import("@/store/planning/module_filter.store").ModuleFilterStore,
  members: undefined as unknown as () => unknown,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  project: {} as Partial<Project>,
  display: undefined as
    | ReturnType<typeof import("@/components/modules/use-module-display").useModuleDisplay>
    | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/use-module-filters", () => ({ useModuleFilters: () => held.filters }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => held.members() }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "u-me" } }),
  useUserProfile: () => ({ data: undefined }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    loader: "loaded",
    getProjectById: (id: string): Project | undefined => {
      const project = held.projects?.getProjectById(id);
      return project && { ...project, ...held.project };
    },
  }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { alt?: string }) => <i title={props.alt} />,
}));
vi.mock("../header", () => ({ ProjectArchivesHeader: () => <header /> }));
vi.mock("@/components/ui/loader/cycle-module-list-loader", () => ({
  CycleModuleListLayoutLoader: () => <p role="status">loading</p>,
}));
vi.mock("@/components/modules/progress-sidebar", () => ({
  ModuleDetailsSidebar: (props: { moduleId: string }) => <aside aria-label={`sidebar of ${props.moduleId}`} />,
}));
vi.mock("@/components/modules/modal", () => ({ CreateUpdateModuleModal: () => null }));

/** A route module's component, given its route's params as the router gives them. */
type RouteProps = { params: Readonly<Record<string, string | undefined>> };
const routeOf = (Component: unknown) => Component as ComponentType<RouteProps>;
function RouteOf({ Component }: { Component: ComponentType<RouteProps> }) {
  const params = useParams();
  return <Component params={params} />;
}
let location: Location | undefined;
function Where() {
  location = useLocation();
  return null;
}
/** How the person lays out and orders the route's project's modules, as the list's header does. */
function Display() {
  const { projectId = "" } = useParams();
  held.display = useModuleDisplay(projectId);
  return null;
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 12) });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  signedIn();
  localStorage.clear();
  toasts.length = 0;
  Object.assign(held, { project: {}, filters: new ModuleFilterStore() });
  held.modules = listedModules();
  held.members = (await membersAtAnotherRouter()).useMember;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const text = () => host.textContent ?? "";
const buttons = () => [...document.querySelectorAll("button")];
const button = (name: string) => buttons().find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);
/** The modules' rows' links, each with the name it shows (its title, after the ring of its progress). */
const links = () =>
  [...host.querySelectorAll<HTMLAnchorElement>("a[href*='/modules/']")].map((link) => ({
    link,
    name: link.querySelector(".truncate.text-13")?.textContent,
  }));
const names = () => links().map(({ name }) => name);
/** The options under the filter of title, in the open filters' menu. */
const optionsOf = (title: string) =>
  [
    ...([...document.querySelectorAll("button[aria-expanded]")]
      .find((one) => one.getAttribute("aria-label")?.startsWith(title))
      ?.closest(".py-2")
      ?.querySelectorAll<HTMLButtonElement>("button[aria-pressed]") ?? []),
  ].map((one) => one.textContent);

/** The caller of a role in acme and in web, at web's archive of modules, web being as project says. */
async function opened(caller: Caller, project: Partial<Project> = {}) {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  Object.assign(held, { permissions, projects, project });
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/archives/modules`]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/archives/modules"
              element={
                <>
                  <Display />
                  <RouteOf Component={routeOf(Layout)} />
                </>
              }
            >
              <Route index element={<RouteOf Component={routeOf(Page)} />} />
            </Route>
          </Routes>
          <Where />
        </MemoryRouter>
      </SWRConfig>
    )
  );
  await settle();
  return { acme, web };
}

/** An archived module of project, completed, unless fields say otherwise. */
const shelved = (project: Project, name: string, fields: Partial<Module> = {}) =>
  moduleOf(project, name, { status: "completed", archived_at: "2026-11-02T09:00:00Z", ...fields });

it("reads the project's archived modules for any of its members, its guest too, and waits for nerve", async () => {
  const { web } = await opened(its.guest);
  expect(held.modules.asked()).toEqual([`archive of ${web.id}`]);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
});

it("reads them while the project's modules are turned off too: nerve reads them", async () => {
  const { web } = await opened(its.member, { module_view: false });
  expect(held.modules.asked()).toEqual([`archive of ${web.id}`]);
});

it.each([
  { who: "the workspace's admin who is not its member", caller: its.outsider, archived: false },
  { who: "its member while it is archived", caller: its.member, archived: true },
])("reads nothing of the archive for $who", async ({ caller, archived }) => {
  await opened(caller, archived ? { archived_at: "2026-11-01T09:00:00Z" } : {});
  expect(held.modules.asked()).toEqual([]);
});

it("says why nerve could not read them, waiting no more, and reads them again on retry", async () => {
  const { web } = await opened(its.member);
  await held.modules.refuse("archive", new TypeError("Failed to fetch"));
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.unknown");
  expect(host.querySelector('[role="status"]')).toBeNull();
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.modules.asked()).toEqual([`archive of ${web.id}`, `archive of ${web.id}`]);
  await held.modules.answer("archive", [moduleRowOf(shelved(web, "Old"))]);
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(names()).toEqual(["Old"]);
});

it("says the project has no archived modules yet", async () => {
  await opened(its.member);
  await held.modules.answer("archive", []);
  await settle();
  expect(text()).toContain("workspace_empty_state.archive_modules.title");
  expect(host.querySelector('[role="status"]')).toBeNull();
});

it("orders them as the project's list is ordered, the one order the browser keeps for the project", async () => {
  const { web } = await opened(its.member);
  // nerve's order: the latest archived first
  await held.modules.answer("archive", [
    moduleRowOf(shelved(web, "Zeta", { sort_order: 1 }), { total_issues: 2, completed_issues: 2 }),
    moduleRowOf(shelved(web, "alpha", { sort_order: 2 }), { total_issues: 4, completed_issues: 1 }),
  ]);
  await settle();
  expect(names()).toEqual(["alpha", "Zeta"]);
  // the list's header orders the project's modules by their progress, the highest first
  await act(async () => held.display?.setOrderBy("-progress"));
  expect(names()).toEqual(["Zeta", "alpha"]);
  // the archive's own order menu orders the list too
  await act(async () => button("common.order_by.label: project_modules.order_by.progress")?.click());
  await settle();
  const manual = [...document.querySelectorAll('[role="menuitem"]')].find(
    (one) => one.textContent === "project_modules.order_by.manual"
  );
  await act(async () => (manual as HTMLElement | undefined)?.click());
  await settle();
  expect(held.display?.orderBy).toBe("sort_order");
  expect(names()).toEqual(["Zeta", "alpha"]);
});

it("lists each archived module, whose row opens its preview, not its page", async () => {
  const { acme, web } = await opened(its.member);
  await held.modules.answer("archive", [moduleRowOf(shelved(web, "Old"))]);
  await settle();
  await act(async () => links()[0]?.link.click());
  await settle();
  expect(location?.pathname).toBe(`/${acme.slug}/projects/${web.id}/archives/modules`);
  expect(location?.search).toBe("?peekModule=m-WEB-Old");
  expect(host.querySelector('aside[aria-label="sidebar of m-WEB-Old"]')).not.toBeNull();
  expect(held.modules.asked()).toEqual([
    `archive of ${web.id}`,
    "module of m-WEB-Old",
    "progress of m-WEB-Old",
    "links of m-WEB-Old",
  ]);
});

it("searches and filters them by the archive's own search and filters, the address's workspace members", async () => {
  const { web } = await opened(its.guest);
  await held.modules.answer("archive", [
    moduleRowOf(shelved(web, "Old", { lead_id: "u-Ann" })),
    moduleRowOf(shelved(web, "Older", { member_ids: ["u-Bob"], target_date: "2026-10-30" })),
  ]);
  await settle();
  await act(async () => button("common.filters")?.click());
  await settle();
  expect(text()).not.toContain("project_module.status");
  // the router names globex, whose member is Gil: the archive offers acme's, the address's workspace
  expect(optionsOf("lead")).toEqual(["Ann", "Bob"]);
  expect(optionsOf("members")).toEqual(["Ann", "Bob"]);
  await act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")]
      .find((one) => one.textContent === "Ann" && one.closest(".py-2")?.textContent?.startsWith("lead"))
      ?.click()
  );
  expect(held.filters.getFilters(web.id, "archived")).toEqual({ lead: ["u-Ann"] });
  expect(names()).toEqual(["Old"]);
  // the list's filters are its own
  await act(async () => held.filters.updateFilters(web.id, "default", { members: ["u-Bob"] }));
  expect(names()).toEqual(["Old"]);
  await act(async () =>
    held.filters.updateFilters(web.id, "archived", { lead: [], target_date: ["2026-10-30;after"] })
  );
  expect(names()).toEqual(["Older"]);
  await act(async () => held.filters.updateSearchQuery(web.id, "archived", "zzz"));
  expect(names()).toEqual([]);
  expect(text()).toContain("project_module.remove_search_criteria_to_see_all_modules");
  await act(async () => held.filters.updateSearchQuery(web.id, "archived", ""));
  await act(async () => held.filters.updateFilters(web.id, "archived", { target_date: ["2026-10-31;after"] }));
  expect(names()).toEqual([]);
  expect(text()).toContain("project_module.remove_filters_to_see_all_modules");
  await act(async () => button("common.clear_all")?.click());
  expect(names()).toEqual(["Old", "Older"]);
  // the list's search does not search the archive
  await act(async () => held.filters.updateSearchQuery(web.id, "default", "zzz"));
  expect(names()).toEqual(["Old", "Older"]);
});

it("restores an archived module by its quick actions, saying so once nerve has", async () => {
  const { acme, web } = await opened(its.member);
  const old = shelved(web, "Old");
  await held.modules.answer("archive", [moduleRowOf(old)]);
  await settle();
  await act(async () => button("common.item_options Old")?.click());
  await settle();
  const restore = [...document.querySelectorAll('[role="menuitem"]')].find((one) => one.textContent === "restore");
  await act(async () => (restore as HTMLElement | undefined)?.click());
  await settle();
  expect(held.modules.unarchive.mock.calls).toEqual([[old.id]]);
  expect(names()).toEqual(["Old"]);
  await act(async () => held.modules.sent.at(-1)?.answer({ ...old, archived_at: null }));
  await settle();
  expect(toasts.map((toast) => toast.title)).toEqual(["project_module.action.restore.success.title"]);
  expect(location?.pathname).toBe(`/${acme.slug}/projects/${web.id}/archives/modules`);
  expect(text()).toContain("workspace_empty_state.archive_modules.title");
});
