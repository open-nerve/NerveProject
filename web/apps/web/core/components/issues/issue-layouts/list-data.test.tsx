// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { EIssuesStoreType } from "@nerve/types";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab, issuePageOf, issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects, preferencesOf, projectOf } from "@/store/project/fake-projects";
import { getSourceFromDropPayload, getDestinationFromDropPayload } from "./utils";
import { ProjectLayoutRoot } from "./roots/project-layout-root";
import { ArchivedIssueLayoutRoot } from "./roots/archived-issue-layout-root";
import { ProfileIssuesPage } from "@/components/profile/profile-issues";

type DropRegistration = Parameters<
  typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements
>[0];
type DragRegistration = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const dnd = vi.hoisted(() => ({
  drops: new Map<Element, DropRegistration>(),
  drags: new Map<Element, DragRegistration>(),
}));
// jsdom has no scroll geometry; this suite exercises producer payloads, not auto scrolling.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
class VisibleObserver {
  constructor(private readonly callback: IntersectionObserverCallback) {}
  observe(target: Element) {
    this.callback(
      [{ target, isIntersecting: true } as IntersectionObserverEntry],
      this as unknown as IntersectionObserver
    );
  }
  unobserve() {}
  disconnect() {}
}
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  dropTargetForElements: (options: DropRegistration) => {
    dnd.drops.set(options.element, options);
    return () => dnd.drops.delete(options.element);
  },
  draggable: (options: DragRegistration) => {
    dnd.drags.set(options.element, options);
    return () => dnd.drags.delete(options.element);
  },
}));
const fixture = vi.hoisted(() => ({
  app: undefined as unknown,
  role: 15 as number | undefined,
  params: { workspaceSlug: "ws", projectId: "A", userId: "actor", profileViewId: "assigned" },
}));
vi.mock("@/lib/store-context", async () => {
  const { createContext } = await import("react");
  return {
    StoreContext: createContext(undefined),
    get rootStore() {
      return fixture.app;
    },
  };
});
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "A" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "A" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => fixture.params,
}));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: () => fixture.role,
    getWorkspaceRoleByWorkspaceSlug: () => fixture.role,
    allowPermissions: () => true,
  }),
  useUser: () => ({ data: { id: "actor" } }),
  useUserProfile: () => ({ data: { start_of_the_week: 0 } }),
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-10-15T12:00:00Z"));
  vi.stubGlobal("IntersectionObserver", VisibleObserver);
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  dnd.drops.clear();
  dnd.drags.clear();
  toasts.length = 0;
  tab = await issueStoreTab();
  fixture.app = tab.app;
  fixture.role = 15;
  fixture.params.projectId = "A";
  runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "A", userId: "actor" }));
  container = document.createElement("div");
  document.body.append(container);
  view = createRoot(container);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
function mount(child = <ProjectLayoutRoot />) {
  view.render(
    <StoreContext.Provider value={tab.app}>
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter>{child}</MemoryRouter>
      </SWRConfig>
    </StoreContext.Provider>
  );
}
it("mountedProjectListUsesGeneratedRequest waits for saved preferences and sends their query", async () => {
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  expect(tab.nerve.calls[0]).toMatchObject({ method: "GET", path: "/api/v0/me/projects/A/preferences" });
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "list"));
  expect(tab.nerve.calls[1]).toMatchObject({
    method: "GET",
    path: "/api/v0/projects/A/issues",
    query: { priority: "high", order_by: "-created_at", sub_issue: "true", limit: "100" },
  });
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});

