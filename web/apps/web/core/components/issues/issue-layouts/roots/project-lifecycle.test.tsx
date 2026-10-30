// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueOf, issueDetailOf } from "@/store/issue/fake-issues";
import { preferencesOf, stateOf, loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ProjectLayoutRoot } from "./project-layout-root";

type Drop = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").dropTargetForElements>[0];
type Drag = Parameters<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter").draggable>[0];
const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  drops: new Map<Element, Drop>(),
  drags: new Map<Element, Drag>(),
  scrolls: new Set<Element>(),
}));
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
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Only browser viewport/drag transport are replaced; root, loading gate, rows, modal and stores are real.
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) => (as === "tr" ? <tr>{children}</tr> : children),
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({
  autoScrollForElements: ({ element }: { element: Element }) => {
    bound.scrolls.add(element);
    return () => bound.scrolls.delete(element);
  },
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", async (original) => ({
  ...(await original<typeof import("@atlaskit/pragmatic-drag-and-drop/element/adapter")>()),
  dropTargetForElements: (options: Drop) => {
    bound.drops.set(options.element, options);
    return () => bound.drops.delete(options.element);
  },
  draggable: (options: Drag) => {
    bound.drags.set(options.element, options);
    return () => bound.drags.delete(options.element);
  },
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const parent = { ...issueOf("parent"), state_id: "state", sub_issues_count: 1 };
const list = (layout: "list" | "kanban" | "spreadsheet") => ({
  total: 1,
  groups: [{ key: layout === "kanban" ? "state" : "", total: 1, data: [parent], next_cursor: null }],
});
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const deletes = () => tab.nerve.calls.filter((call) => call.method === "DELETE");
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: 20 }));
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  bound.drops.clear();
  bound.drags.clear();
  bound.scrolls.clear();
  toasts.length = 0;
});
afterEach(async () => tab.close());
async function mount(layout: "list" | "kanban" | "spreadsheet", answer = true) {
  const prefs = preferencesOf();
  prefs.display_filters.layout = layout;
  prefs.display_filters.group_by = layout === "kanban" ? "state_id" : null;
  prefs.display_filters.order_by = "sort_order";
  prefs.display_properties = {
    key: true,
    assignee: false,
    start_date: false,
    due_date: false,
    labels: false,
    priority: false,
    state: false,
    sub_issue_count: false,
    link: false,
    created_on: false,
    updated_on: false,
  };
  await tab.render(
    <StrictMode>
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter>
          <ProjectLayoutRoot />
        </MemoryRouter>
      </SWRConfig>
    </StrictMode>
  );
  await until(() => tab.nerve.calls.length === 1, "preferences");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, prefs)));
  await until(() => lists().length === 1, "initial list");
  if (answer) await act(async () => lists()[0]!.answer(json(200, list(layout))));
}
async function requestDelete(id = "parent") {
  const source = [...bound.drags.values()].find((drag) => drag.getInitialData!({} as never).id === id)!;
  expect(source).toBeDefined();
  const target = [...bound.drops.values()].find((drop) => drop.getData!({} as never).type === "DELETE")!;
  await act(async () =>
    target.onDrop!({
      source: { element: source.element, data: source.getInitialData!({} as never) },
      location: { initial: { dropTargets: [{ data: { type: "COLUMN", groupId: "state" } }] } },
    } as never)
  );
}
const dialog = () => document.querySelector('[role="dialog"][aria-modal="true"]');
const confirmDelete = () =>
  [...dialog()!.querySelectorAll("button")].find((button) => button.textContent === "Delete")!;
