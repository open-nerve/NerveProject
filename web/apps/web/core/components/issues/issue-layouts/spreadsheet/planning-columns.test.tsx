// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.5, R3, R11, R14, R20, R21 (re-72 F10), P7b spec 2.8, R9: a work item's modules and cycle as two
// columns of a project's table, after its labels, against the session's stores and a fake nerve. Each shows while its
// display property is on and the page's project has the domain on; its header is its title and its icon, which open
// its sort by the names both ways (the list's catalog names them, P7b), a saved name order shown and sent as it is; a
// cell is read-only for a caller who cannot edit the row, named by its row, an archived value by its name and its
// caption, the project's lists read once for all its rows; a cycle picked is the cycle alone, a module picked one
// addition or removal, applied in the work item's turn to the set it then holds. The layout, the table, rows,
// dropdowns, stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, Project, ProjectPreferences } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, preferencesOf } from "@/store/project/fake-projects";
import { ProjectLayoutRoot } from "../roots/project-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "cells" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "cells" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const ARCHIVED = "2026-10-20T00:00:00Z";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  const { now, old, alpha, beta, gamma } = planning(tab.project);
  /** The project's cycles and modules as nerve lists them: not archived, and archived (archived=true). */
  const directory = {
    cycles: { listed: [rowOf(now)], archived: [rowOf(old)] },
    modules: { listed: [moduleRowOf(alpha), moduleRowOf(beta)], archived: [moduleRowOf(gamma)] },
  };
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    states?.(call);
    const match = /^\/api\/v0\/projects\/A\/(cycles|modules)$/.exec(call.path);
    const lists = match && directory[match[1] as "cycles" | "modules"];
    if (call.method === "GET" && lists)
      call.answer(json(200, { data: call.query.archived === "true" ? lists.archived : lists.listed }));
  };
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  toasts.length = 0;
});
afterEach(async () => tab.close());

/** A project's cycles Now (under way) and Old (archived), and its modules Alpha, Beta and Gamma (archived). */
const planning = (project: Project) => ({
  now: cycleOf(project, "Now", { status: "current" }),
  old: cycleOf(project, "Old", { status: "completed", archived_at: ARCHIVED }),
  alpha: moduleOf(project, "Alpha", { status: "in-progress" }),
  beta: moduleOf(project, "Beta", { status: "planned" }),
  gamma: moduleOf(project, "Gamma", { status: "completed", archived_at: ARCHIVED }),
});
/** The row of the work item id of project A in its cycle and its modules, as nerve lists it. */
const row = (id: string, cycle_id: string | null, module_ids: string[]): Issue =>
  listIssue(id, 1000, { project_id: "A", state_id: "state", cycle_id, module_ids });

/** The table's column headers after the work items' own, by their titles. */
const titles = () => [...document.querySelectorAll("thead th")].slice(1).map((header) => header.textContent);
const header = (title: string) =>
  [...document.querySelectorAll<HTMLTableCellElement>("thead th")].find((cell) => cell.textContent === title);
/** The cell's control of the field, by the name its row gives it (R20). */
const control = (field: "common.cycle" | "common.modules", issue: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("tbody button")].find(
    (button) => button.textContent?.startsWith(`${field}: `) && button.textContent.endsWith(`, ${issue}`)
  );
/** The reads of the project's list of the domain so far: "listed" for the list, "archived" for its archive. */
const readsOf = (domain: "cycles" | "modules") =>
  tab.nerve.calls
    .filter((call) => call.method === "GET" && call.path === `/api/v0/projects/A/${domain}`)
    .map((call) => (call.query.archived === "true" ? "archived" : "listed"));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
async function pick(text: string) {
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === text
  );
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

/**
 * The project's table, its rows as nerve lists them; display changes the shown properties, order_by the saved order.
 */
async function table(
  rows: Issue[],
  display: Partial<ProjectPreferences["display_properties"]> = {},
  order_by: ProjectPreferences["display_filters"]["order_by"] = preferencesOf().display_filters.order_by
) {
  const prefs = preferencesOf();
  prefs.display_filters.layout = "spreadsheet";
  prefs.display_filters.order_by = order_by;
  prefs.display_properties = { ...prefs.display_properties, ...display };
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
  await until(() => tab.nerve.to("/api/v0/me/projects/A/preferences").length === 1, "the preferences");
  await act(async () => tab.nerve.to("/api/v0/me/projects/A/preferences")[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "the list");
  await act(async () => lists()[0]!.answer(json(200, { total: rows.length, data: rows, next_cursor: null })));
  await settle();
}

it("shows the modules' and the cycle's columns after the labels, each header its title and its icon opening its sort", async () => {
  await table([row("one", null, [])]);
  expect(titles()).toEqual([
    "common.state",
    "common.priority",
    "common.assignees",
    "common.labels",
    "common.modules",
    "common.cycle",
    "common.order_by.start_date",
    "common.order_by.due_date",
    "common.sort.created_on",
    "common.sort.updated_on",
    "common.link",
    "content_files.title",
    "issue.display.properties.sub_issue",
  ]);
  for (const title of ["common.modules", "common.cycle"]) {
    const cell = header(title)!;
    expect(cell.querySelector("svg"), title).not.toBeNull();
    expect(cell.querySelector("button"), title).not.toBeNull();
    expect(cell.getAttribute("aria-sort"), title).toBe("none");
  }
  expect(header("common.labels")!.querySelector("button")).not.toBeNull();
  expect(header("common.labels")!.getAttribute("aria-sort")).toBe("none");
  expect([control("common.modules", "one"), control("common.cycle", "one")].every(Boolean)).toBe(true);
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
])("shows no column of $what", async ({ display, project, shown }) => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...project }]);
  tab.nerve.calls.length = 0;
  await table([row("one", null, [])], display);
  expect([titles().includes("common.modules"), titles().includes("common.cycle")]).toEqual([
    shown.includes("modules"),
    shown.includes("cycle"),
  ]);
  expect([control("common.modules", "one"), control("common.cycle", "one")].map((item) => item !== undefined)).toEqual([
    shown.includes("modules"),
    shown.includes("cycle"),
  ]);
});