it("the real project layout reports a last-row delete once after its row unmounts", async () => {
  // The observed menu and dialog are visible; jsdom does not provide layout geometry.
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 100, 30));
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, preferencesOf())));
  await act(async () => until(() => tab.nerve.calls.length === 2, "project list"));
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf([issueDetailOf("only")]))));
  const options = document.querySelector<HTMLButtonElement>('[aria-label="common.options: only"]');
  expect(options).not.toBeNull();
  await act(async () => options!.click());
  const remove = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (item) => item.textContent === "common.actions.delete"
  );
  expect(remove).toBeDefined();
  await act(async () => remove!.click());
  const confirm = [...document.querySelectorAll<HTMLButtonElement>('[role="dialog"] button')].find(
    (candidate) => candidate.textContent === "delete"
  );
  expect(confirm).toBeDefined();
  await act(async () => {
    confirm!.click();
    confirm!.click();
  });
  const deletes = () => tab.nerve.to("/api/v0/issues/only").filter((call) => call.method === "DELETE");
  await until(() => deletes().length === 1, "project delete");
  await act(async () => deletes()[0]!.answer(noContent()));
  expect(document.querySelector("#issue-only")).toBeNull();
  expect(toasts).toEqual([expect.objectContaining({ type: "success", message: "entity.delete.success" })]);
  expect(deletes()).toHaveLength(1);
});
it.each([
  ["list", "month", "100"],
  ["kanban", "month", "30"],
  ["spreadsheet", "month", "100"],
  ["calendar", "month", "4"],
  ["calendar", "week", "30"],
] as const)("the actual %s layout (%s) mounts its first query", async (layout, calendar, limit) => {
  const prefs = preferencesOf();
  prefs.display_filters.layout = layout;
  prefs.display_filters.calendar.layout = calendar;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, prefs));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "layout list"));
  const query = tab.nerve.calls[1]!.query;
  expect(query).toMatchObject({ order_by: "-created_at", sub_issue: "true", limit });
  if (layout === "calendar") {
    const dates = tab.app.issue.issueCalendarView.getStartAndEndDate(calendar)!;
    expect(query).toMatchObject({ group_by: "target_date", target_date: `${dates.startDate}..${dates.endDate}` });
  }
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});

it("a rejected preference displays an error without a list request", async () => {
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(problem(403, "permission_denied"));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(tab.nerve.calls).toHaveLength(1);
});

it.each([
  ["archived", <ArchivedIssueLayoutRoot key="archived" />],
  ["profile", <ProfileIssuesPage key="profile" type="assigned" />],
] as const)("guest cannot mount %s list", async (_name, child) => {
  fixture.role = 5;
  await act(async () => {
    mount(child);
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toEqual([]);
});
it("a non-member cannot mount a project list", async () => {
  fixture.role = undefined;
  await act(async () => {
    mount();
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(tab.nerve.calls).toEqual([]);
});
it("a guest project list leaves own-only enforcement to the server", async () => {
  fixture.role = 5;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "guest preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf()));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "guest list"));
  expect(tab.nerve.calls[1]!.query).not.toHaveProperty("assignee_id");
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
});
it.each([
  ["archived", <ArchivedIssueLayoutRoot key="archived" />, "/api/v0/projects/A/issues"],
  ["profile", <ProfileIssuesPage key="profile" type="assigned" />, "/api/v0/workspaces/ws/members/actor/issues"],
] as const)("member mounts the %s list", async (_name, child, path) => {
  await act(async () => mount(child));
  await act(async () => until(() => tab.nerve.calls.length === 1, "list"));
  expect(tab.nerve.calls[0]).toMatchObject({
    method: "GET",
    path,
    query: _name === "archived" ? { archived: "true" } : { relation: "assigned" },
  });
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, issuePageOf([])));
  });
});

