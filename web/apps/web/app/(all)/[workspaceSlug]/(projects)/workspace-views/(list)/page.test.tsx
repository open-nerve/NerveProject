// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The workspace's views list page (M7/P6b spec 2.7, 2.10, R12, R13): titled by the workspace; the four default rows,
// each a link to its page; for the workspace's admins and members (workspaceViewsOn) its own views after them, read
// once, waited for, and, nerve failing, why, with a retry; none of them is no empty state; its guests read none. Its
// search, named, keeps the rows whose names hold it whatever its case and its spaces around (a default view's by its
// name's translation), says when none does, and is the page's: another workspace opens its list afresh. The stores,
// the layout, SWR and HTTP are real (fake-list-page.tsx), at the app's routes; the app's header a frame that shows the
// header, the list's loader a marker. The rows' menus are page-rows.test.tsx's.
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Workspace, WorkspaceRole } from "@nerve/api-client";
import { where } from "@/components/common/fake-quick-actions";
import { problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import {
  addView,
  answerViews,
  DEFAULT_KEYS,
  DEFAULTS,
  listTab,
  loading,
  rows,
  search,
  searchBox,
} from "./fake-list-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));
vi.mock("@/components/ui/loader/view-list-loader", () => ({
  ViewListLoader: () => <p role="status">views loading</p>,
}));

let tab: Awaited<ReturnType<typeof listTab>>;
beforeEach(() => {
  document.title = "";
});
afterEach(async () => tab.close());

/** A tab of ws whose caller has role there, the workspaces others too, at ws's views list. */
async function opened(role: WorkspaceRole, others: Workspace[] = []) {
  tab = await listTab(role, others);
  bound.app = tab.app;
  await tab.open();
}
const names = () => rows().map(([name]) => name);
const alert = () => document.querySelector('[role="alert"]');
const text = () => document.body.textContent ?? "";

it.each([
  ["its admin", 20],
  ["its member", 15],
] as const)(
  "shows %s (role %i) the four default rows, then the workspace's own views, read once, waited for",
  async (_, role) => {
    await opened(role);
    expect(document.title).toBe("ws - views");
    expect(rows()).toEqual(DEFAULTS.map((name, i) => [name, `/ws/workspace-views/${DEFAULT_KEYS[i]}`]));
    expect(loading()).toBe(true);
    expect(addView() !== undefined).toBe(true);
    await answerViews(tab);
    expect(loading()).toBe(false);
    expect(rows().slice(4)).toEqual([
      ["First", `/ws/workspace-views/${tab.first.id}`],
      ["Second", `/ws/workspace-views/${tab.second.id}`],
    ]);
    expect(tab.viewLists()).toHaveLength(1);
  }
);

it("shows its guest the four default rows alone, reading none of its own views, and none matching his search", async () => {
  await opened(5);
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(document.title).toBe("ws - views");
  expect(names()).toEqual(DEFAULTS);
  expect([loading(), alert(), addView()]).toEqual([false, null, undefined]);
  expect(tab.viewLists()).toEqual([]);
  await search("first");
  expect(rows()).toEqual([]);
  expect(text()).toContain("common.search.no_matches_found");
});

it("says why nerve could not read the workspace's views, under the default rows, and reads them again on retry", async () => {
  await opened(15);
  await until(() => tab.viewLists().length === 1, "the workspace's views");
  await act(async () => tab.viewLists()[0]!.answer(problem(503, "server_busy")));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(names()).toEqual(DEFAULTS);
  expect([loading(), alert()?.textContent?.includes("common.retry")]).toEqual([false, true]);
  const retry = [...(alert()?.querySelectorAll("button") ?? [])].find((b) => b.textContent === "common.retry");
  await act(async () => retry?.click());
  await answerViews(tab, [tab.first]);
  expect(names()).toEqual([...DEFAULTS, "First"]);
  expect([loading(), alert()]).toEqual([false, null]);
  expect(tab.viewLists()).toHaveLength(2);
});

it("shows the four default rows alone when the workspace has no views of its own, no empty state", async () => {
  await opened(15);
  await answerViews(tab, []);
  expect(names()).toEqual(DEFAULTS);
  expect([loading(), alert()]).toEqual([false, null]);
  expect(text()).not.toContain("common.search.no_matches_found");
  expect(text()).not.toContain("empty_state");
});

it("keeps the rows whose names hold its search, whatever its case, a default row by its name's translation", async () => {
  await opened(15);
  await answerViews(tab);
  expect(searchBox()?.getAttribute("placeholder")).toBe("common.search.label");
  await search("ASSIGNED");
  expect(names()).toEqual(["default_global_view.assigned"]);
  await search("first");
  expect(names()).toEqual(["First"]);
  await search("  SECOND ");
  expect(names()).toEqual(["Second"]);
  await search("nothing like it");
  expect(rows()).toEqual([]);
  expect(text()).toContain("common.search.no_matches_found");
  await search("");
  expect(names()).toEqual([...DEFAULTS, "First", "Second"]);
  expect(text()).not.toContain("common.search.no_matches_found");
});

it("keeps its search to the workspace: another workspace's list, and the first's again, open with none", async () => {
  await opened(15, [workspaceOf("other")]);
  await answerViews(tab);
  await search("first");
  expect(names()).toEqual(["First"]);
  await act(async () => where.navigate?.("/other/workspace-views"));
  await answerViews(tab, [], "other");
  expect(searchBox()?.value).toBe("");
  expect(rows()).toEqual(DEFAULTS.map((name, i) => [name, `/other/workspace-views/${DEFAULT_KEYS[i]}`]));
  await act(async () => where.navigate?.("/ws/workspace-views"));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(searchBox()?.value).toBe("");
  expect(names()).toEqual([...DEFAULTS, "First", "Second"]);
});