it("names the cells' archived cycle and modules with their caption, the project's lists read once for all its rows", async () => {
  const { now, old, alpha, gamma } = planning(tab.project);
  await table([row("one", old.id, [gamma.id]), row("two", old.id, [alpha.id, gamma.id]), row("three", now.id, [])]);
  await act(async () => until(() => readsOf("cycles").length === 2, "the cycles' lists"));
  await act(async () => until(() => readsOf("modules").length === 2, "the modules' lists"));
  await settle();
  expect(control("common.cycle", "one")?.textContent).toBe("common.cycle: Old, common.archived, one");
  expect(control("common.cycle", "two")?.textContent).toBe("common.cycle: Old, common.archived, two");
  expect(control("common.cycle", "three")?.textContent).toBe("common.cycle: Now, three");
  expect(control("common.modules", "one")?.textContent).toBe("common.modules: Gamma, common.archived, one");
  expect(control("common.modules", "two")?.textContent).toBe(
    "common.modules: module.count: Alpha; Gamma, common.archived, two"
  );
  expect(control("common.modules", "three")?.textContent).toBe("common.modules: module.no_module, three");
  expect(readsOf("cycles")).toEqual(["listed", "archived"]);
  expect(readsOf("modules")).toEqual(["listed", "archived"]);
});

it("sends a cycle picked in a cell as the cycle alone, out of an archived one", async () => {
  const { now, old } = planning(tab.project);
  await table([row("one", old.id, [])]);
  await act(async () => until(() => readsOf("cycles").length === 2, "the cycles' lists"));
  await settle();
  await act(async () => control("common.cycle", "one")!.click());
  await act(async () => until(() => readsOf("cycles").length === 3, "the options"));
  await settle();
  await pick("Now");
  expect(patches().map((call) => [call.path, call.body])).toEqual([["/api/v0/issues/one", { cycle_id: now.id }]]);
  await act(async () =>
    patches()[0]!.answer(json(200, { ...issueDetailOf("one"), cycle_id: now.id, updated_at: "2026-10-30T00:00:00Z" }))
  );
  await settle();
  expect(control("common.cycle", "one")?.textContent).toBe("common.cycle: Now, one");
  expect(toasts).toEqual([]);
});

it("sends two module picks in a cell as an addition and a removal, each applied in its turn to the set the row then holds", async () => {
  const { alpha, beta, gamma } = planning(tab.project);
  await table([row("one", null, [alpha.id, gamma.id])]);
  await act(async () => until(() => readsOf("modules").length === 2, "the modules' lists"));
  await settle();
  await act(async () => control("common.modules", "one")!.click());
  await act(async () => until(() => readsOf("modules").length === 3, "the options"));
  await settle();
  await pick("Beta");
  await pick("Alpha");
  expect(patches().map((call) => call.body)).toEqual([{ module_ids: [alpha.id, gamma.id, beta.id] }]);
  await act(async () =>
    patches()[0]!.answer(
      json(200, {
        ...issueDetailOf("one"),
        module_ids: [alpha.id, gamma.id, beta.id],
        updated_at: "2026-10-30T00:00:00Z",
      })
    )
  );
  await act(async () => until(() => patches().length === 2, "the second pick, in its turn"));
  expect(patches()[1]!.body).toEqual({ module_ids: [gamma.id, beta.id] });
  await act(async () =>
    patches()[1]!.answer(
      json(200, { ...issueDetailOf("one"), module_ids: [gamma.id, beta.id], updated_at: "2026-10-30T00:01:00Z" })
    )
  );
  await settle();
  expect(control("common.modules", "one")?.textContent).toBe(
    "common.modules: module.count: Gamma, common.archived; Beta, one"
  );
  expect(toasts).toEqual([]);
});

