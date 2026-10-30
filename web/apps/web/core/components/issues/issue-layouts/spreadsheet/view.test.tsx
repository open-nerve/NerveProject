// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode, createElement } from "react";
import { createRoot } from "react-dom/client";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueListPage, ProjectPreferences } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { preferencesOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import { ProjectLayoutRoot } from "../roots/project-layout-root";
import { SpreadsheetSubIssueColumn } from "./columns/sub-issue-column";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, projectId: "A" }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: bound.projectId }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as: string }) => createElement(as, null, children),
}));
const intersections = new Map<Element, IntersectionObserverCallback>();
class VisibleObserver {
  constructor(private callback: IntersectionObserverCallback) {}
  observe(element: Element) {
    intersections.set(element, this.callback);
  }
  unobserve(element: Element) {
    intersections.delete(element);
  }
  disconnect() {}
}
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const writes = () => tab.nerve.to("/api/v0/me/projects/A/preferences").filter((call) => call.method === "PATCH");
const button = (name: string) => [...host.querySelectorAll("button")].find((node) => node.textContent?.trim() === name);
const first = listIssue("first", 10, { sub_issues_count: 2, sequence_id: 9 });
const second = listIssue("second", 20, { sequence_id: 10 });
const rows = () =>
  [...host.querySelectorAll("tbody > tr > td:first-child .line-clamp-1 .truncate")].map((node) => node.textContent);
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("IntersectionObserver", VisibleObserver);
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  bound.projectId = "A";
  intersections.clear();
  toasts.length = 0;
  tab = await issueStoreTab();
  bound.app = tab.app;
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function render(children: ReactNode) {
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter>{children}</MemoryRouter>
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
}
async function mount(page: IssueListPage | Response) {
  const prefs = preferencesOf();
  prefs.display_filters.layout = "spreadsheet";
  prefs.display_filters.order_by = "sub_issues_count";
  for (const key of Object.keys(prefs.display_properties) as (keyof ProjectPreferences["display_properties"])[])
    prefs.display_properties[key] = key === "sub_issue_count" || key === "key";
  await render(<ProjectLayoutRoot />);
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "first page");
  await act(async () => lists()[0]!.answer(page instanceof Response ? page : json(200, page)));
  return prefs;
}
it("real table continuation retains order and cursor, deduplicates, detaches on refusal and stops at the last page", async () => {
  await mount({ total: 2, data: [second], next_cursor: "count:more" });
  const more = button("common.load_more");
  expect(more).toBeDefined();
  await act(async () => {
    more!.focus();
    more!.click();
    more!.click();
  });
  await until(() => lists().length === 2, "continuation");
  expect(document.activeElement).toBe(more);
  expect(more!.disabled).toBe(true);
  expect(lists()[1]!.query).toEqual({
    order_by: "sub_issues_count",
    sub_issue: "true",
    limit: "100",
    cursor: "count:more",
  });
  await act(async () => lists()[1]!.answer(problem(503, "internal_error")));
  expect(intersections.size).toBe(0);
  expect(toasts).toHaveLength(1);
  expect(host.textContent).toContain("second");
  await act(async () => {
    button("common.retry")!.click();
    button("common.retry")!.click();
  });
  await until(() => lists().length === 3, "retry");
  expect(lists()[2]!.query).toEqual(lists()[1]!.query);
  await act(async () => lists()[2]!.answer(json(200, { total: 2, data: [second, first], next_cursor: null })));
  expect(rows()).toEqual(["second", "first"]);
  expect(button("common.load_more")).toBeUndefined();
  expect(intersections.size).toBe(0);
});
it("the real sorting menu keeps confirmed preference, arrow and rows on refusal; only confirmation refreshes", async () => {
  const prefs = await mount({ total: 2, data: [second, first], next_cursor: null });
  const header = () => host.querySelector<HTMLTableCellElement>("th[aria-sort]")!;
  async function descending() {
    const trigger = header().querySelector("button")!;
    vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 180, 40));
    await act(async () => trigger.click());
    await act(async () => document.querySelector<HTMLButtonElement>('[role="menuitem"]')!.click());
  }
  expect(header().getAttribute("aria-sort")).toBe("ascending");
  await descending();
  await until(() => writes().length === 1, "sort write");
  expect(writes()[0]!.body).toEqual({ display_filters: { ...prefs.display_filters, order_by: "-sub_issues_count" } });
  expect(header().getAttribute("aria-sort")).toBe("ascending");
  expect(rows()).toEqual(["second", "first"]);
  await act(async () => writes()[0]!.answer(problem(422, "validation_failed")));
  expect(header().getAttribute("aria-sort")).toBe("ascending");
  expect(rows()).toEqual(["second", "first"]);
  expect(lists()).toHaveLength(1);
  expect(toasts).toHaveLength(1);
  await descending();
  await until(() => writes().length === 2, "retry write");
  prefs.display_filters.order_by = "-sub_issues_count";
  await act(async () => writes()[1]!.answer(json(200, prefs)));
  await until(() => lists().length === 2, "confirmed query");
  expect(lists()[1]!.query.order_by).toBe("-sub_issues_count");
  await act(async () => lists()[1]!.answer(json(200, { total: 2, data: [first, second], next_cursor: null })));
  expect(header().getAttribute("aria-sort")).toBe("descending");
  expect(rows()).toEqual(["first", "second"]);
});
it.each([null, "2026-10-30T00:00:00Z"])(
  "sub-items use a real project identifier and browse URL, archived=%s",
  async (archived_at) => {
    await render(<SpreadsheetSubIssueColumn issue={{ ...first, archived_at }} />);
    expect(host.querySelector("a")?.getAttribute("href")).toBe(`/ws/browse/${tab.project.identifier}-9`);
    expect(host.textContent).toBe("2 sub-work items");
    expect(tab.nerve.calls).toHaveLength(0);
  }
);
it.each([{ project_id: "cold" }, { sequence_id: 0 }, { sub_issues_count: 0 }])(
  "sub-items have no fabricated or empty link: %j",
  async (changes) => {
    await render(<SpreadsheetSubIssueColumn issue={{ ...first, ...changes }} />);
    expect(host.querySelector("a")).toBeNull();
    expect(host.querySelector("[tabindex]")).toBeNull();
    expect(tab.nerve.calls).toHaveLength(0);
  }
);

