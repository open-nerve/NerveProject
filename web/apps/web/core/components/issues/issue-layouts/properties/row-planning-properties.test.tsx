// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.5, R3, R5, R11, R14, R20: a work item's modules and cycle among a list's or a board's row
// properties, against the session's stores and a fake nerve. Each shows while its display property is on and the row's
// own project has the domain on; read-only for a caller who cannot edit the row; named by its row; an archived one by
// its name and its caption, the project's lists read once for all its rows (on the member view, once a project); a
// module picked is one addition, applied to the set the row then holds. The display panel hides the cycle from the
// rows once nerve confirms the project's preferences. Layouts, rows, dropdowns, stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, Project, ProjectPreferences } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { seedAccount } from "@/store/user/fake-account";
import { HeaderFilters } from "../../filters";
import { ProjectLayoutRoot } from "../roots/project-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "rows" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "rows" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// jsdom has no viewport geometry.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const ARCHIVED = "2026-10-20T00:00:00Z";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
/** Each project's cycles and modules as nerve lists them: not archived, and archived (archived=true). */
let directory: Record<string, Record<"cycles" | "modules", { listed: unknown[]; archived: unknown[] }>>;
let projectB: Project;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  projectB = projectOf("B", tab.workspace.id, { id: "B" });
  directory = { A: directoryOf(tab.project), B: directoryOf(projectB) };
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    states?.(call);
    const match = /^\/api\/v0\/projects\/([^/]+)\/(cycles|modules)$/.exec(call.path);
    const lists = match && directory[match[1]!]?.[match[2] as "cycles" | "modules"];
    if (call.method === "GET" && lists)
      call.answer(json(200, { data: call.query.archived === "true" ? lists.archived : lists.listed }));
  };
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  toasts.length = 0;
});
afterEach(async () => tab.close());

/** A project's cycles Now (under way) and Old (archived), and its modules Alpha and Gamma (archived). */
const planning = (project: Project) => ({
  now: cycleOf(project, "Now", { status: "current" }),
  old: cycleOf(project, "Old", { status: "completed", archived_at: ARCHIVED }),
  alpha: moduleOf(project, "Alpha", { status: "in-progress" }),
  gamma: moduleOf(project, "Gamma", { status: "completed", archived_at: ARCHIVED }),
});
function directoryOf(project: Project) {
  const { now, old, alpha, gamma } = planning(project);
  return {
    cycles: { listed: [rowOf(now)], archived: [rowOf(old)] },
    modules: { listed: [moduleRowOf(alpha)], archived: [moduleRowOf(gamma)] },
  };
}
/** The row of the work item id of the project in its cycle and its modules, as nerve lists it. */
const row = (id: string, cycle_id: string | null, module_ids: string[], project_id = "A"): Issue =>
  listIssue(id, 1000, { project_id, state_id: "state", cycle_id, module_ids });

/** The row's control of the field, by the name its row gives it (R20). */
const control = (field: "common.cycle" | "common.modules", issue: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.startsWith(`${field}: `) && button.textContent.endsWith(`, ${issue}`)
  );
/** The reads of the project's list of the domain so far: "listed" for the list, "archived" for its archive. */
const readsOf = (project: string, domain: "cycles" | "modules") =>
  tab.nerve.calls
    .filter((call) => call.method === "GET" && call.path === `/api/v0/projects/${project}/${domain}`)
    .map((call) => (call.query.archived === "true" ? "archived" : "listed"));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");