it("switching projects waits for the new preferences before selecting its list", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  tab.nerve.calls.length = 0;
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "A preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 2, "A list"));
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
  });
  fixture.params.projectId = "B";
  await act(async () => {
    runInAction(() => tab.app.router.setQuery({ workspaceSlug: "ws", projectId: "B", userId: "actor" }));
    mount();
  });
  await act(async () => until(() => tab.nerve.calls.length === 3, "B preferences"));
  expect(tab.nerve.calls[2]).toMatchObject({ path: "/api/v0/me/projects/B/preferences" });
  await act(async () => {
    tab.nerve.calls[2]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
  });
  await act(async () => until(() => tab.nerve.calls.length === 4, "B list"));
  expect(tab.nerve.calls[3]).toMatchObject({ path: "/api/v0/projects/B/issues", query: { priority: "low" } });
  await act(async () => {
    tab.nerve.calls[3]!.answer(json(200, issuePageOf([])));
  });
});

it.each([
  [["2026-10-15"], "2026-10-15"],
  [["2026-10-10..2026-11-10"], "2026-10-10..2026-10-31"],
  [["2026-11-02.."], undefined],
  [["none"], undefined],
] as const)("calendar intersects saved dates %s with its actual window", async (dates, expected) => {
  vi.setSystemTime(new Date("2026-10-15T12:00:00Z"));
  const prefs = preferencesOf({ filters: { target_date: [...dates] } });
  prefs.display_filters.layout = "calendar";
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => {
    tab.nerve.calls[0]!.answer(json(200, prefs));
    await vi.advanceTimersByTimeAsync(100);
  });
  const calls = tab.nerve.to("/api/v0/projects/A/issues");
  expect(calls).toHaveLength(expected ? 1 : 0);
  if (expected) {
    expect(calls[0]!.queryEntries.filter(([key]) => key === "target_date")).toEqual([["target_date", expected]]);
    await act(async () =>
      calls[0]!.answer(
        json(200, {
          total: 1,
          groups: [
            {
              key: "2026-10-15",
              total: 1,
              data: [{ ...issueDetailOf("in-window"), target_date: "2026-10-15" }],
              next_cursor: null,
            },
          ],
        })
      )
    );
  }
  expect(tab.app.issue.projectIssues.total).toBe(expected ? 1 : 0);
  if (expected) expect(tab.app.issue.projectIssues.getIssueIds("2026-10-15")).toEqual(["in-window"]);
  expect(tab.app.issue.projectIssues.groups?.size).toBe(expected ? 1 : 0);
  expect(tab.app.issue.projectIssuesFilter.getIssueFilters("A")?.richFilters).toEqual({
    and: dates.map((value) =>
      value.includes("..") ? { target_date__range: value.replace("..", ",") } : { target_date__exact: value }
    ),
  });
});

