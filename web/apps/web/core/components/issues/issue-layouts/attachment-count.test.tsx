// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode, createElement } from "react";
import { createRoot } from "react-dom/client";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueListOrder, IssueListPage, ProjectPreferences } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueStoreTab, listIssue } from "@/store/issue/fake-issues";
import { loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";
import { ProjectLayoutRoot } from "./roots/project-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
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
  useParams: () => ({ workspaceSlug: "ws", projectId: "A", userId: "actor", profileViewId: "assigned" }),
}));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => 15,
    getWorkspaceRoleByWorkspaceSlug: () => 15,
    allowPermissions: () => true,
  }),
  useUser: () => ({ data: { id: "actor" } }),
  useUserProfile: () => ({ data: { start_of_the_week: 0 } }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as = "div" }: { children: ReactNode; as?: string }) => createElement(as, null, children),
}));
// Continuation sentinels never fire on their own here: a later page stays unloaded until a test asks for it.
class QuietObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
type Layout = "list" | "kanban" | "spreadsheet";
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
const row = (id: string, attachment_count: number, changes: Partial<Issue> = {}) =>
  listIssue(id, 10, { attachment_count, priority: "high", ...changes });
const many = row("many", 51);
const single = row("single", 1);
const zero = row("zero", 0);
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const reads = () => tab.nerve.calls.map((call) => `${call.method} ${call.path}`);
const chips = () =>
  [...host.querySelectorAll(".sr-only")]
    .filter((node) => node.textContent === "content_files.title")
    .map((node) => node.parentElement!.textContent);
const cells = () =>
  [...host.querySelectorAll("tbody > tr")].map((tr) => tr.querySelector("td:nth-child(2)")?.textContent);
const names = () =>
  [...host.querySelectorAll("tbody > tr > td:first-child .line-clamp-1 .truncate")].map((node) => node.textContent);
function pageOf(layout: Layout, data: Issue[], next: string | null = null, total = data.length): IssueListPage {
  return layout === "kanban"
    ? { total, groups: [{ key: "high", total, data, next_cursor: next }] }
    : { total, data, next_cursor: next };
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("IntersectionObserver", QuietObserver);
  vi.stubGlobal("ResizeObserver", QuietObserver);
  toasts.length = 0;
  tab = await issueStoreTab();
  bound.app = tab.app;
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", userId: "actor" }));
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function render(children: ReactNode) {
  await act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter>{children}</MemoryRouter>
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
}
async function mount(
  layout: Layout,
  page: IssueListPage,
  { order = "-attachment_count", shown = true }: { order?: IssueListOrder; shown?: boolean } = {}
) {
  const prefs = preferencesOf();
  prefs.display_filters.layout = layout;
  prefs.display_filters.group_by = layout === "kanban" ? "priority" : null;
  prefs.display_filters.order_by = order;
  for (const key of Object.keys(prefs.display_properties) as (keyof ProjectPreferences["display_properties"])[])
    prefs.display_properties[key] = key === "key" || (key === "attachment_count" && shown);
  await render(<ProjectLayoutRoot />);
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "first page");
  await act(async () => lists()[0]!.answer(json(200, page)));
  return prefs;
}
/**
 * The detail read that opens a work item, or with afterWrite the read that follows a confirmed upload or deletion.
 * A post-write read settles only once the lists it made stale have read again, so it is returned unsettled.
 */
async function readIssue(issue: Issue, answer: number | Response, afterWrite = true) {
  const path = `/api/v0/issues/${issue.id}`;
  const before = tab.nerve.to(path).length;
  let done: Promise<unknown> = Promise.resolve();
  await act(async () => {
    done = tab.app.issue.issues.refreshAuthoritative(issue.id, { afterWrite }).then(
      () => undefined,
      (error: unknown) => error
    );
  });
  await until(() => tab.nerve.to(path).length === before + 1, "authoritative read");
  await act(async () => tab.nerve.to(path)[before]!.answer(countAnswer(issue, answer)));
  return { done };
}
function countAnswer(issue: Issue, answer: number | Response) {
  return typeof answer === "number"
    ? json(200, { ...issueDetailOf(issue.id), ...issue, attachment_count: answer })
    : answer;
}