it("a failed initial read offers retry and an empty confirmed page has no continuation", async () => {
  await mount(problem(503, "internal_error"));
  expect(button("common.retry")).toBeDefined();
  await act(async () => button("common.retry")!.click());
  await until(() => lists().length === 2, "initial retry");
  expect(lists()[1]!.query).toEqual(lists()[0]!.query);
  await act(async () => lists()[1]!.answer(json(200, { total: 0, data: [], next_cursor: null })));
  expect(button("common.retry")).toBeUndefined();
  expect(button("common.load_more")).toBeUndefined();
});

it("another project in the same session and its preference reload drive their own sorting arrow", async () => {
  const projectB = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, projectB]);
  tab.nerve.calls.length = 0;
  const prefs = await mount({ total: 1, data: [first], next_cursor: null });
  expect(host.querySelector("th[aria-sort]")?.getAttribute("aria-sort")).toBe("ascending");
  bound.projectId = "B";
  await render(<ProjectLayoutRoot />);
  await act(async () => runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "B" })));
  const preferencesB = () => tab.nerve.to("/api/v0/me/projects/B/preferences");
  await until(() => preferencesB().length === 1, "B preference");
  prefs.display_filters.order_by = "-sub_issues_count";
  await act(async () => preferencesB()[0]!.answer(json(200, prefs)));
  const pagesB = () => tab.nerve.to("/api/v0/projects/B/issues");
  await until(() => pagesB().length === 1, "B page");
  await act(async () =>
    pagesB()[0]!.answer(json(200, { total: 1, data: [{ ...first, id: "b", project_id: "B" }], next_cursor: null }))
  );
  expect(host.querySelector("th[aria-sort]")?.getAttribute("aria-sort")).toBe("descending");
  await act(async () => view.unmount());
  view = createRoot(host);
  await render(<ProjectLayoutRoot />);
  await until(() => preferencesB().length === 2, "reload B preference");
  await act(async () => preferencesB()[1]!.answer(json(200, prefs)));
  await until(() => pagesB().length === 2, "reload B page");
  await act(async () =>
    pagesB()[1]!.answer(json(200, { total: 1, data: [{ ...first, id: "b", project_id: "B" }], next_cursor: null }))
  );
  expect(host.querySelector("th[aria-sort]")?.getAttribute("aria-sort")).toBe("descending");
  expect(pagesB().map((call) => call.query.order_by)).toEqual(["-sub_issues_count", "-sub_issues_count"]);
  expect(writes()).toHaveLength(0);
});
