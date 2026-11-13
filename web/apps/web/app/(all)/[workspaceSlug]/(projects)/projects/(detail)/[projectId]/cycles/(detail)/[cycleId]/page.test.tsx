// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ComponentType } from "react";
import { observable, runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Project } from "@nerve/api-client";
import { heldChange, signedIn, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { cycleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "../layout";
import Page from "./page";

// A cycle's page (P5a spec 2.2, 2.5, R11; M3 handoff 1): its layout, its header and its body, at the route's address.
// It reads the cycle and its progress (an archived cycle's project's archive too, whose row has its members) for a
// member of the project, and says that nerve finds no such cycle, or why it could not read it, with a retry. Its
// subtree is the route cycle's: another cycle mounts it afresh, its host with it. The caller's role is the permission
// store's own (fake-permissions.ts); the cycles a stand-in store's, whose reads nerve settles when the test says; the
// sidebar, the switcher of the header's crumb, the dialog that changes a cycle and the cycle's work items (P5b,
// cycle-layout-root.test.tsx) stand-ins that say what they got; the header's controls of those work items (P5b,
// header.test.tsx) stand-ins that offer nothing.

type Switcher = {
  title?: string;
  navigationItems: { value: string; query: string }[];
  onChange?: (id: string) => void;
};
const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof pageCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  archivedProject: false,
  switcher: undefined as Switcher | undefined,
  sidebars: [] as string[],
  edit: undefined as { isOpen: boolean; cycleId?: string } | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    loader: "loaded",
    getProjectById: (id: string): Project | undefined => {
      const project = held.projects?.getProjectById(id);
      return project && held.archivedProject ? { ...project, archived_at: "2026-11-01T09:00:00Z" } : project;
    },
  }),
}));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  BreadcrumbNavigationSearchDropdown: (props: Switcher) => {
    held.switcher = props;
    return <span>{props.title}</span>;
  },
}));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: never }) => <header>{props.header}</header>,
}));
vi.mock("@/components/cycles/progress-sidebar", () => ({
  CycleDetailsSidebar: (props: { cycleId: string; handleClose: () => void }) => {
    // one entry each time a sidebar mounts, none as it renders again
    useState(() => held.sidebars.push(props.cycleId));
    return (
      <aside aria-label={props.cycleId}>
        <button type="button" onClick={props.handleClose}>
          close
        </button>
      </aside>
    );
  },
}));
vi.mock("@/components/cycles/transfer-issues", () => ({
  TransferIssues: (props: { cycleId: string }) => <p data-banner={props.cycleId} />,
}));
vi.mock("@/components/issues/issue-layouts/roots/planning-page-layout-root", () => ({
  PlanningPageLayoutRoot: () => <section data-work-items />,
}));
vi.mock("@/components/cycles/peek-on-entry", () => ({ PeekOnEntry: () => null }));
vi.mock("@/components/issues/filters", () => ({ HeaderFilters: () => null }));
vi.mock("@/hooks/store/use-issues", () => ({
  useIssues: () => ({ issues: { pageTotal: () => undefined }, issuesFilter: { getIssueFilters: () => undefined } }),
}));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateIssueModal: () => {} }),
}));
vi.mock("@/components/cycles/modal", () => ({
  CycleCreateUpdateModal: (props: NonNullable<typeof held.edit>) => {
    held.edit = props;
    return null;
  },
}));

/**
 * The cycles as the page reads them: the project's cycles not archived, listed already (the project's wrapper fetched
 * them), and each read the page makes, which nerve settles when the test says (reads); nerve's answer to the read of
 * the cycle is in the store before the read resolves, as the store does.
 */