async function mountWithPreferences(prefs = preferencesOf()) {
  await act(async () => mount());
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await act(async () => until(() => tab.nerve.calls.length === 2, "list"));
}
it("actual calendar producer and day drop use newly created entity and refreshed start date", async () => {
  const prefs = preferencesOf();
  prefs.display_filters.layout = "calendar";
  await mountWithPreferences(prefs);
  await act(async () => tab.nerve.calls[1]!.answer(json(200, { total: 0, groups: [] })));
  // A later first page replaces the initially empty immutable issues map.
  await act(async () => mount());
  let creation: Promise<unknown>;
  await act(async () => {
    creation = tab.app.issue.projectIssues.createIssue("ws", "A", { name: "calendar-x", target_date: "2026-10-15" });
    await vi.advanceTimersByTimeAsync(10);
  });
  const issue = { ...issueDetailOf("calendar-x"), start_date: "2026-10-01", target_date: "2026-10-15" };
  await act(async () => tab.nerve.calls[2]!.answer(json(201, issue)));
  await act(async () => {
    await creation!;
  });
  const drag = [...dnd.drags.values()].find((entry) => entry.getInitialData?.({} as never).id === issue.id)!;
  expect(drag).toBeDefined();
  expect(
    [...dnd.drops.values()].some(
      (entry) =>
        entry.getData?.({ input: { clientX: 0, clientY: 0 }, element: entry.element } as never).date === "2026-10-20"
    )
  ).toBe(true);
  const day = () =>
    [...dnd.drops.values()].find(
      (entry) =>
        entry.getData?.({ input: { clientX: 0, clientY: 0 }, element: entry.element } as never).date === "2026-10-20"
    )!;
  const drop = () => {
    const target = day();
    target.onDrop!({
      source: { data: drag.getInitialData!({} as never), element: drag.element },
      self: { data: target.getData!({} as never) },
    } as never);
  };
  await act(async () => {
    drop();
    await vi.advanceTimersByTimeAsync(20);
  });
  const writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes).toHaveLength(1);
  expect(writes[0]).toMatchObject({ path: "/api/v0/issues/calendar-x", body: { target_date: "2026-10-20" } });
  await act(async () =>
    writes[0]!.answer(json(200, { ...issue, target_date: "2026-10-20", updated_at: "2026-10-29T00:00:00Z" }))
  );
  await act(async () =>
    tab.app.issue.acceptListIssues(
      [{ ...issue, start_date: "2026-10-25", updated_at: "2026-10-30T00:00:00Z" }],
      tab.app.issue.captureRead()
    )
  );
  await act(async () => {
    drop();
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
});
it("actual ungrouped ListGroup and card producers reorder through parsers and root", async () => {
  const prefs = preferencesOf();
  prefs.display_filters.order_by = "sort_order";
  await mountWithPreferences(prefs);
  const first = { ...issueDetailOf("first"), sort_order: 10 };
  const second = { ...issueDetailOf("second"), sort_order: 20 };
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf([first, second]))));
  const column = [...dnd.drops.values()].find(
    (entry) => entry.getData?.({ input: { clientX: 0, clientY: 0 }, element: entry.element } as never).type === "COLUMN"
  )!;
  const drag = [...dnd.drags.values()].find((entry) => entry.getInitialData?.({} as never).id === "first")!;
  expect(column.getData!({} as never)).toMatchObject({ groupId: "", type: "COLUMN" });
  expect(drag.getInitialData!({} as never)).toMatchObject({ id: "first", groupId: "", type: "ISSUE" });
  const target = { data: column.getData!({} as never), element: column.element };
  await act(async () => {
    column.onDrop!({
      source: { data: drag.getInitialData!({} as never), element: drag.element },
      location: { initial: { dropTargets: [target] }, current: { dropTargets: [target] } },
    } as never);
    await vi.advanceTimersByTimeAsync(20);
  });
  const writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes).toHaveLength(1);
  expect(writes[0]).toMatchObject({ path: "/api/v0/issues/first", body: { sort_order: 65555 } });
  await act(async () =>
    writes[0]!.answer(json(200, { ...first, sort_order: 65555, updated_at: "2026-10-29T00:00:00Z" }))
  );
  expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["second", "first"]);
});

function mountedFilter() {
  return tab.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT, "A")!;
}
it("actual filter menu excludes unsupported created and updated dates", async () => {
  await mountWithPreferences();
  await act(async () => {
    tab.nerve.calls[1]!.answer(json(200, issuePageOf([])));
    mountedFilter().toggleVisibility(true);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(100);
  });
  const button = container.querySelector<HTMLButtonElement>('button[id^="headlessui-popover-button"]')!;
  expect(button).not.toBeNull();
  // The visible test button has geometry; Headless UI correctly closes a zero-sized anchor.
  vi.spyOn(button, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 100, 30));
  await act(async () => {
    button.focus();
    button.click();
  });
  const options = [...document.querySelectorAll('[role="option"]')].map((node) => node.textContent?.toLowerCase());
  expect(options).toContain("common.filter_options.fields.target_date");
  expect(options).not.toContain("common.filter_options.fields.created_at");
  expect(options).not.toContain("common.filter_options.fields.updated_at");
});
it("an unchanged preference confirmation retains the focused condition and marks its baseline saved", async () => {
  await mountWithPreferences();
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf([]))));
  const filter = mountedFilter();
  await act(async () => {
    filter.toggleVisibility(true);
    filter.resetExpression({ and: [{ priority__in: "high" }] }, false);
    await vi.advanceTimersByTimeAsync(100);
  });
  const button = [...container.querySelectorAll<HTMLButtonElement>("button")].find(
    (node) => node.textContent?.trim().toLowerCase() === "high"
  );
  expect(button).toBeDefined();
  button!.focus();
  expect(document.activeElement).toBe(button);
  expect(filter.hasChanges).toBe(true);
  const write = tab.nerve.calls.find((call) => call.method === "PATCH")!;
  await act(async () => {
    write.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
    await vi.advanceTimersByTimeAsync(100);
  });
  expect(document.activeElement).toBe(button);
  expect(button!.isConnected).toBe(true);
  expect(filter.hasChanges).toBe(false);
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
});

