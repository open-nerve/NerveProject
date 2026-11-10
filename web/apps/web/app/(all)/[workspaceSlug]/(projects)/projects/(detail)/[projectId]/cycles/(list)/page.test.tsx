// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentType, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { signedIn } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { listedCycles } from "@/components/cycles/fake-cycle-lists";
import { rowOf } from "@/store/planning/fake-cycles";
import { CycleFilterStore } from "@/store/planning/cycle_filter.store";
import { cycleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "./layout";
import Page from "./page";

// A project's cycles list page (P5a spec 2.5, R8): its layout, its header and its body, at the route's address. Turned
// off, it says so, whose way to the project's features is its admin's; else it shows nerve's list of the project's
// cycles not archived (the read the project's wrapper makes, which the page shares), and waits for it, says why nerve
// could not give it with a retry, or how to set the first cycle. Those who write the project's cycles search, filter
// and add them. The caller's role is the permission store's own (fake-permissions.ts); the cycles a stand-in store's,
// the filters a session's; the list itself (CyclesView) a stand-in that says whose it shows.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
  filters: undefined as unknown as CycleFilterStore,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  project: {} as Partial<Project>,
  toggle: vi.fn(),
  mobileHeader: undefined as unknown,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/use-cycle-filters", () => ({ useCycleFilters: () => held.filters }));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateCycleModal: held.toggle }),
}));
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
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => {
    held.mobileHeader = props.mobileHeader;
    return <header>{props.header}</header>;
  },
}));
vi.mock("@/components/cycles/cycles-view", () => ({
  CyclesView: (props: { projectId: string }) => <p>the cycles of {props.projectId}</p>,
}));
vi.mock("@/components/ui/loader/cycle-module-list-loader", () => ({
  CycleModuleListLayoutLoader: () => <p role="status">loading</p>,
}));
vi.mock("@/components/cycles/modal", () => ({ CycleCreateUpdateModal: () => null }));

/** A route module's component, given its route's params as the router gives them. */
type RouteProps = { params: Readonly<Record<string, string | undefined>> };
const routeOf = (Component: unknown) => Component as ComponentType<RouteProps>;
function RouteOf({ Component }: { Component: ComponentType<RouteProps> }) {
  const params = useParams();
  return <Component params={params} />;
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
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
  held.toggle.mockReset();
  Object.assign(held, { project: {}, mobileHeader: undefined, filters: new CycleFilterStore() });
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
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);

/** The caller of a role in acme and in web, at web's cycles list, web being as project says. */
async function opened(caller: Caller, project: Partial<Project> = {}) {
  const { permissions, projects, acme, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll
  );
  Object.assign(held, { permissions, projects, project });
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/cycles`]}>
          <Routes>
            <Route path=":workspaceSlug/projects/:projectId/cycles" element={<RouteOf Component={routeOf(Layout)} />}>
              <Route index element={<RouteOf Component={routeOf(Page)} />} />
            </Route>
            <Route path=":workspaceSlug/settings/projects/:projectId/features/cycles" element={<p>the features</p>} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
  await settle();
  return { acme, web };
}

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: false },
])(
  "shows $caller.who the project's cycles turned off, reading none, whose features its admin alone manages",
  async ({ caller, may }) => {
    await opened(caller, { cycle_view: false });
    expect(text()).toContain("disabled_project.empty_state.cycle.title");
    expect(held.cycles.asked()).toEqual([]);
    const manage = button("disabled_project.empty_state.cycle.primary_button.text");
    expect(manage?.disabled).toBe(!may);
    await act(async () => manage?.click());
    expect(text().includes("the features")).toBe(may);
  }
);

it("waits for nerve's list of the project's cycles, the read the project's wrapper makes too", async () => {
  const { web } = await opened(its.member);
  expect(held.cycles.asked()).toEqual([`cycles of ${web.id}`]);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  expect(text()).not.toContain("the cycles of");
});

it("says why nerve could not list them, waiting no more, and lists them again on retry", async () => {
  const { web } = await opened(its.member);
  await held.cycles.refuse("cycles", refusal(503, "server_busy"));
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(host.querySelector('[role="status"]')).toBeNull();
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.cycles.asked()).toEqual([`cycles of ${web.id}`, `cycles of ${web.id}`]);
  await held.cycles.answer("cycles", [rowOf(cycleOf(web, "S1"))]);
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(text()).toContain(`the cycles of ${web.id}`);
});

it.each([
  { caller: its.member, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestAdmin, may: false },
  { caller: its.guestAdminSeesAll, may: true },
])(
  "shows $caller.who, the project having no cycles, how to set its first, which he may if he writes them",
  async ({ caller, may }) => {
    await opened(caller);
    await held.cycles.answer("cycles", []);
    await settle();
    expect(text()).toContain("project_empty_state.cycles.title");
    const first = button("project_empty_state.cycles.cta_primary");
    expect(first?.disabled).toBe(!may);
    await act(async () => first?.click());
    expect(held.toggle.mock.calls).toEqual(may ? [[true]] : []);
  }
);

it("shows the project's cycles, under the filters applied to them", async () => {
  const { web } = await opened(its.member);
  await held.cycles.answer("cycles", [rowOf(cycleOf(web, "S1"))]);
  await settle();
  expect(text()).toContain(`the cycles of ${web.id}`);
  expect(button("common.clear_all")).toBeUndefined();
  await act(async () => held.filters.updateFilters(web.id, "archived", { start_date: ["2026-11-01;after"] }));
  expect(button("common.clear_all")).toBeUndefined();
  await act(async () => held.filters.updateFilters(web.id, "default", { status: ["draft"] }));
  expect(text()).toContain("project_cycles.status.draft");
  await act(async () => button("common.clear_all")?.click());
  expect(held.filters.getFilters(web.id, "default")).toEqual({});
  expect(button("common.clear_all")).toBeUndefined();
});

it.each([
  { caller: its.admin, may: true },
  { caller: its.member, may: true },
  { caller: its.guest, may: false },
  { caller: its.guestAdmin, may: false },
  { caller: its.guestAdminSeesAll, may: true },
])(
  "gives $caller.who the search, the filters and a new cycle if he writes the project's cycles",
  async ({ caller, may }) => {
    const { acme, web } = await opened(caller);
    const crumb = [...host.querySelectorAll("header a")].find((link) => link.textContent === "cycles");
    expect(crumb?.getAttribute("href")).toBe(`/${acme.slug}/projects/${web.id}/cycles`);
    expect(button("common.search.label") !== undefined).toBe(may);
    expect(button("common.filters") !== undefined).toBe(may);
    const add = [...host.querySelectorAll("header button")].find((one) =>
      one.textContent?.includes("project_cycles.add_cycle")
    );
    expect(add !== undefined).toBe(may);
    await act(async () => (add as HTMLButtonElement | undefined)?.click());
    expect(held.toggle.mock.calls).toEqual(may ? [[true]] : []);
  }
);

it("has no layout menu for a small screen: the list has one layout", async () => {
  await opened(its.member);
  expect(held.mobileHeader).toBeUndefined();
});
