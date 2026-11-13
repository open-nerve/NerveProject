// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The page's own preferences read again on the three pages of a list of work items, a cycle's, a module's and the
// project's (P6b spec 2.8, R4; handoff item 9): once nerve has given them, a read of them again that fails (as SWR
// makes on reconnect) leaves the list as it shows under the preferences read, with one line above it that says the read
// failed, why, and a retry that reads them again; the header's layouts, filters and display stay. Its retry read,
// the line is gone. A first read that fails is as before: the cycle's and the module's page say why with a retry
// (PlanningReadFailure), the project's page says M4's words alone (spec 5, M8). The pages' headers and work items, the
// stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { CycleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header";
import { ModuleIssuesHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { IssuesHeader } from "../../header";
import {
  buttonOf,
  C1,
  cyclePreferencesOf,
  layouts,
  M1,
  modulePreferencesOf,
  planningPageTab,
} from "./fake-planning-page";
import { ProjectLayoutRoot } from "./project-layout-root";

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
// jsdom has no viewport geometry; the rows and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// The crumbs and the cycle's and the module's quick actions have their own tests.
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/cycles/quick-actions", () => ({ CycleQuickActions: () => null }));
vi.mock("@/components/modules/quick-actions", () => ({ ModuleQuickActions: () => null }));

/** A page open in a tab: what nerve was asked for it, and what nerve gives it. */
type Opened = {
  close: () => Promise<void>;
  preferences: () => Call[];
  lists: () => Call[];
  /** Renders the page: its header, then its work items. */
  mount: () => Promise<void>;
  /** Sets how nerve answers each request the test does not answer itself. */
  respond: (answer: (call: Call) => boolean) => void;
  /** nerve's preferences of the page, in the list layout. */
  read: unknown;
  /** nerve's list of the page, of its one work item "first". */
  list: unknown;
};

const header = (node: ReactNode) => <div data-header>{node}</div>;

/** A planning page of project A for a member, as planningPageTab: the cycle's C1 or the module's M1. */
async function planningPage(type: EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE): Promise<Opened> {
  const page = await planningPageTab(type, { role: 15 });
  bound.app = page.app;
  const cycle = type === EIssuesStoreType.CYCLE;
  bound.params = { workspaceSlug: "ws", projectId: "A", ...(cycle ? { cycleId: C1 } : { moduleId: M1 }) };
  const before = page.nerve.respond;
  return {
    close: page.close,
    preferences: page.preferences,
    lists: page.lists,
    mount: () =>
      page.mount(
        header(
          cycle ? (
            <CycleIssuesHeader workspaceSlug="ws" projectId="A" cycleId={C1} />
          ) : (
            <ModuleIssuesHeader workspaceSlug="ws" projectId="A" moduleId={M1} />
          )
        )
      ),
    respond: (answer) => {
      page.nerve.respond = (call) => {
        if (!answer(call)) before?.(call);
      };
    },
    read: (cycle ? cyclePreferencesOf : modulePreferencesOf)(layouts.list.display),
    list: page.listOf("list"),
  };
}

/** The project A's work items page for one of its members, as the page puts its header and its work items. */
async function projectPage(): Promise<Opened> {
  const tab = await issueSelectionTab();
  bound.app = tab.app;
  bound.params = { workspaceSlug: "ws", projectId: "A" };
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  const before = tab.nerve.respond;
  const read = preferencesOf();
  read.display_filters = { ...read.display_filters, ...layouts.list.display };
  return {
    close: tab.close,
    preferences: () => tab.nerve.to("/api/v0/me/projects/A/preferences"),
    lists: () => tab.nerve.to("/api/v0/projects/A/issues"),
    mount: () =>
      tab.render(
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter>
            {header(<IssuesHeader />)}
            <ProjectLayoutRoot />
          </MemoryRouter>
        </SWRConfig>
      ),
    respond: (answer) => {
      tab.nerve.respond = (call) => {
        if (!answer(call)) before?.(call);
      };
    },
    read,
    list: layouts.list.page(),
  };
}

const pages = [
  { page: "a cycle's page", make: () => planningPage(EIssuesStoreType.CYCLE) },
  { page: "a module's page", make: () => planningPage(EIssuesStoreType.MODULE) },
  { page: "the project's page", make: projectPage },
];
const planningPages = pages.slice(0, 2);

let current: Opened | undefined;
afterEach(async () => {
  await current?.close();
  current = undefined;
});
/** Opens the page make makes, closed after the test. */
async function open(make: () => Promise<Opened>) {
  current = await make();
  return current;
}

/** The page's one work item, as its list shows it. */
const row = () => document.querySelector('a[id="issue-first"]');
const all = ["list", "kanban", "calendar", "spreadsheet"];
const inHeader = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("[data-header] button")].find(
    (node) => (node.getAttribute("aria-label") ?? node.textContent?.trim()) === name
  );
/** What of the page's work items its header offers. */
const offered = () => ({
  layouts: all.filter((layout) => inHeader(`issue.layouts.title.${layout}`)),
  filters: !!inHeader("common.filter_options.add"),
  display: !!inHeader("common.display"),
});
const alerts = () => [...document.querySelectorAll('[role="alert"]')];
/** The line that says the page's preferences could not be read again. */
const notice = () => alerts().find((node) => node.textContent?.includes("issue.layouts.preferences_reread_failed"));