it.each([false, true])("confirmed cleaned filter is visible and cannot overwrite new input newer=%s", async (newer) => {
  await mountWithPreferences();
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf([]))));
  const filter = mountedFilter();
  await act(async () => {
    filter.toggleVisibility(true);
    filter.resetExpression({ and: [{ priority__in: "high" }, { created_at__exact: "2026-10-15" }] });
    await vi.advanceTimersByTimeAsync(10);
  });
  const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes()).toHaveLength(1);
  expect(writes()[0]!.body).toEqual({ filters: { priority: ["high"] } });
  if (newer)
    await act(async () =>
      filter.resetExpression({ and: [{ priority__in: "low" }, { updated_at__exact: "2026-10-16" }] })
    );
  await act(async () => {
    writes()[0]!.answer(json(200, preferencesOf({ filters: { priority: ["high"] } })));
    await vi.advanceTimersByTimeAsync(20);
  });
  const current = filter.allConditions.map((condition) => [condition.property, condition.value]);
  expect(current).toEqual(
    newer
      ? [
          ["priority", "low"],
          ["updated_at", "2026-10-16"],
        ]
      : [["priority", "high"]]
  );
  expect(tab.nerve.to("/api/v0/projects/A/issues").at(-1)!.query).toMatchObject({ priority: "high" });
  await act(async () =>
    tab.nerve
      .to("/api/v0/projects/A/issues")
      .at(-1)!
      .answer(json(200, issuePageOf([])))
  );
  if (newer) {
    expect(writes()).toHaveLength(2);
    expect(writes()[1]!.body).toEqual({ filters: { priority: ["low"] } });
    await act(async () => {
      writes()[1]!.answer(json(200, preferencesOf({ filters: { priority: ["low"] } })));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(filter.allConditions.map((condition) => [condition.property, condition.value])).toEqual([
      ["priority", "low"],
    ]);
    await act(async () =>
      tab.nerve
        .to("/api/v0/projects/A/issues")
        .at(-1)!
        .answer(json(200, issuePageOf([])))
    );
  }
  await act(async () => view.render(null));
  await act(async () => mount());
  expect(mountedFilter().allConditions.map((condition) => [condition.property, condition.value])).toEqual([
    ["priority", newer ? "low" : "high"],
  ]);
  expect(writes()).toHaveLength(newer ? 2 : 1);
});

it.each(["calendar", "list"] as const)(
  "grouped %s first answer renders instead of looking for an empty group key",
  async (layout) => {
    const prefs = preferencesOf();
    prefs.display_filters.layout = layout;
    prefs.display_filters.group_by = layout === "calendar" ? "target_date" : "priority";
    await mountWithPreferences(prefs);
    const issue = { ...issueDetailOf("grouped-visible"), priority: "high" as const, target_date: "2026-10-15" };
    await act(async () =>
      tab.nerve.calls[1]!.answer(
        json(200, {
          total: 1,
          groups: [{ key: layout === "calendar" ? "2026-10-15" : "high", total: 1, data: [issue], next_cursor: null }],
        })
      )
    );
    expect(container.textContent).toContain("grouped-visible");
    expect(
      [...dnd.drops.values()].some((entry) => {
        const data = entry.getData?.({ input: { clientX: 0, clientY: 0 }, element: entry.element } as never);
        return layout === "calendar" ? data?.date === "2026-10-20" : data?.groupId === "high";
      })
    ).toBe(true);
  }
);

