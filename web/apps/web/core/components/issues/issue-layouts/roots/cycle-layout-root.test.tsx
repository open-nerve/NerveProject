// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A cycle's page's work items (P5b spec 2.1, 2.4, 2.6; re-66 H1, L4): its project's list with the cycle, under the
// page's own preferences and in the layout they choose, by the cycle's store type, with the page's own preview; why
// nerve could not read the preferences, with a retry; the cycles' turned-off state, reading nothing, while the
// project's cycles are off; a change of its filters followed as the project page's: nerve confirms it, then the list is
// read again with the pages it has.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { C1, buttonOf, planningPageTab, cyclePreferencesOf, layouts, type Layout } from "./fake-planning-page";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  peek: undefined as { hostScope: string; store: unknown } | undefined,
}));
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
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; rows, cards, days and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// The page's preview: a stand-in that says the host it was given and the store type it is under.
vi.mock("../../peek-overview", async () => {
  const { useContext } = await import("react");
  const { IssuesStoreContext } = await import("@/hooks/use-issue-layout-store");
  return {
    IssuePeekOverview: (props: { hostScope: string }) => {
      bound.peek = { hostScope: props.hostScope, store: useContext(IssuesStoreContext) };
      return null;
    },
  };
});

let page: Awaited<ReturnType<typeof planningPageTab>>;
/** A tab at the cycle's page of project A, as planningPageTab, whose stores the page uses. */
async function opened(fields: Parameters<typeof planningPageTab>[1] = {}) {
  page = await planningPageTab(EIssuesStoreType.CYCLE, fields);
  bound.app = page.app;
}
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  bound.peek = undefined;
});
afterEach(async () => page.close());

const text = () => document.body.textContent ?? "";
/** The layout the page shows, by what only it draws. */
function shown(): Layout | undefined {
  if (document.querySelector("table")) return "spreadsheet";
  if (document.querySelector("button[aria-pressed]")) return "calendar";
  if (document.querySelector(".horizontal-scrollbar")) return "kanban";
  if (document.querySelector('a[id="issue-first"]')) return "list";
}

it.each(Object.keys(layouts) as Layout[])(
  "shows the cycle's work items in the %s layout its page's preferences choose, under the cycle's store type",
  async (layout) => {
    await opened();
    await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
    expect(page.preferences()).toHaveLength(1);
    expect(page.lists()).toHaveLength(1);
    expect(page.lists()[0]!.query).toMatchObject({ cycle_id: C1 });
    expect(shown()).toBe(layout);
    expect(bound.peek?.store).toBe(EIssuesStoreType.CYCLE);
  }
);

it("shows the conditions of the page's filters, and opens its preview in the page's own host, the cycle's", async () => {
  await opened();
  await page.show(cyclePreferencesOf({}, { priority: ["high"] }), layouts.list.page());
  expect(page.lists()[0]!.query).toMatchObject({ cycle_id: C1, priority: "high" });
  expect(buttonOf("Clear all"), "the row of the page's conditions").toBeDefined();
  expect(JSON.parse(bound.peek!.hostScope)).toEqual(["cycle", "ws", "A", C1]);
});

it("says why nerve could not read the page's preferences, and reads them again on retry", async () => {
  await opened();
  await page.mount();
  await until(() => page.preferences().length === 1, "the page's preferences");
  await act(async () => page.preferences()[0]!.answer(problem(503, "server_busy")));
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(page.lists()).toEqual([]);
  await act(async () => buttonOf("common.retry")?.click());
  await until(() => page.preferences().length === 2, "the preferences again");
  await act(async () => page.preferences()[1]!.answer(json(200, cyclePreferencesOf())));
  await until(() => page.lists().length === 1, "the page's list");
  expect(document.querySelector('[role="alert"]')).toBeNull();
});

it("is the cycles' turned-off state while its project's cycles are off, and reads neither preferences nor list", async () => {
  await opened({ project: { cycle_view: false } });
  await page.mount();
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(text()).toContain("disabled_project.empty_state.cycle.title");
  expect(page.preferences()).toEqual([]);
  expect(page.lists()).toEqual([]);
  expect(page.nerve.calls).toEqual([]);
});

it("clears its filters from its empty state, and once nerve confirms reads the cycle's list again with its pages", async () => {
  await opened();
  await page.show(cyclePreferencesOf(layouts.list.display, { priority: ["high"] }), layouts.list.page(false));
  expect(page.lists()[0]!.query).toMatchObject({ cycle_id: C1, priority: "high", limit: "50" });
  expect(text()).toContain("common_empty_state.search.title");
  await act(async () => buttonOf("project_issues.empty_state.issues_empty_filter.secondary_button.text")?.click());
  await until(() => page.preferences().length === 2, "the change of the page's filters");
  expect(page.preferences()[1]).toMatchObject({ method: "PATCH", body: { filters: {} } });
  expect(page.lists()).toHaveLength(1);
  await act(async () => page.preferences()[1]!.answer(json(200, cyclePreferencesOf(layouts.list.display))));
  await until(() => page.lists().length === 2, "the list again");
  expect(page.lists()[1]!.query).toMatchObject({ cycle_id: C1, limit: "50" });
  expect(page.lists()[1]!.query.priority).toBeUndefined();
});
