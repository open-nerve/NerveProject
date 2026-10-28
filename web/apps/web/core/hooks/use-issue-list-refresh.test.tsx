// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab, listIssue, requestedIssue } from "@/store/issue/fake-issues";
import type { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";
import { useIssueListRefresh } from "./use-issue-list-refresh";
const binding = vi.hoisted(() => ({
  app: undefined as unknown,
  toast: vi.fn(),
  session: { loginId: "A" },
  retry: undefined as undefined | (() => Promise<void>),
}));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: binding.session }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => ({ setToast: binding.toast, TOAST_TYPE: { ERROR: "error" } }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
const page = {
  total: 2,
  groups: [
    { key: "A", total: 1, data: [listIssue("a", 10, { state_id: "A" })], next_cursor: "A:opaque" },
    { key: "B", total: 1, data: [listIssue("b", 10, { state_id: "B" })], next_cursor: null },
  ],
};
const Mounted = observer(function Mounted({ list, shown }: { list: IssueListBinding; shown: string[] }) {
  const { retry } = useIssueListRefresh(
    list,
    shown.map((group) => ({ group }))
  );
  binding.retry = retry;
  return <button onClick={() => void retry()}>retry</button>;
});
const mount = async (list: IssueListBinding, shown = ["A"]) => {
  await act(async () => view.render(<Mounted list={list} shown={shown} />));
};
const requestCount = async (count: number) => {
  await act(async () => until(() => tab.nerve.calls.length >= count, "visible stale HTTP"));
  expect(tab.nerve.calls).toHaveLength(count);
};
const answer = async (index: number, status = 200) => {
  await act(async () => {
    tab.nerve.calls[index]!.answer(
      status === 200
        ? json(200, {
            total: 1,
            data: [listIssue(`fresh-${index}`, 10, { state_id: tab.nerve.calls[index]!.query.group_key })],
            next_cursor: null,
          })
        : json(status, { code: "forbidden" })
    );
    await vi.advanceTimersByTimeAsync(20);
  });
};
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
  binding.toast.mockClear();
  binding.session.loginId = "A";
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10, query: { group_by: "state_id" } }),
    ["GET", "/api/v0/projects/A/issues"],
    page,
    "initial page"
  );
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.useRealTimers();
});

it("visible stale is fetched once and hidden group waits until expansion", async () => {
  const list = tab.app.issue.projectIssues.currentList!;
  list.markMembershipStale(listIssue("a", 10, { state_id: "A" }), listIssue("b", 10, { state_id: "B" }));
  await mount(list);
  await requestCount(2);
  expect(tab.nerve.calls[1]!.query).toMatchObject({ group_key: "A" });
  expect(tab.nerve.calls[1]!.query.cursor).toBeUndefined();
  await mount(list);
  await requestCount(2);
  await answer(1);
  expect(list.group("A")?.stale).toBe(false);
  expect(list.group("B")?.stale).toBe(true);
  await mount(list, ["A", "B"]);
  await requestCount(3);
  expect(tab.nerve.calls[2]!.query.group_key).toBe("B");
  await answer(2);
});
it("refusal stays stale without render loops and explicit retry starts one new read", async () => {
  const list = tab.app.issue.projectIssues.currentList!;
  list.markMembershipStale(undefined, listIssue("a", 10, { state_id: "A" }));
  await mount(list);
  await requestCount(2);
  await answer(1, 403);
  expect(list.group("A")?.stale).toBe(true);
  expect(binding.toast).toHaveBeenCalledTimes(1);
  await mount(list);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toHaveLength(2);
  await act(async () => container.querySelector("button")!.click());
  await requestCount(3);
  await answer(2);
  expect(list.group("A")?.stale).toBe(false);
});
it("switching binding or retiring its root suppresses old followers and new requests", async () => {
  const old = tab.app.issue.projectIssues.currentList!;
  old.markMembershipStale(undefined, listIssue("a", 10, { state_id: "A" }));
  await mount(old);
  await requestCount(2);
  const oldRetry = binding.retry!;
  await answered(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20, query: { group_by: "state_id" } }),
    ["GET", "/api/v0/projects/A/issues"],
    page,
    "other binding"
  );
  await mount(tab.app.issue.projectIssues.currentList!);
  await answer(1, 403);
  await oldRetry();
  expect(tab.nerve.calls).toHaveLength(3);
  expect(binding.toast).not.toHaveBeenCalled();
  const list = tab.app.issue.projectIssues.currentList!;
  await act(async () => list.markMembershipStale(undefined, listIssue("b", 10, { state_id: "B" })));
  await mount(list, ["B"]);
  await requestCount(4);
  tab.app.dispose();
  await answer(3);
  await act(async () => container.querySelector("button")!.click());
  expect(tab.nerve.calls).toHaveLength(4);
  expect(binding.toast).not.toHaveBeenCalled();
});

it.each(["first", "next"])(
  "%s read refusal has no unhandled rejection and retries the exact request only explicitly",
  async (kind) => {
    const request = await requestedIssue(
      tab.nerve,
      () =>
        (kind === "first"
          ? tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
              limit: 11,
              query: { group_by: "state_id" },
            })
          : tab.app.issue.projectIssues.fetchNextIssues("ws", "A", "A")
        ).catch(() => undefined),
      ["GET", "/api/v0/projects/A/issues"]
    );
    const list = tab.app.issue.projectIssues.currentList!;
    await mount(list);
    await answer(1, 403);
    await request.promise;
    expect(binding.toast).toHaveBeenCalledTimes(1);
    expect(kind === "first" ? list.reconciled.value : list.group("A")?.rows.map((row) => row.id)).toEqual(
      kind === "first" ? undefined : ["a"]
    );
    await mount(list);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100);
    });
    expect(tab.nerve.calls).toHaveLength(2);
    await act(async () => container.querySelector("button")!.click());
    await requestCount(3);
    expect(tab.nerve.calls[2]!.query).toEqual(request.call.query);
    await act(async () => {
      tab.nerve.calls[2]!.answer(
        json(
          200,
          kind === "first" ? page : { total: 2, data: [listIssue("tail", 20, { state_id: "A" })], next_cursor: null }
        )
      );
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(list.error).toBeUndefined();
  }
);
