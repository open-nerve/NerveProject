// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentType } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation, useParams, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Project } from "@nerve/api-client";
import { signedIn } from "@/lib/auth/fake-tab";
import { listedCycles } from "@/components/cycles/fake-cycle-lists";
import { rowOf } from "@/store/planning/fake-cycles";
import { CycleFilterStore } from "@/store/planning/cycle_filter.store";
import { cycleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "./layout";
import Page from "./page";

// A project's archive of cycles (P5a spec 2.5, R11): nerve's archived cycles of the project, latest archived first,
// read for its member while the project is not archived; a wait for them, why nerve could not read them with a retry
// (never a wait that does not end), or that there are none yet. Searched and filtered by their days (no status: each
// has ended), each row opens the cycle's preview. The caller's role is the permission store's own; the cycles a
// stand-in store's, the filters a session's; the archives' header, the avatars and the preview's sidebar stand-ins.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
  filters: undefined as unknown as CycleFilterStore,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  project: {} as Partial<Project>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/use-cycle-filters", () => ({ useCycleFilters: () => held.filters }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
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
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { getProjectMemberDetails: () => null } }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => <i /> }));
vi.mock("../header", () => ({ ProjectArchivesHeader: () => <header /> }));
vi.mock("@/components/ui/loader/cycle-module-list-loader", () => ({
  CycleModuleListLayoutLoader: () => <p role="status">loading</p>,
}));
vi.mock("@/components/cycles/progress-sidebar", () => ({
  CycleDetailsSidebar: (props: { cycleId: string }) => <aside aria-label={`sidebar of ${props.cycleId}`} />,
}));
vi.mock("@/components/cycles/modal", () => ({ CycleCreateUpdateModal: () => null }));

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

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 11, 12) });
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
  Object.assign(held, { project: {}, filters: new CycleFilterStore() });
  held.cycles = listedCycles();
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
/** The cycles' rows' links, each with the name it shows (its title, after the ring of its progress). */
const links = () =>
  [...host.querySelectorAll<HTMLAnchorElement>("a[href*='/cycles/']")].map((link) => ({
    link,
    name: link.querySelector(".truncate.text-13")?.textContent,
  }));
const names = () => links().map(({ name }) => name);

/** The caller of a role in acme and in web, at web's archive of cycles, web being as project says. */
async function opened(caller: Caller, project: Partial<Project> = {}) {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  Object.assign(held, { permissions, projects, project });
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/archives/cycles`]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/archives/cycles"
              element={<RouteOf Component={routeOf(Layout)} />}
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

/** An archived cycle of project, of a week that has ended, unless fields say otherwise. */
const shelved = (project: Project, name: string, fields: Partial<Cycle> = {}) =>
  cycleOf(project, name, {
    status: "completed",
    start_date: "2026-10-12",
    end_date: "2026-10-18",
    archived_at: "2026-10-20T09:00:00Z",
    ...fields,
  });

it("reads the project's archived cycles for any of its members, its guest too, and waits for nerve", async () => {
  const { web } = await opened(its.guest);
  expect(held.cycles.asked()).toEqual([`archive of ${web.id}`]);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
});

it.each([
  { who: "the workspace's admin who is not its member", caller: its.outsider, archived: false },
  { who: "its member while it is archived", caller: its.member, archived: true },
])("reads nothing of the archive for $who", async ({ caller, archived }) => {
  await opened(caller, archived ? { archived_at: "2026-11-01T09:00:00Z" } : {});
  expect(held.cycles.asked()).toEqual([]);
});

it("says why nerve could not read them, waiting no more, and reads them again on retry", async () => {
  const { web } = await opened(its.member);
  await held.cycles.refuse("archive", new TypeError("Failed to fetch"));
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.unknown");
  expect(host.querySelector('[role="status"]')).toBeNull();
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.cycles.asked()).toEqual([`archive of ${web.id}`, `archive of ${web.id}`]);
  await held.cycles.answer("archive", [rowOf(shelved(web, "S0"))]);
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(names()).toEqual(["S0"]);
});

it("says the project has no archived cycles yet", async () => {
  await opened(its.member);
  await held.cycles.answer("archive", []);
  await settle();
  expect(text()).toContain("workspace_empty_state.archive_cycles.title");
});

it("lists them in nerve's order, each of whose rows opens the cycle's preview, not its page", async () => {
  const { acme, web } = await opened(its.member);
  await held.cycles.answer("archive", [rowOf(shelved(web, "S1")), rowOf(shelved(web, "S0"))]);
  await settle();
  expect(names()).toEqual(["S1", "S0"]);
  await act(async () =>
    links()
      .find(({ name }) => name === "S0")
      ?.link.click()
  );
  await settle();
  expect(location?.pathname).toBe(`/${acme.slug}/projects/${web.id}/archives/cycles`);
  expect(location?.search).toBe("?peekCycle=c-WEB-S0");
  expect(host.querySelector('aside[aria-label="sidebar of c-WEB-S0"]')).not.toBeNull();
  expect(held.cycles.asked()).toEqual([`archive of ${web.id}`, "cycle of c-WEB-S0", "progress of c-WEB-S0"]);
});

it("searches them, and filters them by their days alone, no status: each has ended", async () => {
  const { web } = await opened(its.guest);
  await held.cycles.answer("archive", [
    rowOf(shelved(web, "S1", { start_date: "2026-10-26", end_date: "2026-11-01" })),
    rowOf(shelved(web, "S0")),
  ]);
  await settle();
  await act(async () => button("common.filters")?.click());
  await settle();
  const panel = text();
  expect(panel).toContain("start_date");
  expect(panel).toContain("end_date");
  expect(panel).not.toContain("project_cycles.filters.status");
  await act(async () => held.filters.updateFilters(web.id, "archived", { start_date: ["2026-10-20;after"] }));
  expect(names()).toEqual(["S1"]);
  // the list's filters are another's
  await act(async () => held.filters.updateFilters(web.id, "default", { start_date: ["2027-01-01;after"] }));
  expect(names()).toEqual(["S1"]);
  await act(async () => held.filters.updateSearchQuery(web.id, "archived", "s0"));
  expect(names()).toEqual([]);
  expect(text()).toContain("project_cycles.remove_search_criteria_to_see_all_cycles");
  await act(async () => held.filters.updateSearchQuery(web.id, "archived", ""));
  await act(async () => button("common.clear_all")?.click());
  expect(names()).toEqual(["S1", "S0"]);
});