type Layout = "list" | "kanban";
/** The project's page in the layout, its rows as nerve lists them; display changes the shown properties. */
async function projectPage(
  layout: Layout,
  rows: Issue[],
  display: Partial<ProjectPreferences["display_properties"]> = {}
) {
  const prefs = preferencesOf();
  prefs.display_filters.layout = layout;
  prefs.display_filters.group_by = layout === "kanban" ? "state_id" : null;
  prefs.display_properties = { ...prefs.display_properties, ...display };
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <HeaderFilters storeType={EIssuesStoreType.PROJECT} entityId="A" />
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  await until(() => tab.nerve.to("/api/v0/me/projects/A/preferences").length === 1, "the preferences");
  await act(async () => tab.nerve.to("/api/v0/me/projects/A/preferences")[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "the list");
  const page =
    layout === "kanban"
      ? { total: rows.length, groups: [{ key: "state", total: rows.length, data: rows, next_cursor: null }] }
      : { total: rows.length, data: rows, next_cursor: null };
  await act(async () => lists()[0]!.answer(json(200, page)));
  await settle();
  return prefs;
}

it.each(["list", "kanban"] as const)(
  "names the rows' archived cycle and modules with their caption in the %s layout, the project's lists read once for all its rows",
  async (layout) => {
    const { now, old, alpha, gamma } = planning(tab.project);
    await projectPage(layout, [
      row("one", old.id, [gamma.id]),
      row("two", old.id, [alpha.id, gamma.id]),
      row("three", now.id, []),
    ]);
    await act(async () => until(() => readsOf("A", "cycles").length === 2, "the cycles' lists"));
    await act(async () => until(() => readsOf("A", "modules").length === 2, "the modules' lists"));
    await settle();
    expect(control("common.cycle", "one")?.textContent).toBe("common.cycle: Old, common.archived, one");
    expect(control("common.cycle", "two")?.textContent).toBe("common.cycle: Old, common.archived, two");
    expect(control("common.cycle", "three")?.textContent).toBe("common.cycle: Now, three");
    expect(control("common.modules", "one")?.textContent).toBe("common.modules: Gamma, common.archived, one");
    expect(control("common.modules", "two")?.textContent).toBe(
      "common.modules: module.count: Alpha; Gamma, common.archived, two"
    );
    expect(control("common.modules", "three")?.textContent).toBe("common.modules: module.no_module, three");
    expect(readsOf("A", "cycles")).toEqual(["listed", "archived"]);
    expect(readsOf("A", "modules")).toEqual(["listed", "archived"]);
  }
);

it("hides the cycle from the rows once nerve confirms the project's preferences the display panel changed", async () => {
  const prefs = await projectPage("list", [row("one", null, [])]);
  expect(control("common.cycle", "one")).toBeDefined();
  await act(async () => document.querySelector<HTMLButtonElement>('button[aria-label="common.display"]')!.click());
  const property = [...document.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].find(
    (button) => button.textContent === "common.cycle"
  );
  await act(async () => property!.click());
  const saves = () => patches().filter((call) => call.path === "/api/v0/me/projects/A/preferences");
  await act(async () => until(() => saves().length === 1, "the preferences' change"));
  const hidden = { ...prefs.display_properties, cycle: false };
  expect(saves()[0]!.body).toEqual({ display_properties: hidden });
  expect(control("common.cycle", "one"), "nerve's preferences until it answers").toBeDefined();
  await act(async () => saves()[0]!.answer(json(200, { ...prefs, display_properties: hidden })));
  expect(control("common.cycle", "one")).toBeUndefined();
  expect(control("common.modules", "one")).toBeDefined();
  expect(patches()).toHaveLength(1);
});

it.each([
  { what: "the cycle while its display property is off", display: { cycle: false }, project: {}, shown: ["modules"] },
  { what: "the modules while theirs is off", display: { modules: false }, project: {}, shown: ["cycle"] },
  {
    what: "the cycle while the project's cycles are off",
    display: {},
    project: { cycle_view: false },
    shown: ["modules"],
  },
  {
    what: "the modules while the project's modules are off",
    display: {},
    project: { module_view: false },
    shown: ["cycle"],
  },
])("shows no $what", async ({ display, project, shown }) => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...project }]);
  tab.nerve.calls.length = 0;
  await projectPage("list", [row("one", null, [])], display);
  expect([control("common.modules", "one"), control("common.cycle", "one")].map((item) => item !== undefined)).toEqual([
    shown.includes("modules"),
    shown.includes("cycle"),
  ]);
});

it("is read-only for a caller who cannot edit the row", async () => {
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: 5 }));
  await projectPage("list", [row("one", null, [])]);
  const before = tab.nerve.calls.length;
  const fields = ["common.modules", "common.cycle"] as const;
  expect(fields.map((field) => control(field, "one")?.disabled)).toEqual([true, true]);
  await act(async () => {
    for (const field of fields) control(field, "one")!.click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(tab.nerve.calls).toHaveLength(before);
});

it("sends a module picked on a row as one addition to the set it holds, its archived module kept", async () => {
  const { alpha, gamma } = planning(tab.project);
  await projectPage("list", [row("one", null, [gamma.id])]);
  await act(async () => control("common.modules", "one")!.click());
  await act(async () =>
    until(() => readsOf("A", "modules").filter((read) => read === "listed").length === 2, "options")
  );
  await settle();
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === "Alpha"
  );
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(patches().map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/one", { module_ids: [gamma.id, alpha.id] }],
  ]);
  await act(async () =>
    patches()[0]!.answer(
      json(200, { ...issueDetailOf("one"), module_ids: [gamma.id, alpha.id], updated_at: "2026-10-30T00:00:00Z" })
    )
  );
  await settle();
  expect(control("common.modules", "one")?.textContent).toBe(
    "common.modules: module.count: Gamma, common.archived; Alpha, one"
  );
  expect(toasts).toEqual([]);
});

it("reads each project's lists once on the member view, and shows a row's planning by its own project's switches", async () => {
  await seedAccount(tab.nerve, tab.app.user, accountOf("alice"));
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    { ...projectB, module_view: false },
  ]);
  bound.params = { workspaceSlug: "ws", userId: "u-bob" };
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", userId: "u-bob" }));
  tab.nerve.calls.length = 0;
  const a = planning(tab.project);
  const b = planning(projectB);
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProfileIssuesPage type="assigned" />
      </MemoryRouter>
    </SWRConfig>
  );
  const member = () => tab.nerve.to("/api/v0/workspaces/ws/members/u-bob/issues");
  await until(() => member().length === 1, "the member's list");
  const rows = [
    row("a1", a.old.id, [a.gamma.id]),
    row("a2", a.old.id, [a.gamma.id]),
    row("b1", b.old.id, [b.gamma.id], "B"),
    row("b2", b.old.id, [b.gamma.id], "B"),
  ];
  await act(async () => member()[0]!.answer(json(200, { total: rows.length, data: rows, next_cursor: null })));
  await act(async () => until(() => readsOf("A", "cycles").length === 2, "A's cycles"));
  await act(async () => until(() => readsOf("B", "cycles").length === 2, "B's cycles"));
  await settle();
  for (const id of ["a1", "a2", "b1", "b2"])
    expect(control("common.cycle", id)?.textContent).toBe(`common.cycle: Old, common.archived, ${id}`);
  expect(control("common.modules", "a1")?.textContent).toBe("common.modules: Gamma, common.archived, a1");
  expect([control("common.modules", "b1"), control("common.modules", "b2")]).toEqual([undefined, undefined]);
  expect([readsOf("A", "cycles"), readsOf("B", "cycles")]).toEqual([
    ["listed", "archived"],
    ["listed", "archived"],
  ]);
  expect([readsOf("A", "modules"), readsOf("B", "modules")]).toEqual([["listed", "archived"], []]);
});
