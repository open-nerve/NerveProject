// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's view's page's work items (M7/P6b spec 2.3, 2.8, R7, R8): the workspace's list under the page's
// preferences, by GLOBAL, drawn as the spreadsheet alone, with the page's own preview host. Each row's properties are
// editable by the caller's role in its own project (a member's yes, a guest's no); the page creates nothing from its
// table (no quick add), and shows no cycle or modules column, which a table of no project never has. The stores, the
// layouts, SWR and HTTP are real (fake-view-page.tsx); the preview a stand-in that says its host and its store type.
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import { issueOf } from "@/store/issue/fake-issues";
import { workspacePageId } from "@/store/issue/workspace/page-id";
import type { RootStore } from "@/store/root.store";
import { WorkspaceViewLayoutRoot } from "./workspace-view-layout-root";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  peek: undefined as { hostScope: string; store: unknown } | undefined,
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
  useParams: () => ({ workspaceSlug: "ws", globalViewId: "all-issues" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("../../peek-overview", async () => {
  const { useContext } = await import("react");
  const { IssuesStoreContext } = await import("@/hooks/use-issue-layout-store");
  return {
    IssuePeekOverview: (props: { hostScope: string }) => {
      bound.peek = { hostScope: props.hostScope, store: useContext(IssuesStoreContext) };
      return null;
    },
  };
});

let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
beforeEach(async () => {
  window.localStorage.clear();
  page = await workspaceViewPageTab(15);
  bound.app = page.app;
  bound.peek = undefined;
});
afterEach(async () => page.close());

/** The page of All work items, its preferences read, and its list as nerve answers it: a row of A, a row of B. */
async function shown() {
  await page.app.issue.globalIssuesFilter.fetchFilters("ws", workspacePageId("ws", "all-issues"));
  await page.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <WorkspaceViewLayoutRoot workspaceSlug="ws" globalViewId="all-issues" />
      </MemoryRouter>
    </SWRConfig>
  );
  await page.answerList([issueOf("in-a", "A"), issueOf("in-b", "B")]);
}
/** The priority control of the row of the work item id. */
const priorityOf = (id: string) =>
  [...(document.getElementById(`issue-${id}`)?.closest("tr")?.querySelectorAll("button") ?? [])].filter((button) =>
    button.textContent?.startsWith("common.priority: ")
  );

it("draws the workspace's work items as the spreadsheet, in the page's own preview host", async () => {
  await shown();
  expect(page.lists()).toHaveLength(1);
  expect(document.querySelector("table")).not.toBeNull();
  expect(document.querySelectorAll('[id^="issue-in-"]').length).toBeGreaterThan(0);
  expect(bound.peek).toEqual({
    hostScope: JSON.stringify(["workspace-view", "ws", "all-issues"]),
    store: EIssuesStoreType.GLOBAL,
  });
});

it("lets each row's properties be changed by the caller's role in its own project: a member's yes, a guest's no", async () => {
  await shown();
  expect(priorityOf("in-a").length, "A's row's priority").toBeGreaterThan(0);
  expect(
    priorityOf("in-a").every((button) => !button.disabled),
    "A's, a member's"
  ).toBe(true);
  expect(priorityOf("in-b").length, "B's row's priority").toBeGreaterThan(0);
  expect(
    priorityOf("in-b").every((button) => button.disabled),
    "B's, a guest's"
  ).toBe(true);
});

it("creates nothing from its table, and shows no cycle or modules column", async () => {
  await shown();
  const buttons = [...document.querySelectorAll("button")].map((button) => button.textContent?.trim());
  expect(buttons).not.toContain("issue.new");
  expect(buttons).not.toContain("issue.add.label");
  const columns = document.querySelector("table thead")?.textContent ?? "";
  expect(columns).toContain("common.priority");
  expect(columns).not.toContain("common.cycle");
  expect(columns).not.toContain("common.modules");
});