it("registers kanban auto-scroll when the actual container arrives and releases it on host end", async () => {
  await mount("kanban", false);
  expect(bound.scrolls.size).toBe(0);
  await act(async () => lists()[0]!.answer(json(200, list("kanban"))));
  const container = document.querySelector(".horizontal-scrollbar")!;
  expect(bound.scrolls.has(container)).toBe(true);
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/projects/A/issues") call.answer(json(200, list("kanban")));
  };
  let verified!: Promise<unknown>;
  await act(async () => {
    verified = tab.app.issue.issues.refreshAuthoritative("parent");
  });
  expect(bound.scrolls.has(container)).toBe(false);
  await act(async () => {
    tab.nerve.to("/api/v0/issues/parent")[0]!.answer(json(200, { ...issueDetailOf("parent"), sub_issues_count: 1 }));
    await verified;
  });
  const replacement = document.querySelector(".horizontal-scrollbar")!;
  expect(replacement).not.toBe(container);
  expect(bound.scrolls.has(replacement)).toBe(true);
  await tab.render(null);
  expect(bound.scrolls.size).toBe(0);
});
it("a refused drag deletion stays open with one error and only a confirmed retry closes it", async () => {
  await mount("kanban");
  await requestDelete();
  expect(confirmDelete()).toBeDefined();
  await act(async () => {
    confirmDelete().click();
    confirmDelete().click();
  });
  await until(() => deletes().length === 1, "delete");
  expect(dialog()).not.toBeNull();
  await act(async () => deletes()[0]!.answer(problem(403, "forbidden")));
  expect(dialog()).not.toBeNull();
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.forbidden" })]);
  expect(tab.app.issue.issues.getIssueById("parent")).toBeDefined();
  await act(async () => confirmDelete().click());
  await until(() => deletes().length === 2, "retry delete");
  await act(async () => deletes()[1]!.answer(new Response(null, { status: 204 })));
  expect(dialog()).toBeNull();
  expect(toasts.filter((toast) => toast.type === "success")).toHaveLength(1);
});
it("a failed first list read exposes a GET retry outside the loading gate", async () => {
  await mount("list", false);
  await act(async () => lists()[0]!.answer(problem(503, "unavailable")));
  const retry = [...document.querySelectorAll("button")].find((button) => button.textContent === "common.retry");
  expect(retry).toBeDefined();
  await act(async () => retry!.click());
  await until(() => lists().length === 2, "first page retry");
  await act(async () => lists()[1]!.answer(json(200, list("list"))));
  expect(document.querySelector('a[id="issue-parent"]')).not.toBeNull();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("spreadsheet child refusal has an accessible expansion control and a read-only retry", async () => {
  await mount("spreadsheet");
  const expand = document.querySelector<HTMLButtonElement>("#issue-parent button.grid")!;
  expect(expand).not.toBeNull();
  await act(async () => expand.click());
  const children = () => tab.nerve.to("/api/v0/issues/parent/sub-issues");
  await until(() => children().length >= 1, "children");
  await act(async () => children().forEach((call) => call.answer(problem(403, "forbidden"))));
  expect(children()).toHaveLength(1);
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.forbidden");
  expect(expand.getAttribute("aria-label")).toContain("parent");
  expect(expand.getAttribute("aria-expanded")).toBe("true");
  await act(async () => document.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
  await until(() => children().length === 2, "children retry");
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  await act(async () => children()[1]!.answer(problem(403, "forbidden")));
});
it.each(["list", "spreadsheet"] as const)(
  "%s child expansion survives its required parent revalidation",
  async (layout) => {
    await mount(layout);
    tab.nerve.respond = (call) => {
      if (call.path.endsWith("/states"))
        call.answer(json(200, { data: [stateOf(tab.project, "Todo", "unstarted", 1, { id: "state" })] }));
      if (call.path === "/api/v0/issues/parent/sub-issues")
        call.answer(json(200, { data: [{ ...issueOf("child"), state_id: "state", parent_id: "parent" }] }));
      if (call.path === "/api/v0/issues/parent")
        call.answer(json(200, { ...issueDetailOf("parent"), state_id: "state", sub_issues_count: 1 }));
      if (call.path === "/api/v0/projects/A/issues") call.answer(json(200, list(layout)));
    };
    await act(async () => document.querySelector<HTMLButtonElement>("#issue-parent button.grid")!.click());
    await act(async () =>
      until(
        () => tab.app.issue.issueDetail.subIssues.subIssuesByIssueId("parent")?.includes("child") ?? false,
        "confirmed children"
      )
    );
    expect(document.querySelector("#issue-child")).not.toBeNull();
  }
);

it.each(["list", "spreadsheet"] as const)(
  "%s expanded intent ends across a permission loss and recovery",
  async (layout) => {
    await mount(layout);
    tab.nerve.respond = (call) => {
      if (call.path.endsWith("/states"))
        call.answer(json(200, { data: [stateOf(tab.project, "Todo", "unstarted", 1, { id: "state" })] }));
      if (call.path === "/api/v0/issues/parent/sub-issues")
        call.answer(json(200, { data: [{ ...issueOf("child"), parent_id: "parent" }] }));
      if (call.path === "/api/v0/issues/parent")
        call.answer(json(200, { ...issueDetailOf("parent"), sub_issues_count: 1 }));
      if (call.path === "/api/v0/projects/A/issues") call.answer(json(200, list(layout)));
    };
    await act(async () => document.querySelector<HTMLButtonElement>("#issue-parent button.grid")!.click());
    await act(async () => until(() => !!document.querySelector("#issue-child"), "expanded child"));
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    });
    await act(async () => until(() => !!document.querySelector("#issue-parent"), "fresh parent"));
    expect(document.querySelector("#issue-child")).toBeNull();
    expect(document.querySelector("#issue-parent button.grid")?.getAttribute("aria-expanded")).toBe("false");
  }
);
it.each([true, false])("late drag deletion after host unmount has no feedback (success=%s)", async (success) => {
  await mount("kanban");
  await requestDelete();
  await act(async () => confirmDelete().click());
  await until(() => deletes().length === 1, "delete");
  await tab.render(null);
  await act(async () =>
    deletes()[0]!.answer(success ? new Response(null, { status: 204 }) : problem(403, "forbidden"))
  );
  expect(toasts).toEqual([]);
});