it.each(["list", "kanban", "spreadsheet"] as const)(
  "%s shows the real attachment count of every row of its page and reads no row",
  async (layout) => {
    await mount(layout, pageOf(layout, [many, single, zero]));
    if (layout === "spreadsheet") {
      expect(cells()).toEqual([
        "51 content_files.count_unit",
        "1 content_files.count_unit",
        "0 content_files.count_unit",
      ]);
      expect(host.querySelector('th[aria-sort="descending"]')?.textContent).toContain("content_files.title");
    } else expect(chips()).toEqual(["content_files.title51", "content_files.title1"]);
    expect(lists()[0]!.query.order_by).toBe("-attachment_count");
    expect(reads()).toEqual(["GET /api/v0/me/projects/A/preferences", "GET /api/v0/projects/A/issues"]);
  }
);

it.each(["list", "kanban", "spreadsheet"] as const)(
  "%s shows no count or column while the saved preference hides attachments",
  async (layout) => {
    await mount(layout, pageOf(layout, [many, single, zero]), { shown: false });
    expect(chips()).toEqual([]);
    expect(host.querySelector("thead")?.textContent ?? "").not.toContain("content_files.title");
    expect(host.textContent).not.toContain("content_files.count_unit");
  }
);

it("the attachments header sorts both ways on the server and keeps its directional ID ties", async () => {
  const tieA = row("tie-a", 1);
  const tieB = row("tie-b", 1);
  const prefs = await mount("spreadsheet", pageOf("spreadsheet", [zero, tieA, tieB, many]), { order: "-created_at" });
  const header = () =>
    [...host.querySelectorAll<HTMLTableCellElement>("th[aria-sort]")].find((th) =>
      th.textContent?.includes("content_files.title")
    )!;
  const writes = () => tab.nerve.to("/api/v0/me/projects/A/preferences").filter((call) => call.method === "PATCH");
  async function choose(index: number, order: IssueListOrder, page: Issue[]) {
    const write = writes().length;
    const trigger = header().querySelector("button")!;
    vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 180, 40));
    await act(async () => trigger.click());
    await act(async () => document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')[index]!.click());
    await until(() => writes().length === write + 1, "sort write");
    expect(writes()[write]!.body).toEqual({ display_filters: { ...prefs.display_filters, order_by: order } });
    prefs.display_filters.order_by = order;
    const read = lists().length;
    await act(async () => writes()[write]!.answer(json(200, prefs)));
    await until(() => lists().length === read + 1, "confirmed query");
    expect(lists()[read]!.query.order_by).toBe(order);
    await act(async () => lists()[read]!.answer(json(200, pageOf("spreadsheet", page))));
  }
  expect(header().getAttribute("aria-sort")).toBe("none");
  await choose(0, "-attachment_count", [many, tieB, tieA, zero]);
  expect(header().getAttribute("aria-sort")).toBe("descending");
  expect(names()).toEqual(["many", "tie-b", "tie-a", "zero"]);
  await choose(1, "attachment_count", [zero, tieA, tieB, many]);
  expect(header().getAttribute("aria-sort")).toBe("ascending");
  expect(names()).toEqual(["zero", "tie-a", "tie-b", "many"]);
  expect(cells()).toEqual([
    "0 content_files.count_unit",
    "1 content_files.count_unit",
    "1 content_files.count_unit",
    "51 content_files.count_unit",
  ]);
  expect(tab.nerve.calls.filter((call) => call.path.startsWith("/api/v0/issues/"))).toHaveLength(0);
});

it("a confirmed count change re-reads the list, so rows move to and from pages it has not loaded", async () => {
  await mount("list", pageOf("list", [many, single], "page:2", 3));
  expect(chips()).toEqual(["content_files.title51", "content_files.title1"]);
  // Files are added from the work item's detail, which reads the item first; it lives on the unloaded page.
  const opened = await readIssue(zero, 0, false);
  await act(async () => opened.done);
  expect(lists()).toHaveLength(1);
  const upload = await readIssue(zero, 70);
  await until(() => lists().length === 2, "re-read after upload");
  expect(lists()[1]!.query).toEqual(lists()[0]!.query);
  const lifted = { ...zero, attachment_count: 70 };
  await act(async () => lists()[1]!.answer(json(200, pageOf("list", [lifted, many], "page:2", 3))));
  await act(async () => upload.done);
  expect(chips()).toEqual(["content_files.title70", "content_files.title51"]);
  // Deleting every attachment of a loaded row sends it to the unloaded page.
  const removal = await readIssue(many, 0);
  await until(() => lists().length === 3, "re-read after delete");
  expect(lists()[2]!.query).toEqual(lists()[0]!.query);
  await act(async () => lists()[2]!.answer(json(200, pageOf("list", [lifted, single], "page:2", 3))));
  await act(async () => removal.done);
  expect(chips()).toEqual(["content_files.title70", "content_files.title1"]);
  expect(host.textContent).not.toContain("many");
  expect(reads().filter((call) => call.startsWith("GET /api/v0/issues/"))).toEqual([
    "GET /api/v0/issues/zero",
    "GET /api/v0/issues/zero",
    "GET /api/v0/issues/many",
  ]);
  expect(reads().filter((call) => call.includes("/attachments"))).toEqual([]);
});

