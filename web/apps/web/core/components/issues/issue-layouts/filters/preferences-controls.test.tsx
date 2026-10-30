// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ISSUE_DISPLAY_PROPERTIES } from "@nerve/constants";
import { StoreContext } from "@/lib/store-context";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { preferencesOf } from "@/store/project/fake-projects";
import { IssueListRetry, useIssueListRefresh } from "@/hooks/use-issue-list-refresh";
import { HeaderFilters } from "../../filters";

const fixture = vi.hoisted(() => ({ app: undefined as unknown, toast: vi.fn() }));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return {
    StoreContext: createContext(undefined),
    get rootStore() {
      return fixture.app;
    },
  };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "preferences" } }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => ({ setToast: fixture.toast, TOAST_TYPE: { ERROR: "error" } }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const initial = preferencesOf({
  display_filters: { ...preferencesOf().display_filters, calendar: { layout: "week", show_weekends: false } },
});
const View = observer(function View() {
  const retry = useIssueListRefresh(tab.app.issue.projectIssues.currentList, []);
  return (
    <>
      <HeaderFilters workspaceSlug="ws" projectId="A" />
      <IssueListRetry {...retry} />
    </>
  );
});
const button = (name: string) =>
  [...host.querySelectorAll<HTMLButtonElement>("button")].find(
    (item) => item.getAttribute("aria-label") === name || item.textContent === name
  )!;
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const click = (name: string) => act(async () => button(name).click());
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.toast.mockClear();
  const pref = await requestedIssue(tab.nerve, () => tab.app.issue.projectIssuesFilter.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  pref.call.answer(json(200, initial));
  await pref.promise;
  const list = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  list.call.answer(json(200, issuePageOf([issueOf("saved")])));
  await list.promise;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <StoreContext.Provider value={tab.app}>
        <View />
      </StoreContext.Provider>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

it("waits for confirmed property values and serializes two changes without losing the first answer", async () => {
  await click("common.display");
  expect(ISSUE_DISPLAY_PROPERTIES.every((property) => !!button(property.titleTranslationKey))).toBe(true);
  await click("common.priority");
  await act(async () => until(() => patches().length === 1, "priority patch"));
  expect(button("common.priority").getAttribute("aria-pressed")).toBe("true");
  await click("common.state");
  expect(patches()).toHaveLength(1);
  const confirmed = { ...initial.display_properties, priority: false, labels: false };
  await act(async () => {
    patches()[0].answer(json(200, preferencesOf({ ...initial, display_properties: confirmed })));
    await until(() => patches().length === 2, "queued state patch");
  });
  expect(patches()[1].body).toEqual({ display_properties: { ...confirmed, state: false } });
  expect(button("common.priority").getAttribute("aria-pressed")).toBe("false");
  expect(button("common.state").getAttribute("aria-pressed")).toBe("true");
  await act(async () =>
    patches()[1].answer(json(200, preferencesOf({ ...initial, display_properties: { ...confirmed, state: false } })))
  );
  expect(button("common.state").getAttribute("aria-pressed")).toBe("false");
  expect(fixture.toast).not.toHaveBeenCalled();
});

it("keeps other fields, null grouping, false and nested calendar settings when independent controls save", async () => {
  await click("issue.layouts.title.spreadsheet");
  await act(async () => until(() => patches().length === 1, "layout patch"));
  const display = { ...initial.display_filters, layout: "spreadsheet" };
  expect(patches()[0].body).toEqual({ display_filters: display });
  expect(button("issue.layouts.title.list").getAttribute("aria-pressed")).toBe("true");
  expect(button("issue.layouts.title.spreadsheet").getAttribute("aria-pressed")).toBe("false");
  await click("common.display");
  await click("common.priority");
  await act(async () => until(() => patches().length === 2, "independent property patch"));
  await act(async () =>
    patches()[1].answer(
      json(200, preferencesOf({ ...initial, display_properties: { ...initial.display_properties, priority: false } }))
    )
  );
  await act(async () => patches()[0].answer(json(200, preferencesOf({ ...initial, display_filters: display }))));
  expect(tab.app.issue.projectIssuesFilter.getIssueFilters("A")).toMatchObject({
    displayFilters: display,
    displayProperties: { priority: false },
  });
  expect(button("issue.layouts.title.spreadsheet").getAttribute("aria-pressed")).toBe("true");
  expect(tab.nerve.calls.filter((call) => call.method === "GET" && call.path.endsWith("/issues"))).toHaveLength(1);
});

it("keeps the confirmed controls and shows one refusal when the save is rejected", async () => {
  await click("common.display");
  await click("common.priority");
  await act(async () => until(() => patches().length === 1, "rejected property patch"));
  await act(async () => patches()[0].answer(problem(403, "forbidden")));
  expect(button("common.priority").getAttribute("aria-pressed")).toBe("true");
  expect(fixture.toast).toHaveBeenCalledTimes(1);
  expect(tab.nerve.calls.filter((call) => call.method === "GET" && call.path.endsWith("/issues"))).toHaveLength(1);
});

it("keeps a saved display setting when its new list read fails and retries only that read", async () => {
  await click("common.display");
  await click("issue.display.extra.show_sub_issues");
  await act(async () => until(() => patches().length === 1, "sub-issue display patch"));
  const display = { ...initial.display_filters, sub_issue: false };
  await act(async () => {
    patches()[0].answer(json(200, preferencesOf({ ...initial, display_filters: display })));
    await until(() => tab.nerve.calls.length === 4, "new list query");
  });
  const read = tab.nerve.calls[3];
  expect(read.method).toBe("GET");
  await act(async () => read.answer(problem(503, "server_busy")));
  expect(button("issue.display.extra.show_sub_issues").getAttribute("aria-pressed")).toBe("false");
  expect(fixture.toast).toHaveBeenCalledTimes(1);
  await click("common.retry");
  await act(async () => until(() => tab.nerve.calls.length === 5, "read retry"));
  expect(tab.nerve.calls[4]).toMatchObject({ method: "GET", path: read.path, query: read.query });
  expect(patches()).toHaveLength(1);
  await act(async () => tab.nerve.calls[4].answer(json(200, issuePageOf([issueOf("fresh")]))));
  expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["fresh"]);
});
