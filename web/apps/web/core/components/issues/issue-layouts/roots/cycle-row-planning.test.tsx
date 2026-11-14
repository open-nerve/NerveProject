// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row's cycle on a cycle's page (M6 P7a spec 2.5, P5b spec 2.6, R10): moved to another cycle from its own control,
// in the list and on the board, it is one PATCH of the cycle alone; once nerve confirms it the row leaves the page and
// the page reads its cycle's progress again, as the page's other writes are followed. The layouts, rows, dropdowns,
// stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { C1, planningPageTab, cyclePreferencesOf, layouts } from "./fake-planning-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
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
// jsdom has no viewport geometry.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));

const C2 = "019a0000-0000-7000-8000-0000000000c2";
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", cycleId: C1 };
  toasts.length = 0;
});
afterEach(async () => page.close());

/** The row's cycle control, named by its row (R20). */
const cycleControl = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.startsWith("common.cycle: ") && button.textContent.endsWith(", first")
  );
const row = () => document.querySelector('[id^="issue-first"]');
const writes = () => page.nerve.to("/api/v0/issues/first").filter((call) => call.method === "PATCH");
const progressReads = () => page.nerve.to(cycleRequests.progress({ id: C1 }));

it.each(["list", "kanban"] as const)(
  "moves a row to another cycle in one PATCH from the %s; once nerve confirms, the row leaves and the progress is read again",
  async (layout) => {
    page = await planningPageTab(EIssuesStoreType.CYCLE);
    bound.app = page.app;
    await page.show(cyclePreferencesOf(layouts[layout].display), layouts[layout].page());
    expect(cycleControl()?.textContent).toBe("common.cycle: Now, first");
    await act(async () => cycleControl()!.click());
    const options = () => page.nerve.to(cycleRequests.list(page.project));
    await act(async () => until(() => options().length === 1, "the cycles' read"));
    const now = cycleOf(page.project, "Now", { id: C1, status: "current" });
    const next = cycleOf(page.project, "Next", { id: C2, status: "upcoming" });
    await act(async () => {
      options()[0]!.answer(json(200, { data: [rowOf(now), rowOf(next)] }));
      await vi.advanceTimersByTimeAsync(20);
    });
    const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
      (item) => item.textContent === "Next"
    );
    await act(async () => {
      option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(writes().map((call) => call.body)).toEqual([{ cycle_id: C2 }]);
    expect(progressReads()).toEqual([]);
    const moved = {
      ...issueDetailOf("first"),
      cycle_id: C2,
      state_id: "state",
      priority: "high",
      target_date: "2026-12-31",
      sort_order: 1000,
      updated_at: "2026-12-31T13:00:00Z",
    };
    await act(async () => writes()[0]!.answer(json(200, moved)));
    await until(() => row() === null, "the row off the page");
    await until(() => progressReads().length === 1, "the progress again");
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(progressReads()).toHaveLength(1);
    expect(writes()).toHaveLength(1);
    expect(toasts).toEqual([]);
  }
);
