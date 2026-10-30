// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import type { IssueListPage } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueOf } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ProjectLayoutRoot } from "../roots/project-layout-root";
type Drop = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements>[0];
type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  drops: new Map<Element, Drop>(),
  drags: new Map<Element, Drag>(),
  scrolls: new Set<Element>(),
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
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// The DOM has no drag transport or viewport; all dates, cards, queries and confirmation consumers are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({
  autoScrollForElements: ({ element }: { element: Element }) => {
    bound.scrolls.add(element);
    return () => bound.scrolls.delete(element);
  },
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  dropTargetForElements: (options: Drop) => {
    bound.drops.set(options.element, options);
    return () => bound.drops.delete(options.element);
  },
  draggable: (options: Drag) => {
    bound.drags.set(options.element, options);
    return () => bound.drags.delete(options.element);
  },
}));
export let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
export const sourceDate = "2026-12-31",
  destinationDate = "2027-01-01";
export const first = {
  ...issueOf("first"),
  target_date: sourceDate,
  start_date: "2026-12-30",
  priority: "urgent" as const,
};
export const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
export const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
export const page = (): IssueListPage => ({
  total: 1,
  groups: [
    { key: sourceDate, total: 1, data: [first], next_cursor: null },
    { key: destinationDate, total: 0, data: [], next_cursor: null },
  ],
});
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  vi.setSystemTime(new Date(2026, 11, 31, 12));
  tab.app.issue.issueCalendarView.updateCalendarFilters({
    activeMonthDate: new Date(2026, 11, 31),
    activeWeekDate: new Date(2026, 11, 31),
  });
  bound.scrolls.clear();
  bound.drops.clear();
  bound.drags.clear();
  toasts.length = 0;
});
afterEach(async () => tab.close());
export async function mount(layout: "month" | "week" = "month", answer = true) {
  const prefs = preferencesOf({ filters: { priority: ["urgent"], target_date: ["2026-12-15.."] } });
  prefs.display_filters.layout = "calendar";
  prefs.display_filters.calendar = { layout, show_weekends: true };
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
  await until(() => tab.nerve.calls.length === 1, "calendar prefs");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "calendar read");
  if (answer) await act(async () => lists()[0]!.answer(json(200, page())));
}
export function tile(day: string) {
  const target = [...bound.drops.values()].find((item) => item.getData?.({} as never).date === day);
  expect(target, `real day ${day}`).toBeDefined();
  return target!;
}
export const cards = (day: string) => [...tile(day).element.querySelectorAll('a[id^="issue-"]')].map((node) => node.id);
export async function drop(day = destinationDate) {
  const source = [...bound.drags.values()].find((item) => item.getInitialData?.({} as never).id === first.id)!;
  const target = tile(day);
  await act(async () =>
    target.onDrop!({
      source: { element: source.element, data: source.getInitialData!({} as never) },
      self: { element: target.element, data: target.getData!({} as never) },
    } as never)
  );
}

export const autoScrollCount = () => bound.scrolls.size;
