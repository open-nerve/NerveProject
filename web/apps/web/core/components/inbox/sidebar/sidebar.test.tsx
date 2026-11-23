// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useLocation, useNavigationType, useSearchParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeIssuePage } from "@nerve/api-client";
import type { EmptyStateDetailed } from "@nerve/propel/empty-state";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { acme, intakeIssueOf, intakeTab, listPath, pageOf } from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { IntakePageStateProvider, useIntakePageState } from "../page-state";
import { useIntakeList } from "../use-intake-list";
import { IntakeSidebar } from "./root";

// The intake page's sidebar (P5a spec 2.5–2.7, R6, R31): its tabs, open and closed, tabs as a screen reader reads
// them, the address's chosen, the open tab saying its list's total once read, the closed none; choosing one goes to its
// address (a push), the selection left behind. Its list: the first page of the tab's filters, a wait until nerve
// answers, why nerve could not read it with a retry (never a wait that does not end), each tab's own empty state; the
// next page read as the list's end comes into view, by nerve's cursor, one at a time, none past the last; each row a
// link named by its identifier and name, to its address in the tab (a push). The list's read (useIntakeList), the
// stores, SWR and HTTP are real, with the test's cache; the end's coming into view is the test's (IntersectionObserver
// a stand-in). @nerve/i18n's text is its key, then the values it was given; an empty state says its title.

const held = vi.hoisted(() => ({
  where: undefined as { search: string; type: string } | undefined,
  opened: 0,
  setFilters: undefined as undefined | ((change: (filters: IntakeFilters) => IntakeFilters) => void),
  observers: [] as { callback: IntersectionObserverCallback; element?: Element }[],
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateDetailed: (props: ComponentProps<typeof EmptyStateDetailed>) => <p data-empty-state>{props.title}</p>,
}));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));