it("returns the focus to its cell once a cell's list closes, the cycle's on a pick and the modules' on Escape", async () => {
  const { now } = planning(tab.project);
  await table([row("one", null, [])]);
  const cell = (field: "common.cycle" | "common.modules") => control(field, "one")!.closest("td");
  await act(async () => control("common.cycle", "one")!.click());
  await act(async () => until(() => readsOf("cycles").length === 1, "the cycles' options"));
  await settle();
  await pick("Now");
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.activeElement).toBe(cell("common.cycle"));
  await act(async () =>
    patches()[0]!.answer(json(200, { ...issueDetailOf("one"), cycle_id: now.id, updated_at: "2026-10-30T00:00:00Z" }))
  );
  await act(async () => control("common.modules", "one")!.click());
  await act(async () => until(() => readsOf("modules").length === 1, "the modules' options"));
  await settle();
  const search = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  expect(document.activeElement).toBe(search);
  await act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.activeElement).toBe(cell("common.modules"));
});

it("keeps both cells read-only for a caller who cannot edit the row", async () => {
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: 5 }));
  await table([row("one", null, [])]);
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

/** The work items' names in the table's order. */
const rowNames = () =>
  [...document.querySelectorAll("tbody > tr > td:first-child .line-clamp-1 .truncate")].map((node) => node.textContent);
const issueLists = () => tab.nerve.to("/api/v0/projects/A/issues");
const preferenceWrites = () =>
  tab.nerve.to("/api/v0/me/projects/A/preferences").filter((call) => call.method === "PATCH");
/** Picks a column's sort from its header's menu, "AZ" (A → Z) or "ZA" (Z → A) by the item's text. */
async function sortBy(title: string, item: "AZ" | "ZA") {
  const trigger = header(title)?.querySelector("button");
  if (!trigger) throw new Error(`The header ${title} offers no sort`);
  vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 180, 40));
  await act(async () => trigger.click());
  const option = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (node) => node.textContent === item
  );
  await act(async () => option!.click());
}
/**
 * Picks a column's sort from its header, which writes the confirmed preferences with that order, confirms them and
 * answers the list they read by that order with the rows of page; gives the preferences confirmed.
 */
async function sortAndList(
  title: string,
  item: "AZ" | "ZA",
  wire: ProjectPreferences["display_filters"]["order_by"],
  held: ProjectPreferences,
  page: Issue[]
) {
  const [written, listed] = [preferenceWrites().length, issueLists().length];
  await sortBy(title, item);
  await until(() => preferenceWrites().length === written + 1, `the sort by ${wire}`);
  const write = preferenceWrites()[written]!;
  expect(write.body).toEqual({ display_filters: { ...held.display_filters, order_by: wire } });
  const confirmed = { ...held, display_filters: { ...held.display_filters, order_by: wire } };
  await act(async () => write.answer(json(200, confirmed)));
  await until(() => issueLists().length === listed + 1, `the list by ${wire}`);
  const list = issueLists()[listed]!;
  expect(list.query.order_by).toBe(wire);
  await act(async () => list.answer(json(200, { total: page.length, data: page, next_cursor: null })));
  await settle();
  return confirmed;
}

it.each([
  {
    title: "common.cycle",
    other: "common.modules",
    order: "cycle_name",
    ascending: ["two", "one", "three"],
    descending: ["one", "two", "three"],
  },
  {
    title: "common.modules",
    other: "common.cycle",
    order: "module_name",
    ascending: ["one", "two", "three"],
    descending: ["two", "one", "three"],
  },
] as const)(
  "sorts by $order both ways from its header: the preference, then the list's query, its rows and aria-sort",
  async ({ title, other, order, ascending, descending }) => {
    const { now, old, alpha, beta, gamma } = planning(tab.project);
    const rows = [row("one", old.id, [alpha.id, gamma.id]), row("two", now.id, [beta.id]), row("three", null, [])];
    await table(rows);
    const page = (names: readonly string[]) => names.map((id) => rows.find((issue) => issue.id === id)!);
    /** The headers' aria-sort: the column's own, then the other planning column's and the labels'. */
    const sorts = () => [title, other, "common.labels"].map((cell) => header(cell)!.getAttribute("aria-sort"));
    const held = preferencesOf();
    held.display_filters.layout = "spreadsheet";
    const confirmed = await sortAndList(title, "AZ", order, held, page(ascending));
    expect(rowNames()).toEqual(ascending);
    expect(sorts()).toEqual(["ascending", "none", "none"]);
    await sortAndList(title, "ZA", `-${order}`, confirmed, page(descending));
    expect(rowNames()).toEqual(descending);
    expect(sorts()).toEqual(["descending", "none", "none"]);
    expect(toasts).toEqual([]);
  }
);

it("shows and sends a saved -module_name as it is: the list's query, its rows and the modules' header descending", async () => {
  const { now, old, alpha, beta, gamma } = planning(tab.project);
  await table(
    [row("two", now.id, [beta.id]), row("one", old.id, [alpha.id, gamma.id]), row("three", null, [])],
    {},
    "-module_name"
  );
  expect(issueLists().map((call) => call.query.order_by)).toEqual(["-module_name"]);
  expect(rowNames()).toEqual(["two", "one", "three"]);
  expect(
    ["common.modules", "common.cycle", "common.labels"].map((title) => header(title)!.getAttribute("aria-sort"))
  ).toEqual(["descending", "none", "none"]);
  expect(preferenceWrites()).toEqual([]);
  expect(toasts).toEqual([]);
});