function pageCycles(listed: Cycle[]) {
  const shown = observable.map(
    listed.map((cycle) => [cycle.id, cycle]),
    { deep: false }
  );
  const reads: { what: string; of: string; nerve: HeldChange<unknown> }[] = [];
  const read = (what: string) =>
    vi.fn((of: string) => {
      const nerve = heldChange<unknown>();
      reads.push({ what, of, nerve });
      return nerve.sent;
    });
  const readCycle = read("cycle");
  return {
    getCycleById: (id: string | null | undefined) => (id ? shown.get(id) : undefined),
    getProjectCycleIds: (projectId: string | null | undefined) =>
      listed.filter((cycle) => cycle.project_id === projectId).map((cycle) => cycle.id),
    fetchCycle: vi.fn(async (id: string) => {
      const answer = (await readCycle(id)) as Cycle;
      runInAction(() => shown.set(id, answer));
      return answer;
    }),
    fetchProgress: read("progress"),
    fetchCycles: read("list"),
    fetchArchivedCycles: read("archive"),
    reads,
    /** What the page has read so far, each as "what of id". */
    asked: () => reads.map((one) => `${one.what} of ${one.of}`),
    /** nerve answers the last read of what. */
    answer: (what: string, body: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.answer(body)),
    /** nerve refuses the last read of what. */
    refuse: (what: string, error: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.refuse(error)),
  };
}

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
  localStorage.clear();
  Object.assign(held, { archivedProject: false, switcher: undefined, sidebars: [], edit: undefined });
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
const sidebar = () => host.querySelector("aside")?.getAttribute("aria-label");
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);

/**
 * The caller of a role in acme and in web, at the page of web's cycle of name: the cycles listed are web's Alpha and
 * Beta (not archived, in nerve's order), the page's own one too unless unlisted (its read then gives it).
 */