const Page = observer(function Page() {
  const { filters, setFilters } = useIntakePageState();
  const list = useIntakeList("p-web", filters);
  const [params] = useSearchParams();
  const location = useLocation();
  held.where = { search: location.search, type: useNavigationType() };
  held.setFilters = setFilters;
  return (
    <IntakeSidebar
      workspaceSlug="acme"
      projectId="p-web"
      list={list}
      selectedId={params.get("inboxIssueId") ?? undefined}
      onOpen={() => held.opened++}
    />
  );
});

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      private readonly seen: { callback: IntersectionObserverCallback; element?: Element };
      constructor(callback: IntersectionObserverCallback) {
        this.seen = { callback };
        held.observers.push(this.seen);
      }
      observe(element: Element) {
        this.seen.element = element;
      }
      unobserve() {
        held.observers.splice(held.observers.indexOf(this.seen), 1);
      }
      disconnect() {}
    }
  );
  toasts.length = 0;
  held.opened = 0;
  held.observers = [];
  tab = await intakeTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [
    projectOf("WEB", acme.id, { intake_view: true, member_role: 20 }),
  ]);
  tab.nerve.calls.length = 0;
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const render = (search = "?currentTab=open") =>
  act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <MemoryRouter initialEntries={[`/acme/projects/p-web/intake${search}`]}>
          <Routes>
            <Route
              path="/:workspaceSlug/projects/:projectId/intake"
              element={
                <IntakePageStateProvider>
                  <Page />
                </IntakePageStateProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
/** The read of p-web's list nerve receives after the `before` reads it received so far. */
async function listRead(before: number): Promise<Call> {
  await act(async () => until(() => tab.nerve.to(listPath("p-web")).length > before, `list read ${before + 1}`));
  return tab.nerve.to(listPath("p-web"))[before]!;
}
async function answer(call: Call, page: IntakeIssuePage) {
  call.answer(json(200, page));
  await settled();
}
const one = intakeIssueOf("One", { sequence_id: 1 });
const two = intakeIssueOf("Two", { sequence_id: 2 });
const three = intakeIssueOf("Three", { sequence_id: 3 });
const tabs = () => [...host.querySelectorAll<HTMLElement>('[role="tab"]')];
const tabNames = () => tabs().map((shown) => shown.getAttribute("aria-label") ?? shown.textContent);
const selectedTabs = () => tabs().map((shown) => shown.getAttribute("aria-selected"));
const rows = () => [...host.querySelectorAll<HTMLAnchorElement>("li a")];
const rowNames = () => rows().map((row) => row.getAttribute("aria-label"));
const placeholder = () => host.querySelector('[role="status"]');
const failure = () => host.querySelector('[role="alert"]');
const emptyState = () => host.querySelector("[data-empty-state]")?.textContent;
const button = (name: string) => [...host.querySelectorAll("button")].find((found) => found.textContent === name);
/** The list's end comes into view, as the browser tells its observer. */
const scrolledToEnd = () =>
  act(async () => {
    const seen = held.observers.at(-1)!;
    seen.callback([{ isIntersecting: true, target: seen.element! } as IntersectionObserverEntry], {} as never);
  });

describe("IntakeSidebar's tabs", () => {
  it("are tabs, the address's chosen, the open tab named with its list's total once read, the closed with none", async () => {
    await render();
    expect(host.querySelector('[role="tablist"]')).not.toBeNull();
    expect(tabNames()).toEqual(["inbox_issue.tabs.open", "inbox_issue.tabs.closed"]);
    expect(selectedTabs()).toEqual(["true", "false"]);
    await answer(await listRead(0), pageOf([one, two], "c-1", 51));
    expect(tabNames()).toEqual(["inbox_issue.a11y.tab_total inbox_issue.tabs.open 51", "inbox_issue.tabs.closed"]);

    await act(async () => tabs()[1]!.click());
    await answer(await listRead(1), pageOf([three], null, 1));
    expect(selectedTabs()).toEqual(["false", "true"]);
    // the closed tab says no total, chosen or not (R31)
    expect(tabNames()).toEqual(["inbox_issue.tabs.open", "inbox_issue.tabs.closed"]);
  });

  it("go to the chosen tab's address, a push, the selection left behind", async () => {
    await render("?currentTab=open&inboxIssueId=i-One");
    await act(async () => tabs()[1]!.click());
    expect(held.where).toEqual({ search: "?currentTab=closed", type: "PUSH" });
    expect(selectedTabs()).toEqual(["false", "true"]);
  });
});

describe("IntakeSidebar's list", () => {
  it("reads the tab's first page: the pending newest first, 50 a page; the closed tab's three final statuses", async () => {
    await render();
    const first = await listRead(0);
    expect(first.query).toEqual({ status: "pending", order_by: "-created_at", limit: "50" });
    await answer(first, pageOf([one]));
    await act(async () => tabs()[1]!.click());
    await settled();
    expect(tab.nerve.to(listPath("p-web")).map((call) => call.query)).toEqual([
      { status: "pending", order_by: "-created_at", limit: "50" },
      { status: "accepted,declined,duplicate", order_by: "-created_at", limit: "50" },
    ]);
  });

  it("waits for nerve's first page, saying it is loading, then shows a row for each work item", async () => {
    await render();
    const first = await listRead(0);
    expect(placeholder()?.textContent).toContain("common.loading");
    expect(failure()).toBeNull();
    await answer(first, pageOf([one, two]));
    expect(placeholder()).toBeNull();
    expect(rowNames()).toEqual(["WEB-1 One", "WEB-2 Two"]);
  });

  it("says why nerve could not read the list, waiting no more, and reads it again on retry", async () => {
    await render();
    (await listRead(0)).answer(problem(503, "server_busy"));
    await settled();
    expect(failure()?.textContent).toContain("errors.server_busy");
    expect(placeholder()).toBeNull();
    await act(async () => button("common.retry")!.click());
    await answer(await listRead(1), pageOf([one]));
    expect(failure()).toBeNull();
    expect(rowNames()).toEqual(["WEB-1 One"]);
  });

  it("says each tab's own empty state, and that no work item matches filters of the tab's own", async () => {
    await render();
    await answer(await listRead(0), pageOf([]));
    expect(emptyState()).toBe("project_empty_state.intake_sidebar.title");
    await act(async () => held.setFilters!((filters) => ({ ...filters, priorities: ["urgent"] })));
    await answer(await listRead(1), pageOf([]));
    expect(emptyState()).toBe("common_empty_state.search.title");
    await act(async () => tabs()[1]!.click());
    await answer(await listRead(2), pageOf([]));
    expect(emptyState()).toBe("project_empty_state.intake_closed.title");
    expect(rows()).toEqual([]);
  });

  it("reads the next page as the list's end comes into view, by nerve's cursor, one at a time, none past the last", async () => {
    await render();
    await answer(await listRead(0), pageOf([one, two], "c-1", 3));
    await scrolledToEnd();
    const next = await listRead(1);
    expect(next.query).toEqual({ status: "pending", order_by: "-created_at", limit: "50", cursor: "c-1" });
    // the end in view again while the next page is out reads nothing more
    await scrolledToEnd();
    await settled();
    expect(tab.nerve.to(listPath("p-web"))).toHaveLength(2);
    await answer(next, pageOf([three], null, 3));
    expect(rowNames()).toEqual(["WEB-1 One", "WEB-2 Two", "WEB-3 Three"]);
    await scrolledToEnd();
    await settled();
    expect(tab.nerve.to(listPath("p-web"))).toHaveLength(2);
  });

  it("says why nerve could not read the next page, keeping the rows read", async () => {
    await render();
    await answer(await listRead(0), pageOf([one], "c-1", 2));
    await scrolledToEnd();
    (await listRead(1)).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(rowNames()).toEqual(["WEB-1 One"]);
  });
});

describe("IntakeSidebar's rows", () => {
  it("each a link named by its identifier and name, to its address in the tab, the selected one current", async () => {
    await render("?currentTab=open&inboxIssueId=i-One");
    await answer(await listRead(0), pageOf([one, two]));
    expect(rows().map((row) => row.getAttribute("href"))).toEqual([
      "/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-One",
      "/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-Two",
    ]);
    expect(rows().map((row) => row.getAttribute("aria-current"))).toEqual(["true", null]);
  });

  it("selects its work item as the user presses it, a push; the selected one's press goes nowhere", async () => {
    await render("?currentTab=open&inboxIssueId=i-One");
    await answer(await listRead(0), pageOf([one, two]));
    await act(async () => rows()[1]!.click());
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Two", type: "PUSH" });
    expect(held.opened).toBe(1);
    await act(async () => rows()[1]!.click());
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Two", type: "PUSH" });
    expect(held.opened).toBe(2);
  });

  it("says the work item's status by its name, a snooze the days it has to go", async () => {
    await render();
    const later = new Date(Date.now() + 3 * 86_400_000).toISOString();
    const snoozed = intakeIssueOf("Four", {
      sequence_id: 4,
      intake: { status: "snoozed", snoozed_till: later, duplicate_to_id: null },
    });
    await answer(await listRead(0), pageOf([one, snoozed]));
    expect(rows()[0]!.textContent).toContain("inbox_issue.status.pending.title");
    expect(rows()[1]!.textContent).toContain("inbox_issue.status.snoozed.description");
  });
});