/** Renders the page; nerve gives its preferences, then its list. */
async function shown(page: Opened) {
  await page.mount();
  await act(async () => until(() => page.preferences().length === 1, "the page's preferences"));
  await act(async () => page.preferences()[0]!.answer(json(200, page.read)));
  await act(async () => until(() => page.lists().length === 1, "the page's list"));
  await act(async () => page.lists()[0]!.answer(json(200, page.list)));
  expect(row(), "the page's work item").not.toBeNull();
}

/** The tab is back online: SWR reads again what the page shows, its preferences among them. */
async function reconnected(page: Opened) {
  const sent = page.preferences().length;
  // past SWR's deduplication of the read before
  await act(async () => vi.advanceTimersByTimeAsync(5000));
  await act(async () => {
    window.dispatchEvent(new Event("online"));
  });
  await act(async () => until(() => page.preferences().length === sent + 1, "the page's preferences read again"));
}

it.each(pages)(
  "$page keeps its list under the preferences read when nerve fails to read them again, and says so above it with a retry",
  async ({ make }) => {
    const opened = await open(make);
    await shown(opened);
    expect(offered()).toEqual({ layouts: all, filters: true, display: true });
    // nerve lists the page as before each time it is read again
    opened.respond((call) => {
      if (call.path !== opened.lists()[0]!.path) return false;
      call.answer(json(200, opened.list));
      return true;
    });
    await reconnected(opened);
    expect(opened.preferences()[1]!.method).toBe("GET");
    await act(async () => opened.preferences()[1]!.answer(problem(503, "server_busy")));

    const line = notice();
    expect(line, "the line that says the read again failed").toBeDefined();
    expect(line!.textContent).toContain("errors.server_busy");
    expect(row(), "the list under the preferences read").not.toBeNull();
    expect(line!.compareDocumentPosition(row()!) & Node.DOCUMENT_POSITION_FOLLOWING, "above the list").toBeTruthy();
    expect(offered(), "the header's controls").toEqual({ layouts: all, filters: true, display: true });

    const retry = [...line!.querySelectorAll("button")].find((node) => node.textContent?.trim() === "common.retry");
    expect(retry, "the line's retry").toBeDefined();
    await act(async () => retry!.click());
    await act(async () => until(() => opened.preferences().length === 3, "the page's preferences on retry"));
    expect(opened.preferences()[2]!.method).toBe("GET");
    await act(async () => opened.preferences()[2]!.answer(json(200, opened.read)));
    expect(notice(), "the line, once its retry read them").toBeUndefined();
    expect(alerts()).toEqual([]);
    expect(row()).not.toBeNull();
    expect(offered()).toEqual({ layouts: all, filters: true, display: true });
  }
);

it.each(pages)(
  "$page keeps saying the read again failed while its retry fails as well, its list still shown",
  async ({ make }) => {
    const opened = await open(make);
    await shown(opened);
    opened.respond((call) => {
      if (call.path !== opened.lists()[0]!.path) return false;
      call.answer(json(200, opened.list));
      return true;
    });
    await reconnected(opened);
    await act(async () => opened.preferences()[1]!.answer(problem(503, "server_busy")));
    expect(notice(), "the line that says the read again failed").toBeDefined();
    await act(async () => buttonOf("common.retry")?.click());
    await act(async () => until(() => opened.preferences().length === 3, "the page's preferences on retry"));
    await act(async () => opened.preferences()[2]!.answer(problem(403, "forbidden")));
    expect(notice(), "the line, its retry failed").toBeDefined();
    expect(notice()!.textContent).toContain("errors.forbidden");
    expect(row()).not.toBeNull();
    expect(offered()).toEqual({ layouts: all, filters: true, display: true });
  }
);

it.each(planningPages)(
  "$page says why nerve could not give its first preferences, with a retry, as before, never the line of a read again",
  async ({ make }) => {
    const opened = await open(make);
    await opened.mount();
    await act(async () => until(() => opened.preferences().length === 1, "the page's preferences"));
    await act(async () => opened.preferences()[0]!.answer(problem(503, "server_busy")));
    expect(alerts()).toHaveLength(1);
    expect(alerts()[0]!.textContent).toContain("errors.server_busy");
    expect(notice(), "no line of a read again").toBeUndefined();
    expect(opened.lists()).toEqual([]);
    expect(offered()).toEqual({ layouts: [], filters: false, display: false });
    await act(async () => buttonOf("common.retry")?.click());
    await act(async () => until(() => opened.preferences().length === 2, "the page's preferences on retry"));
    await act(async () => opened.preferences()[1]!.answer(json(200, opened.read)));
    await act(async () => until(() => opened.lists().length === 1, "the page's list"));
    expect(alerts()).toEqual([]);
  }
);

it("the project's page says M4's words alone when nerve could not give its first preferences, as before", async () => {
  const opened = await open(projectPage);
  await opened.mount();
  await act(async () => until(() => opened.preferences().length === 1, "the page's preferences"));
  await act(async () => opened.preferences()[0]!.answer(problem(503, "server_busy")));
  expect(alerts().map((node) => node.textContent)).toEqual(["errors.server_busy"]);
  expect(buttonOf("common.retry"), "no retry (M8)").toBeUndefined();
  expect(opened.lists()).toEqual([]);
  expect(offered()).toEqual({ layouts: [], filters: false, display: false });
});
