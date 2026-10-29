// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { StoreContext } from "@/lib/store-context";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import { useProjectPreferencesFollow } from "./use-project-preferences-follow";
import { IssueListRetry, useIssueListRefresh } from "./use-issue-list-refresh";

const fixture = vi.hoisted(() => ({ app: undefined as unknown, session: { loginId: "A" }, toast: vi.fn() }));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return {
    StoreContext: createContext(undefined),
    get rootStore() {
      return fixture.app;
    },
  };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => ({ setToast: fixture.toast, TOAST_TYPE: { ERROR: "error" } }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
const Mounted = observer(function Mounted() {
  const follow = useProjectPreferencesFollow();
  const list = tab.app.issue.projectIssues.currentList;
  const retry = useIssueListRefresh(list, []);
  return (
    <>
      <button
        onClick={() =>
          void follow("ws", "A", () =>
            tab.app.issue.projectIssuesFilter.updateFilterExpression("ws", "A", { and: [{ priority__in: "low" }] })
          )
        }
      >
        save
      </button>
      <button
        onClick={() =>
          void follow("ws", "A", () =>
            tab.app.issue.projectIssuesFilter.updateFilterExpression("ws", "A", { and: [{ priority__in: "high" }] })
          )
        }
      >
        restore
      </button>
      <IssueListRetry {...retry} />
    </>
  );
});
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.session.loginId = "A";
  fixture.toast.mockClear();
  const pref = await requestedIssue(tab.nerve, () => tab.app.issue.projectIssuesFilter.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  pref.call.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  await pref.promise;
  const old = await requestedIssue(
    tab.nerve,
    () =>
      tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { start_date: ["2026-01-01..2026-02-01"] },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  old.call.answer(json(200, issuePageOf([issueOf("old")])));
  await old.promise;
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <Mounted />
      </StoreContext.Provider>
    )
  );
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.useRealTimers();
});
it("retains a confirmed preference when its new list read fails and retries only that read", async () => {
  const store = tab.app.issue.projectIssuesFilter;
  const issues = tab.app.issue.projectIssues;
  const previous = issues.currentList;
  await act(async () => {
    container.querySelector("button")!.click();
    await until(() => tab.nerve.calls.length === 3, "preference patch");
  });
  expect(tab.nerve.calls[2]).toMatchObject({
    method: "PATCH",
    path: "/api/v0/me/projects/A/preferences",
    body: { filters: { priority: ["low"] } },
  });
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  expect(issues.currentList).toBe(previous);
  expect(issues.getIssueIds()).toEqual(["old"]);
  await act(async () => {
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
    await until(() => tab.nerve.calls.length === 4, "new list");
  });
  const newRead = tab.nerve.calls[3]!;
  expect(newRead).toMatchObject({ method: "GET", path: "/api/v0/projects/A/issues", query: { priority: "low" } });
  expect(issues.currentList).not.toBe(previous);
  await act(async () => {
    newRead.answer(problem(503, "server_busy"));
    await until(() => !!issues.currentList?.error, "list refusal");
  });
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "low" }] });
  expect(issues.currentList?.error).toBeDefined();
  expect(container.querySelectorAll("button")).toHaveLength(3);
  await act(async () => {
    container.querySelectorAll("button")[2]!.click();
    await until(() => tab.nerve.calls.length === 5, "retry");
  });
  expect(tab.nerve.calls[4]).toMatchObject({ method: "GET", path: newRead.path, query: newRead.query });
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  await act(async () => {
    tab.nerve.calls[4]!.answer(json(200, issuePageOf([issueOf("fresh")])));
    await until(() => !issues.currentList?.error, "retry answer");
  });
  expect(issues.getIssueIds()).toEqual(["fresh"]);
  expect(container.querySelectorAll("button")).toHaveLength(2);
});

it("refreshes the final confirmed query when queued changes restore the original value", async () => {
  const issues = tab.app.issue.projectIssues;
  await act(async () => {
    container.querySelectorAll("button")[0]!.click();
    container.querySelectorAll("button")[1]!.click();
    await until(() => tab.nerve.calls.length === 3, "first patch");
  });
  expect(tab.nerve.calls[2]?.body).toEqual({ filters: { priority: ["low"] } });
  expect(issues.getIssueIds()).toEqual(["old"]);
  await act(async () => {
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
    await until(() => tab.nerve.calls.length === 5, "second patch and low list");
  });
  const second = tab.nerve.calls.find((c, i) => i >= 3 && c.method === "PATCH")!;
  const low = tab.nerve.calls.find((c, i) => i >= 3 && c.method === "GET")!;
  expect(second.body).toEqual({ filters: { priority: ["high"] } });
  expect(low.query).toMatchObject({ priority: "low" });
  await act(async () => {
    low.answer(json(200, issuePageOf([issueOf("low-row")])));
    await until(() => issues.getIssueIds()?.includes("low-row") === true, "low rows");
  });
  await act(async () => {
    second.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
    await until(() => tab.nerve.calls.length === 6, "high list");
  });
  const high = tab.nerve.calls[5]!;
  expect(high).toMatchObject({ method: "GET", path: "/api/v0/projects/A/issues", query: { priority: "high" } });
  expect(issues.currentList?.scope.query.priority).toEqual(["high"]);
  await act(async () => {
    high.answer(json(200, issuePageOf([issueOf("high-row")])));
    await until(() => issues.getIssueIds()?.includes("high-row") === true, "high rows");
  });
  expect(issues.getIssueIds()).toEqual(["high-row"]);
  expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(2);
});

it("does not refetch a genuinely unchanged preference with extra list parameters", async () => {
  await act(async () => {
    container.querySelectorAll("button")[1]!.click();
    await until(() => tab.nerve.calls.length === 3, "no-op patch");
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toHaveLength(3);
  expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["old"]);
});

it.each(["session", "project"])("does not refresh the old list when %s changes while saving", async (change) => {
  await act(async () => {
    container.querySelectorAll("button")[0]!.click();
    await until(() => tab.nerve.calls.length === 3, "held patch");
    if (change === "session") fixture.session.loginId = "B";
    else runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "B" }));
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
    await until(
      () =>
        JSON.stringify(tab.app.issue.projectIssuesFilter.getIssueFilters("A")?.richFilters) ===
        JSON.stringify({ and: [{ priority__in: "low" }] }),
      "confirmation"
    );
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toHaveLength(3);
  expect(fixture.toast).not.toHaveBeenCalled();
});
