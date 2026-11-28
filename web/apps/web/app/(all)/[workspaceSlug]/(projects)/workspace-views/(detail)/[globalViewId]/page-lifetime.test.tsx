// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's own view's page over its lifetime (M7/P6b spec 2.8, R11; P6a review m1, re-review n1; M3 handoff 1):
// each entry reads the view itself, though SWR (one cache for the tab, as the app's) holds the entry before's answer;
// an earlier entry's read nerve answers late changes nothing of the page shown now; the page draws nothing of its view
// only while its own host is leaving after deleting it from its header, and says the view does not exist once nerve
// deleted it at another's asking; another view mounts the page afresh (its layout is keyed by the route's view), so
// nothing of the view before (a dialog, its draft) follows. The stores, the layouts, SWR and HTTP are real
// (fake-view-page.tsx), at the app's routes; the preview a marker, the app's header a frame that shows the header.
import { act, Profiler, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { dialog, dialogButton, settle, where } from "@/components/common/fake-quick-actions";
import { WorkspaceViewRoutes, shownNow, workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { json, noContent } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import Layout from "../layout";
import Page from "./page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => <i data-peek="" /> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
let first: View;
const consoleErrors = () => vi.mocked(console.error).mock.calls;
beforeEach(() => {
  toasts.length = 0;
  vi.spyOn(console, "error");
});
afterEach(async () => page.close());

/** What the page's body has drawn, in order, as React commits it (shownNow). */
const drawn: string[] = [];
function drawing() {
  const now = shownNow();
  if (drawn.at(-1) !== now) drawn.push(now);
}
/** A tab of ws with the caller its admin, at the page of its view First; SWR's cache is the tab's. */
async function opened() {
  page = await workspaceViewPageTab(20);
  bound.app = page.app;
  drawn.length = 0;
  first = page.view("First", { filters: { priority: ["high"] } });
  await page.render(
    <Profiler id="the page" onRender={drawing}>
      <WorkspaceViewRoutes path={`/ws/workspace-views/${first.id}`} Layout={Layout} Page={Page} cache={new Map()} />
    </Profiler>
  );
}
const text = () => document.body.textContent ?? "";
/** Leaves for the workspace's views, then goes Back to the view's page at once, as a person may. */
async function leftAndBack() {
  // what follows nerve's last answer settles first (SWR forgets a list's read once it answered, in a timer)
  await act(async () => vi.advanceTimersByTimeAsync(100));
  await act(async () => void where.navigate?.("/ws/workspace-views"));
  expect(text()).toContain("the list");
  await act(async () => void where.navigate?.(-1));
}
/** Opens the header's quick actions of the view, and chooses its item of title. */
async function choose(title: string) {
  const menu = document.querySelector<HTMLButtonElement>('header button[aria-haspopup="menu"]');
  await act(async () => menu?.click());
  await settle();
  const item = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find(
    (node) => node.querySelector("h5")?.textContent === title
  );
  await act(async () => item?.click());
  await settle();
}

it("reads its view anew as it is entered again at once after nerve answered: one read more", async () => {
  await opened();
  await page.answerRead(first);
  await page.answerList();
  // within SWR's dedupe window of that answer
  await leftAndBack();
  await until(() => page.reads(first).length === 2, "the view's read again");
  expect(shownNow()).toBe("reading");
  await page.answerRead(first);
  await until(() => page.lists().length === 2, "the list again");
  await page.answerList();
  expect(shownNow()).toBe("the view");
  expect(page.reads(first)).toHaveLength(2);
});

it("stays as it is when nerve answers the entry before's read after this entry's own: no failure, no toast", async () => {
  await opened();
  await until(() => page.reads(first).length === 1, "the first entry's read");
  await leftAndBack();
  await until(() => page.reads(first).length === 2, "this entry's read");
  // nerve answers this entry's read, then the one before's (two reads need not be answered in the order sent)
  await act(async () => page.reads(first)[1]!.answer(json(200, first)));
  await page.answerList();
  expect(shownNow()).toBe("the view");
  drawn.length = 0;
  await act(async () => page.reads(first)[0]!.answer(json(200, first)));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(shownNow()).toBe("the view");
  expect(
    drawn.filter((one) => one !== "the view"),
    "what the page drew of the late answer"
  ).toEqual([]);
  expect(page.reads(first)).toHaveLength(2);
  expect(toasts).toEqual([]);
  expect(consoleErrors()).toEqual([]);
});

it("goes to the workspace's views once nerve deletes the view from its header, drawing nothing of it before", async () => {
  await opened();
  await page.answerRead(first);
  await page.answerList();
  await choose("delete");
  await act(async () => dialogButton("delete")?.click());
  await until(() => page.writes().length === 1, "the deletion");
  expect(page.writes()[0]).toMatchObject({ method: "DELETE", path: `/api/v0/views/${first.id}` });
  drawn.length = 0;
  await act(async () => page.writes()[0]!.answer(noContent()));
  await settle();
  expect(where.location?.pathname).toBe("/ws/workspace-views");
  expect(text()).toContain("the list");
  expect(drawn, "what the page drew before it went").not.toContain("not found");
  expect(consoleErrors()).toEqual([]);
});

it("says the view does not exist, with the way back, once nerve deletes it at another's asking (re-review n1)", async () => {
  await opened();
  await page.answerRead(first);
  await page.answerList();
  expect(shownNow()).toBe("the view");
  // nerve deletes the view as another page asked (one left by a Back): not this page, which stays
  let deleted: Promise<void> | undefined;
  await act(async () => void (deleted = page.app.views.remove(first.id)));
  await until(() => page.writes().length === 1, "the deletion");
  await act(async () => page.writes()[0]!.answer(noContent()));
  await act(async () => deleted);
  await settle();
  expect(where.location?.pathname).toBe(`/ws/workspace-views/${first.id}`);
  expect(drawn.at(-1), "what the page drew as nerve deleted the view").toBe("not found");
  expect(toasts).toEqual([]);
  expect(consoleErrors()).toEqual([]);
  await act(async () =>
    [...document.querySelectorAll("button")].find((b) => b.textContent === "workspace_views.not_found.action")?.click()
  );
  expect(where.location?.pathname).toBe("/ws/workspace-views");
});

it("mounts afresh for another view: the view before's dialog and draft do not follow, nor is the other ever not found", async () => {
  await opened();
  const second = page.view("Second");
  await page.answerRead(first);
  await page.answerList();
  await choose("delete");
  expect(dialog(), "First's deletion, asked for").not.toBeNull();
  drawn.length = 0;
  await act(async () => void where.navigate?.(`/ws/workspace-views/${second.id}`));
  expect(dialog(), "First's deletion, on Second's page").toBeNull();
  await page.answerRead(second);
  await until(() => page.lists().length === 2, "Second's list");
  await page.answerList();
  expect(shownNow()).toBe("the view");
  expect(drawn, "what the page drew as it went to Second").not.toContain("not found");
  expect(page.app.issue.globalIssuesFilter.getIssueFilters(first.id), "First's draft").toBeUndefined();
  expect(page.writes()).toEqual([]);
});
