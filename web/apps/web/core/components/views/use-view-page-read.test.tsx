// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A view's page's read of its view (P6a spec 2.6, R2; re-96 I1, I2): each mount makes its own, as it mounts, though SWR
// still holds an earlier mount's answer within its dedupe window; it ends the page's draft as it unmounts, so that
// nothing of the draft shows before the next mount's read answers; a read nerve answers after the page left writes
// nothing, even after the next mount's own answered (review m1); a reconnect reads nothing. The hook, the session's
// stores, SWR (one cache, as the app's) and HTTP are real.
import { act } from "react";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { viewReadTab } from "@/components/views/fake-view-page";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toProjectFilters } from "@/lib/issue-filters";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import { useViewPageRead } from "./use-view-page-read";

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

let page: Awaited<ReturnType<typeof viewReadTab>>;
let view: View;
// one SWR cache for the whole tab, as the app has
let cache = new Map();
beforeEach(async () => {
  page = await viewReadTab();
  bound.app = page.app;
  view = page.view("First", { filters: { priority: ["high"] } });
  cache = new Map();
  rendered = [];
  toasts.length = 0;
  vi.spyOn(console, "error");
});
afterEach(async () => page.close());

/** Each state the page has rendered, in order. */
let rendered: string[] = [];
/** The page's read of the view: where the page is with it, as an observer page renders it. */
const Probe = observer(function Probe() {
  const { state } = useViewPageRead(EIssuesStoreType.PROJECT_VIEW, "ws", view.id);
  rendered.push(state);
  return <p>{state}</p>;
});
const mount = () =>
  page.render(
    <SWRConfig value={{ provider: () => cache }}>
      <Probe />
    </SWRConfig>
  );
const unmount = () => page.render(<SWRConfig value={{ provider: () => cache }}>{null}</SWRConfig>);
const state = () => document.querySelector("p")?.textContent;
const draft = () => page.app.issue.projectViewIssuesFilter.getIssueFilters(view.id);
/** The conditions of the page's draft, in nerve's words. */
const conditions = () => toProjectFilters(draft()?.richFilters ?? {});

it("reads its view as it mounts, and again as it mounts right after nerve answered, leaving reading each time", async () => {
  await mount();
  await until(() => page.reads(view).length === 1, "the view's read");
  expect(state()).toBe("reading");
  await page.answerRead(view);
  expect(state()).toBe("read");
  // at once, within SWR's dedupe window of that answer
  await unmount();
  await mount();
  await until(() => page.reads(view).length === 2, "the view's read again");
  expect(state()).toBe("reading");
  await page.answerRead(view);
  expect(state()).toBe("read");
  expect(page.reads(view)).toHaveLength(2);
});

it("ends the page's draft as it unmounts: entering again shows nothing of it until nerve answers anew", async () => {
  await mount();
  await page.answerRead(view);
  expect(conditions()).toEqual({ priority: ["high"] });
  await unmount();
  expect(draft()).toBeUndefined();
  await mount();
  expect(state()).toBe("reading");
  expect(draft()).toBeUndefined();
  await page.answerRead({ ...view, filters: { priority: ["low"] } });
  expect(conditions()).toEqual({ priority: ["low"] });
});

it("reads nothing as the tab reconnects: the page stays as it is, its draft its own", async () => {
  await mount();
  await page.answerRead(view);
  await act(async () => vi.advanceTimersByTimeAsync(5000));
  rendered = [];
  // past SWR's dedupe window, the tab goes offline and back online
  await act(async () => {
    window.dispatchEvent(new Event("offline"));
    window.dispatchEvent(new Event("online"));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(page.reads(view)).toHaveLength(1);
  expect(
    rendered.filter((one) => one !== "read"),
    "a read again, of the draft"
  ).toEqual([]);
  expect(state()).toBe("read");
});

it("stays as it is when nerve answers an earlier mount's read after this mount's own, which answered first", async () => {
  await mount();
  await until(() => page.reads(view).length === 1, "the first mount's read");
  await unmount();
  await mount();
  await until(() => page.reads(view).length === 2, "this mount's read");
  // nerve answers this mount's read, then the earlier mount's (two reads need not be answered in the order sent)
  await act(async () => page.reads(view)[1]!.answer(json(200, view)));
  expect(state()).toBe("read");
  rendered = [];
  await act(async () => page.reads(view)[0]!.answer(json(200, view)));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(state()).toBe("read");
  expect(
    rendered.filter((one) => one !== "read"),
    "what the page drew of the earlier mount's answer"
  ).toEqual([]);
  expect(conditions()).toEqual({ priority: ["high"] });
  expect(page.reads(view)).toHaveLength(2);
  expect(toasts).toEqual([]);
  expect(vi.mocked(console.error).mock.calls).toEqual([]);
});

it("writes nothing of a read nerve answers after the page left, and the next mount reads anew", async () => {
  await mount();
  await until(() => page.reads(view).length === 1, "the view's read");
  await unmount();
  await page.answerRead(view);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(draft()).toBeUndefined();
  expect(toasts).toEqual([]);
  expect(vi.mocked(console.error).mock.calls).toEqual([]);
  await mount();
  await until(() => page.reads(view).length === 2, "the view's read again");
  expect(state()).toBe("reading");
});
