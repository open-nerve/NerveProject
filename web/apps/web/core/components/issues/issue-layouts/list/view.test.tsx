// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueListPage } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { labelOf, preferencesOf } from "@/store/project/fake-projects";
import { ProjectLayoutRoot } from "../roots/project-layout-root";
const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
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
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry. Keep real rows, properties, headers, pagination and HTTP.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
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
const a = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const c = "cccccccc-cccc-4ccc-8ccc-cccccccccccc";
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const button = (name: string) => [...host.querySelectorAll("button")].find((node) => node.textContent?.trim() === name);
const first = listIssue("first", 10, { label_ids: [a, c], sequence_id: 9 });
const second = listIssue("second", 20, { label_ids: [a, c], sequence_id: 10000 });
const other = listIssue("other-group", 30, { label_ids: [a] });
const grouped = (): IssueListPage => ({
  total: 3,
  groups: [
    { key: a, total: 3, data: [first, other], next_cursor: "A:more" },
    { key: c, total: 2, data: [first], next_cursor: "C:more" },
    { key: "none", total: 0, data: [], next_cursor: null },
  ],
});
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("IntersectionObserver", VisibleObserver);
  intersections.clear();
  toasts.length = 0;
  tab = await issueStoreTab();
  bound.app = tab.app;
  await answered(
    tab.nerve,
    () => tab.app.label.fetchProjectLabels("A"),
    ["GET", "/api/v0/projects/A/labels"],
    {
      data: [
        { ...labelOf(tab.project, "Label A", 1), id: a },
        { ...labelOf(tab.project, "Label C", 2), id: c },
      ],
    },
    "labels"
  );
  tab.nerve.calls.length = 0;
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
async function mount(page: IssueListPage, groupedView = true, empty = false) {
  const prefs = preferencesOf({ filters: { label_id: [a] } });
  prefs.display_filters.group_by = groupedView ? "label_id" : null;
  prefs.display_filters.order_by = "sort_order";
  prefs.display_filters.show_empty_groups = empty;
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter>
            <ProjectLayoutRoot />
          </MemoryRouter>
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "first page");
  await act(async () => lists()[0]!.answer(json(200, page)));
  return prefs;
}
function section(name: string) {
  const header = [...host.querySelectorAll(".group\\/list-header")].find((node) => node.textContent?.startsWith(name));
  expect(header).toBeDefined();
  return header!.parentElement!.parentElement!;
}
it("real list continuation keeps A intersection C, retries only C, preserves other groups and authoritative totals", async () => {
  await mount(grouped());
  const group = section("Label C");
  expect(group.textContent).toContain("Label C2"); // 2 on server; only 1 row loaded.
  const more =
    group.querySelector<HTMLButtonElement>("button[type=button]") &&
    [...group.querySelectorAll("button")].find((node) => node.textContent?.includes("common.load_more"));
  expect(more, "named keyboard-accessible continuation").toBeDefined();
  await act(async () => {
    more!.focus();
    more!.click();
    more!.click();
  });
  expect(document.activeElement).toBe(more);
  await until(() => lists().length >= 2, "C continuation");
  expect(lists()).toHaveLength(2);
  expect(lists()[1]!.query).toEqual({
    label_id: a,
    group_by: "label_id",
    group_key: c,
    order_by: "sort_order",
    sub_issue: "true",
    limit: "50",
    cursor: "C:more",
  });
  expect(more!.disabled).toBe(true);
  await act(async () => lists()[1]!.answer(problem(503, "internal_error")));
  expect(toasts).toHaveLength(1);
  expect(section("Label A").textContent).toContain("other-group");
  const retry = button("common.retry")!;
  expect(retry).toBeDefined();
  await act(async () => {
    retry.click();
    retry.click();
  });
  await until(() => lists().length >= 3, "C retry");
  expect(lists()).toHaveLength(3);
  expect(lists()[2]!.query).toEqual(lists()[1]!.query);
  await act(async () => lists()[2]!.answer(json(200, { total: 2, data: [first, second], next_cursor: null })));
  expect(button("common.retry")).toBeUndefined();
  expect(section("Label C").textContent).toContain("second");
  expect([...section("Label C").querySelectorAll("p")].filter((node) => node.textContent === "first")).toHaveLength(1);
  expect(section("Label C").textContent).toContain("Label C2");
  expect(section("Label C").textContent).not.toContain("common.load_more");
  expect(section("Label A").textContent).toContain("other-group");
  expect(tab.app.issue.issues.getIssueById("first")?.label_ids).toEqual([a, c]);
  const widths = [...host.querySelectorAll<HTMLElement>('[style*="min-width"]')].map((node) => node.style.minWidth);
  expect(widths).toEqual(["49px", "49px", "49px", "49px"]);
});
it("ungrouped failed continuation stays quiet until explicit retry and ends at a null cursor", async () => {
  await mount({ total: 2, data: [first], next_cursor: "flat:more" }, false);
  expect(button("common.load_more ↓")).toBeDefined();
  const [target, callback] = [...intersections][0]!;
  // This is the actual sentinel observer callback; jsdom cannot calculate intersections.
  await act(async () =>
    callback([{ isIntersecting: true, target } as IntersectionObserverEntry], {} as IntersectionObserver)
  );
  await until(() => lists().length === 2, "flat continuation");
  await act(async () => lists()[1]!.answer(problem(503, "internal_error")));
  expect(intersections.size, "a refusal detaches automatic continuation").toBe(0);
  expect(lists()).toHaveLength(2);
  await act(async () => button("common.retry")!.click());
  await until(() => lists().length === 3, "flat retry");
  await act(async () => lists()[2]!.answer(json(200, { total: 2, data: [first, second], next_cursor: null })));
  expect(intersections.size).toBe(0);
  expect(button("common.load_more ↓")).toBeUndefined();
  expect(host.textContent).toContain("second");
  expect(lists()).toHaveLength(3);
});
it.each([false, true])("empty group visibility follows confirmed preference: %s", async (empty) => {
  await mount(grouped(), true, empty);
  const titles = [...host.querySelectorAll(".group\\/list-header")].map((node) => node.textContent);
  expect(titles.some((name) => name?.includes("None"))).toBe(empty);
  expect(lists()).toHaveLength(1);
});
