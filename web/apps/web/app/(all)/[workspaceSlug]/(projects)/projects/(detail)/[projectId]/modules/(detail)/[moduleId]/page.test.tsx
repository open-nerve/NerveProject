// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ComponentType } from "react";
import { observable, runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import { heldChange, signedIn, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { moduleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import Layout from "../layout";
import Page from "./page";

// A module's page (P6a spec 2.6, R3, R12; M3 handoff 1; P6b spec 2.7): its layout, its header and its body, at the
// route's address. It reads the module, its progress and its links for a member of the project, whether its modules
// are turned on or not, and says that nerve finds no such module, or why it could not read one of them, with a retry.
// Its body is the module's work items, its sidebar on their right. Its subtree is the route module's: another module
// mounts it afresh, its host with it. The caller's role is the permission store's own (fake-permissions.ts); the
// modules a stand-in store's, whose reads nerve settles when the test says; the sidebar, the switcher of the header's
// crumb, the dialog that changes a module and the module's work items (P6b, module-layout-root.test.tsx) stand-ins
// that say what they got; the header's quick actions the real ones.

type Switcher = {
  title?: string;
  navigationItems: { value: string; query: string }[];
  onChange?: (id: string) => void;
};
const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof pageModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  archivedProject: false,
  switcher: undefined as Switcher | undefined,
  sidebars: [] as string[],
  edit: undefined as { isOpen: boolean; moduleId?: string } | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
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
vi.mock("@/components/modules/progress-sidebar", () => ({
  ModuleDetailsSidebar: (props: { moduleId: string; handleClose: () => void }) => {
    // one entry each time a sidebar mounts, none as it renders again
    useState(() => held.sidebars.push(props.moduleId));
    return (
      <aside aria-label={props.moduleId}>
        <button type="button" onClick={props.handleClose}>
          close
        </button>
      </aside>
    );
  },
}));
vi.mock("@/components/issues/issue-layouts/roots/planning-page-layout-root", () => ({
  PlanningPageLayoutRoot: (props: { storeType: string }) => <section data-work-items={props.storeType} />,
}));
vi.mock("@/components/modules/modal", () => ({
  CreateUpdateModuleModal: (props: NonNullable<typeof held.edit>) => {
    held.edit = props;
    return null;
  },
}));

/**
 * The modules as the page reads them: the project's modules not archived, listed already (the project's wrapper
 * fetched them), and each read the page makes, which nerve settles when the test says (reads); nerve's answer to the
 * read of the module is in the store before the read resolves, as the store does.
 */
function pageModules(listed: Module[]) {
  const shown = observable.map(
    listed.map((module) => [module.id, module]),
    { deep: false }
  );
  const reads: { what: string; of: string; nerve: HeldChange<unknown> }[] = [];
  const read = (what: string) =>
    vi.fn((of: string) => {
      const nerve = heldChange<unknown>();
      reads.push({ what, of, nerve });
      return nerve.sent;
    });
  const readModule = read("module");
  return {
    getModuleById: (id: string | null | undefined) => (id ? shown.get(id) : undefined),
    getProjectModuleIds: (projectId: string | null | undefined) =>
      listed.filter((module) => module.project_id === projectId && module.archived_at === null).map(({ id }) => id),
    fetchModule: vi.fn(async (id: string) => {
      const answer = (await readModule(id)) as Module;
      runInAction(() => shown.set(id, answer));
      return answer;
    }),
    fetchProgress: read("progress"),
    fetchLinks: read("links"),
    unarchive: vi.fn(),
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
const ARCHIVED = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;

/**
 * The caller of a role in acme and in web (of fields), at the page of web's module of name: the modules listed are
 * web's Alpha and Beta (not archived, in nerve's order), the page's own one too unless unlisted (its read then gives
 * it).
 */
async function opened(
  caller: Caller,
  name: string,
  {
    fields = {},
    unlisted = false,
    project = {},
  }: { fields?: Partial<Module>; unlisted?: boolean; project?: object } = {}
) {
  const { permissions, projects, acme, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    false,
    project
  );
  Object.assign(held, { permissions, projects });
  const alpha = moduleOf(web, "Alpha", { status: "planned" });
  const beta = moduleOf(web, "Beta", { status: "in-progress" });
  const module = name === "Alpha" ? alpha : name === "Beta" ? beta : moduleOf(web, name, fields);
  held.modules = pageModules(unlisted ? [alpha, beta] : [...new Set([alpha, beta, module])]);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/modules/${module.id}`]}>
          <Routes>
            <Route
              path=":workspaceSlug/projects/:projectId/modules/:moduleId"
              element={<RouteOf Component={routeOf(Layout)} />}
            >
              <Route index element={<RouteOf Component={routeOf(Page)} />} />
            </Route>
            <Route path=":workspaceSlug/projects/:projectId/modules" element={<p>the list</p>} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
  await settle();
  return { module, alpha, beta, web };
}

/** The items the menu of the header's quick actions offers, by their titles. */
async function headerMenu() {
  const menuButton = host.querySelector<HTMLButtonElement>('header button[aria-haspopup="menu"]');
  await act(async () => {
    menuButton?.focus();
    menuButton?.click();
  });
  await settle();
  const menu = document.getElementById(menuButton?.getAttribute("aria-controls") ?? "");
  return [...(menu?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])];
}
const titlesOf = (items: HTMLButtonElement[]) => items.map((item) => item.querySelector("h5")?.textContent);

it("reads a member's module, its progress and its links, and shows the module once nerve has given it", async () => {
  const { module } = await opened(its.member, "Gamma", { unlisted: true });
  expect(held.modules.asked()).toEqual([`module of ${module.id}`, `progress of ${module.id}`, `links of ${module.id}`]);
  expect(sidebar()).toBeUndefined();
  expect(text()).not.toContain("project_module.not_found.title");
  await held.modules.answer("module", module);
  await settle();
  expect(sidebar()).toBe(module.id);
});

it("shows a module its project's list holds while its own read is out", async () => {
  const { alpha } = await opened(its.member, "Alpha");
  expect(held.modules.asked()).toEqual([`module of ${alpha.id}`, `progress of ${alpha.id}`, `links of ${alpha.id}`]);
  expect(sidebar()).toBe(alpha.id);
});

it.each([
  { who: "the workspace's admin who is not its member", caller: its.outsider, archived: false },
  { who: "its member while it is archived", caller: its.member, archived: true },
])("reads nothing of the module for $who", async ({ caller, archived }) => {
  held.archivedProject = archived;
  await opened(caller, "Alpha");
  expect(held.modules.asked()).toEqual([]);
  expect(sidebar()).toBeUndefined();
});

it("reads a guest's module, and one of a project whose modules are turned off, read-only (R3)", async () => {
  const { alpha } = await opened(its.admin, "Alpha", { project: { module_view: false } });
  expect(held.modules.asked()).toEqual([`module of ${alpha.id}`, `progress of ${alpha.id}`, `links of ${alpha.id}`]);
  expect(sidebar()).toBe(alpha.id);
  expect(titlesOf(await headerMenu())).toEqual(["open_in_new_tab", "copy_link"]);
  await act(async () => root.unmount());
  root = createRoot(host);
  const { beta } = await opened(its.guest, "Beta");
  expect(held.modules.asked()).toEqual([`module of ${beta.id}`, `progress of ${beta.id}`, `links of ${beta.id}`]);
  expect(sidebar()).toBe(beta.id);
});

it("shows an archived module as any other, read-only, out of the switcher of the project's modules (R12)", async () => {
  const { module, alpha, beta } = await opened(its.admin, "Old", { fields: ARCHIVED, unlisted: true });
  await held.modules.answer("module", module);
  await settle();
  expect(sidebar()).toBe(module.id);
  expect(held.switcher?.title).toBe("Old");
  expect(held.switcher?.navigationItems.map(({ value, query }) => [value, query])).toEqual([
    [alpha.id, "Alpha"],
    [beta.id, "Beta"],
  ]);
  expect(titlesOf(await headerMenu())).toEqual(["open_in_new_tab", "copy_link", "restore"]);
});

it("says a module nerve does not find does not exist, with the way back to the list", async () => {
  await opened(its.member, "Gone", { unlisted: true });
  await held.modules.refuse("module", refusal(404, "planning.module_not_found"));
  await settle();
  expect(text()).toContain("project_module.not_found.title");
  expect(sidebar()).toBeUndefined();
  await act(async () => button("project_module.not_found.action")?.click());
  expect(text()).toContain("the list");
});

it("says a module of another project does not exist here", async () => {
  const { module } = await opened(its.member, "Elsewhere", { unlisted: true });
  await held.modules.answer("module", { ...module, project_id: "p-other" });
  await settle();
  expect(text()).toContain("project_module.not_found.title");
  expect(sidebar()).toBeUndefined();
});

it.each([
  { what: "module", error: refusal(503, "server_busy"), says: "errors.server_busy" },
  { what: "progress", error: new TypeError("Failed to fetch"), says: "errors.unknown" },
  { what: "links", error: refusal(503, "server_busy"), says: "errors.server_busy" },
])("says why its $what could not be read, and reads it again on retry", async ({ what, error, says }) => {
  const { module } = await opened(its.member, "Gamma", { unlisted: true });
  if (what !== "module") await held.modules.answer("module", module);
  await held.modules.refuse(what, error);
  await settle();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain(says);
  expect(text()).not.toContain("project_module.not_found.title");
  expect(sidebar()).toBeUndefined();
  const before = held.modules.asked().length;
  await act(async () => button("common.retry")?.click());
  await settle();
  expect(held.modules.asked().slice(before)).toEqual([`${what} of ${module.id}`]);
  await held.modules.answer(what, what === "module" ? module : {});
  await settle();
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(sidebar()).toBe(module.id);
});

it("mounts the page of another module afresh, its host with it: the dialog of the module before ends", async () => {
  const { alpha, beta } = await opened(its.admin, "Alpha");
  const items = await headerMenu();
  await act(async () => items.find((item) => item.querySelector("h5")?.textContent === "edit")?.click());
  await settle();
  expect(held.edit).toMatchObject({ isOpen: true, moduleId: alpha.id });
  await act(async () => held.switcher?.onChange?.(beta.id));
  await settle();
  expect(held.edit).toMatchObject({ isOpen: false });
  expect(held.sidebars).toEqual([alpha.id, beta.id]);
  expect(held.modules.asked()).toEqual([
    `module of ${alpha.id}`,
    `progress of ${alpha.id}`,
    `links of ${alpha.id}`,
    `module of ${beta.id}`,
    `progress of ${beta.id}`,
    `links of ${beta.id}`,
  ]);
});

it("switches to the project's modules that are not archived, in nerve's order, by name", async () => {
  const { alpha, beta, module } = await opened(its.member, "Gamma");
  expect(held.switcher?.title).toBe("Gamma");
  expect(held.switcher?.navigationItems.map(({ value, query }) => [value, query])).toEqual([
    [alpha.id, "Alpha"],
    [beta.id, "Beta"],
    [module.id, "Gamma"],
  ]);
});

it("shows in its body the module's work items, by the module's store type, then its sidebar on their right", async () => {
  const { alpha } = await opened(its.member, "Alpha");
  const workItems = host.querySelector("[data-work-items]");
  expect(workItems?.getAttribute("data-work-items")).toBe("MODULE");
  const aside = host.querySelector("aside");
  expect(aside?.getAttribute("aria-label")).toBe(alpha.id);
  expect(aside && workItems?.compareDocumentPosition(aside)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
});

it("hides and shows its sidebar by the header's toggle, and hides it by the sidebar's close", async () => {
  const { alpha } = await opened(its.member, "Alpha");
  expect(sidebar()).toBe(alpha.id);
  await act(async () => button("aria_labels.toggle_details")?.click());
  expect(sidebar()).toBeUndefined();
  expect(localStorage.getItem("module_sidebar_collapsed")).toBe("true");
  await act(async () => button("aria_labels.toggle_details")?.click());
  expect(sidebar()).toBe(alpha.id);
  await act(async () => button("close")?.click());
  expect(sidebar()).toBeUndefined();
});