it("another list's page with a changed count re-reads a list sorted by it instead of keeping its place", async () => {
  await mount("list", pageOf("list", [many, single], "page:2", 3));
  await tab.app.issue.profileIssuesFilter.fetchFilters("ws", "actor");
  const member = () => tab.nerve.to("/api/v0/workspaces/ws/members/actor/issues");
  let other: Promise<unknown> = Promise.resolve();
  await act(async () => {
    other = tab.app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 20 }, "assigned");
  });
  await until(() => member().length === 1, "member page");
  await act(async () => member()[0]!.answer(json(200, pageOf("list", [{ ...single, attachment_count: 60 }]))));
  await act(async () => other);
  await until(() => lists().length === 2, "re-read for the changed order");
  expect(lists()[1]!.query).toEqual(lists()[0]!.query);
  await act(async () =>
    lists()[1]!.answer(json(200, pageOf("list", [{ ...single, attachment_count: 60 }, many], "page:2", 3)))
  );
  expect(chips()).toEqual(["content_files.title60", "content_files.title51"]);
  expect(reads().filter((call) => call.startsWith("GET /api/v0/issues/"))).toEqual([]);
});

it.each([403, 503])(
  "a %i read after an attachment change shows no old count, and only a confirmed retry shows the new one",
  async (status) => {
    await mount("list", pageOf("list", [many, single], "page:2", 3));
    const change = await readIssue(many, problem(status, status === 403 ? "forbidden" : "server_busy"));
    expect(await change.done).toMatchObject({ status });
    expect(chips()).toEqual(["content_files.title1"]);
    expect(toasts).toHaveLength(1);
    const retry = [...host.querySelectorAll("button")].find((node) => node.textContent === "common.retry")!;
    await act(async () => retry.click());
    const path = "/api/v0/issues/many";
    await until(() => tab.nerve.to(path).length === 2, "explicit retry");
    await act(async () => tab.nerve.to(path)[1]!.answer(countAnswer(many, 52)));
    await until(() => lists().length === 2, "re-read for the changed count");
    await act(async () =>
      lists()[1]!.answer(json(200, pageOf("list", [{ ...many, attachment_count: 52 }, single], "page:2", 3)))
    );
    expect(chips()).toEqual(["content_files.title52", "content_files.title1"]);
  }
);

it("a work item missing after an attachment change leaves the list through its re-read", async () => {
  await mount("list", pageOf("list", [many, single], "page:2", 3));
  const change = await readIssue(many, problem(404, "not_found"));
  expect(chips()).not.toContain("content_files.title51");
  await until(() => lists().length === 2, "re-read for the missing row");
  await act(async () => lists()[1]!.answer(json(200, pageOf("list", [single], "page:2", 2))));
  await act(async () => change.done);
  expect(chips()).toEqual(["content_files.title1"]);
  expect(host.textContent).not.toContain("many");
});

it("the member view shows counts of rows from several projects and reads no row", async () => {
  const projectB = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, projectB]);
  tab.nerve.calls.length = 0;
  await render(<ProfileIssuesPage type="assigned" />);
  const member = () => tab.nerve.to("/api/v0/workspaces/ws/members/actor/issues");
  await until(() => member().length === 1, "member list");
  const other = { ...many, id: "other", name: "other", project_id: "B" };
  await act(async () => member()[0]!.answer(json(200, pageOf("list", [zero, single, other]))));
  expect(chips()).toEqual(["content_files.title1", "content_files.title51"]);
  expect(reads()).toEqual(["GET /api/v0/workspaces/ws/members/actor/issues"]);
});