it.each(["", undefined, null, 0, {}])("drop parsers distinguish legal empty group keys from invalid %s", (groupId) => {
  const target = { data: { type: "COLUMN", groupId } };
  const payload = {
    source: { data: { id: "x", type: "ISSUE", groupId } },
    location: { initial: { dropTargets: [target] }, current: { dropTargets: [target] } },
  };
  expect(Boolean(getSourceFromDropPayload(payload as never))).toBe(groupId === "");
  expect(Boolean(getDestinationFromDropPayload(payload as never))).toBe(groupId === "");
});
it("a refused filter save keeps the current draft without refreshing or retrying PATCH", async () => {
  await mountWithPreferences();
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf([]))));
  const filter = mountedFilter();
  await act(async () => {
    filter.resetExpression({ and: [{ priority__in: "high" }] });
    await vi.advanceTimersByTimeAsync(10);
  });
  await act(async () => {
    tab.nerve.calls[2]!.answer(problem(403, "permission_denied"));
    await vi.advanceTimersByTimeAsync(30);
  });
  expect(filter.allConditions.map((condition) => [condition.property, condition.value])).toEqual([
    ["priority", "high"],
  ]);
  expect(tab.app.issue.projectIssuesFilter.getIssueFilters("A")?.richFilters).toEqual({});
  expect(tab.nerve.calls).toHaveLength(3);
});

const dataOf = (entry: DropRegistration) =>
  entry.getData!({ input: { clientX: 50, clientY: 90 }, element: entry.element } as never);
it("actual middle list row exposes a bottom indicator and drops below through the shared root", async () => {
  const prefs = preferencesOf();
  prefs.display_filters.order_by = "sort_order";
  await mountWithPreferences(prefs);
  const rows = ["x", "a", "b", "c"].map((id, i) => Object.assign(issueDetailOf(id), { sort_order: (i + 1) * 1000 }));
  await act(async () => tab.nerve.calls[1]!.answer(json(200, issuePageOf(rows))));
  const column = [...dnd.drops.values()].find((entry) => dataOf(entry).type === "COLUMN")!;
  const row = [...dnd.drops.values()].find((entry) => dataOf(entry).id === "b")!;
  vi.spyOn(row.element, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 0, 100, 100));
  const drag = [...dnd.drags.values()].find((entry) => entry.getInitialData?.({} as never).id === "x")!;
  const target = { element: column.element, data: dataOf(column) };
  const card = { element: row.element, data: dataOf(row) };
  await act(async () => row.onDrag!({ self: card } as never));
  expect(
    row.element.lastElementChild?.classList.contains("bg-accent-primary"),
    "middle row has its bottom drop indicator"
  ).toBe(true);
  const payload = {
    source: { element: drag.element, data: drag.getInitialData!({} as never) },
    location: { initial: { dropTargets: [target] }, current: { dropTargets: [card, target] } },
  };
  expect(getDestinationFromDropPayload(payload as never)?.canAddIssueBelow).toBe(true);
  await act(async () => {
    column.onDrop!(payload as never);
    await until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "middle drop");
  });
  const call = tab.nerve.calls.find((entry) => entry.method === "PATCH")!;
  expect(call.body).toEqual({ sort_order: 3500 });
  await act(async () =>
    call.answer(json(200, { ...issueDetailOf("x"), sort_order: 3500, updated_at: "2026-10-29T00:00:00Z" }))
  );
  expect(tab.app.issue.projectIssues.getIssueIds()).toEqual(["a", "b", "x", "c"]);
});