async function opened(caller: Caller, name: string, fields: Partial<Cycle> = {}, unlisted = false) {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  Object.assign(held, { permissions, projects });
  const alpha = cycleOf(web, "Alpha", { status: "upcoming" });
  const beta = cycleOf(web, "Beta", { status: "current" });
  const cycle = name === "Alpha" ? alpha : name === "Beta" ? beta : cycleOf(web, name, fields);
  held.cycles = pageCycles(unlisted ? [alpha, beta] : [...new Set([alpha, beta, cycle])]);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/cycles/${cycle.id}`]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/cycles/:cycleId"
              element={<RouteOf Component={routeOf(Layout)} />}
            >
              <Route index element={<RouteOf Component={routeOf(Page)} />} />
            </Route>
            <Route path=":workspaceSlug/projects/:projectId/cycles" element={<p>the list</p>} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
  await settle();
  return { cycle, alpha, beta, web };
}

it("reads a member's cycle and its progress, and shows the cycle once nerve has given it", async () => {
  const { cycle, web } = await opened(its.member, "Gamma", { status: "upcoming" }, true);
  expect(held.cycles.asked()).toEqual([`cycle of ${cycle.id}`, `progress of ${cycle.id}`]);
  expect(sidebar()).toBeUndefined();
  expect(text()).not.toContain("project_cycles.not_found.title");
  await held.cycles.answer("cycle", cycle);
  await settle();
  expect(sidebar()).toBe(cycle.id);
  expect(held.cycles.asked()).toEqual([`cycle of ${cycle.id}`, `progress of ${cycle.id}`, `list of ${web.id}`]);
});

it("reads its project's cycles too for a cycle not archived, whose row has its members, its wrapper's read or not", async () => {
  // the project's wrapper reads them while the project's cycles are on, by the same key: the two share one read
  const { alpha, web } = await opened(its.member, "Alpha");
  expect(held.cycles.asked()).toEqual([`cycle of ${alpha.id}`, `progress of ${alpha.id}`, `list of ${web.id}`]);
  expect(sidebar()).toBe(alpha.id);
});

it.each([
  { who: "the workspace's admin who is not its member", caller: its.outsider, archived: false },
  { who: "its member while it is archived", caller: its.member, archived: true },
])("reads nothing of the cycle for $who", async ({ caller, archived }) => {
  held.archivedProject = archived;
  await opened(caller, "Alpha");
  expect(held.cycles.asked()).toEqual([]);
  expect(sidebar()).toBeUndefined();
});

it("reads its project's archive too for an archived cycle, whose row has its members (R11)", async () => {
  const shelved = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;
  const { cycle, web } = await opened(its.member, "Old", shelved, true);
  await held.cycles.answer("cycle", cycle);
  await settle();
  expect(held.cycles.asked()).toEqual([`cycle of ${cycle.id}`, `progress of ${cycle.id}`, `archive of ${web.id}`]);
  expect(sidebar()).toBe(cycle.id);
});

it("says a cycle nerve does not find does not exist, with the way back to the list", async () => {
  await opened(its.member, "Gone", {}, true);
  await held.cycles.refuse("cycle", refusal(404, "planning.cycle_not_found"));
  await settle();
  expect(text()).toContain("project_cycles.not_found.title");
  expect(sidebar()).toBeUndefined();
  await act(async () => button("project_cycles.not_found.action")?.click());
  expect(text()).toContain("the list");
});

it.each([
  { what: "cycle", error: refusal(503, "server_busy"), says: "errors.server_busy" },
  { what: "progress", error: new TypeError("Failed to fetch"), says: "errors.unknown" },
  { what: "list", error: refusal(503, "server_busy"), says: "errors.server_busy" },
])("says why its $what could not be read, and reads it again on retry", async ({ what, error, says }) => {
  const { cycle } = await opened(its.member, "Gamma", { status: "upcoming" }, true);
  if (what !== "cycle") await held.cycles.answer("cycle", cycle);
  await held.cycles.refuse(what, error);
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain(says);
  expect(sidebar()).toBeUndefined();
  const before = held.cycles.asked().length;
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.cycles.asked().slice(before)).toEqual([`${what} of ${what === "list" ? cycle.project_id : cycle.id}`]);
  await held.cycles.answer(what, what === "cycle" ? cycle : {});
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(sidebar()).toBe(cycle.id);
});

/** Chooses the item of title in the menu of the header's quick actions. */
async function chooseInHeader(title: string) {
  const menuButton = host.querySelector<HTMLButtonElement>('header button[aria-haspopup="menu"]');
  await act(async () => {
    menuButton?.focus();
    menuButton?.click();
  });
  await settle();
  const menu = document.getElementById(menuButton?.getAttribute("aria-controls") ?? "");
  const items = [...(menu?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])];
  await act(async () => items.find((item) => item.querySelector("h5")?.textContent === title)?.click());
  await settle();
}

it("mounts the page of another cycle afresh, its host with it: the dialog of the cycle before ends", async () => {
  const { alpha, beta, web } = await opened(its.admin, "Alpha");
  await chooseInHeader("edit");
  expect(held.edit).toMatchObject({ isOpen: true, cycleId: alpha.id });
  await act(async () => held.switcher?.onChange?.(beta.id));
  await settle();
  expect(held.edit).toMatchObject({ isOpen: false });
  expect(held.sidebars).toEqual([alpha.id, beta.id]);
  // the project's list is the project's, the one read of it shared by both
  expect(held.cycles.asked()).toEqual([
    `cycle of ${alpha.id}`,
    `progress of ${alpha.id}`,
    `list of ${web.id}`,
    `cycle of ${beta.id}`,
    `progress of ${beta.id}`,
  ]);
});

it("switches to the project's cycles that are not archived, in nerve's order, by name", async () => {
  const shelved = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;
  const { cycle, alpha, beta } = await opened(its.member, "Old", shelved, true);
  await held.cycles.answer("cycle", cycle);
  await settle();
  expect(held.switcher?.title).toBe("Old");
  expect(held.switcher?.navigationItems.map(({ value, query }) => [value, query])).toEqual([
    [alpha.id, "Alpha"],
    [beta.id, "Beta"],
  ]);
});

it("shows in its body the banner of the cycle's transfer, which says itself whether it shows, its work items under it, then its sidebar", async () => {
  const { alpha } = await opened(its.member, "Alpha");
  const banner = host.querySelector("[data-banner]");
  expect(banner?.getAttribute("data-banner")).toBe(alpha.id);
  const workItems = host.querySelector("[data-work-items]");
  expect(workItems && banner?.compareDocumentPosition(workItems)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  const aside = host.querySelector("aside");
  expect(aside && workItems?.compareDocumentPosition(aside)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
});

it("hides and shows its sidebar by the header's toggle, and hides it by the sidebar's close", async () => {
  const { alpha } = await opened(its.member, "Alpha");
  expect(sidebar()).toBe(alpha.id);
  await act(async () => button("aria_labels.toggle_details")?.click());
  expect(sidebar()).toBeUndefined();
  expect(localStorage.getItem("cycle_sidebar_collapsed")).toBe("true");
  await act(async () => button("aria_labels.toggle_details")?.click());
  expect(sidebar()).toBe(alpha.id);
  await act(async () => button("close")?.click());
  expect(sidebar()).toBeUndefined();
});
