// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { accountOf } from "@/store/member/fake-users";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ArchivedIssuesHeader } from "../../archived-issues-header";
import { ArchivedIssueLayoutRoot } from "./archived-issue-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "archive-page" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "archive-page" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Only the browser viewport transport is replaced; the header, filters, list and rows are real.
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const archived = { ...issueOf("old"), archived_at: "2026-10-30T00:00:00Z" };
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (b) => b.getAttribute("aria-label") === name || b.textContent?.trim() === name
  );
const click = (name: string) =>
  act(async () => {
    const found = button(name);
    expect(found, name).toBeDefined();
    found!.click();
  });
beforeEach(async () => {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  localStorage.clear();
  tab = await issueSelectionTab();
  bound.app = tab.app;
  runInAction(() => {
    tab.app.user.data = accountOf("alice");
  });
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
  toasts.length = 0;
});
afterEach(async () => {
  await tab.close();
  localStorage.clear();
  vi.unstubAllGlobals();
});
async function mount() {
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ArchivedIssuesHeader />
        <ArchivedIssueLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
}
async function loaded() {
  await mount();
  await act(async () => until(() => lists().length === 1, "archive list"));
  await act(async () => lists()[0]!.answer(json(200, issuePageOf([archived]))));
}
it("does not mount the guest toolbar, list or filter directories", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  tab.nerve.calls.length = 0;
  await mount();
  expect(button("common.display")).toBeUndefined();
  expect(tab.nerve.calls).toEqual([]);
  expect(document.querySelector('[id^="issue_"]')).toBeNull();
});
it("restores local archive filters and renders properties read-only without a preference PATCH", async () => {
  localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "ARCHIVED",
        workspaceSlug: "ws",
        viewId: "A",
        userId: "u-alice",
        filters: {
          richFilters: { priority__in: "none" },
          display_filters: { layout: "kanban", group_by: null, sub_group_by: "priority" },
        },
      },
    ])
  );
  await loaded();
  expect(lists()[0]!.query).toMatchObject({ archived: "true", priority: "none" });
  const row = document.querySelector("#issue-old")!;
  expect(row).not.toBeNull();
  const priority = [...row.querySelectorAll<HTMLButtonElement>("button")].find((b) =>
    b.textContent?.startsWith("common.priority:")
  )!;
  expect(priority).toBeDefined();
  expect(priority.disabled).toBe(true);
  await act(async () => priority.click());
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.querySelector('[aria-label="Add work item"]')).toBeNull();
  await click("common.display");
  await click("common.state");
  expect(tab.app.issue.archivedIssuesFilter.getIssueFilters("A")?.displayProperties?.state).toBe(false);
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("shows a first-read refusal and retries only the archive GET", async () => {
  await mount();
  await act(async () => until(() => lists().length === 1, "archive list"));
  await act(async () => lists()[0]!.answer(problem(503, "unavailable")));
  expect(button("common.retry")).toBeDefined();
  await click("common.retry");
  await act(async () => until(() => lists().length === 2, "archive retry"));
  await act(async () => lists()[1]!.answer(json(200, issuePageOf([archived]))));
  expect(document.querySelector("#issue-old")).not.toBeNull();
  expect(lists().every((call) => call.query.archived === "true")).toBe(true);
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("keeps a locally selected order after its read fails, reports once and retries only GET", async () => {
  await loaded();
  await click("common.display");
  await click("common.order_by.last_created");
  await act(async () => until(() => lists().length === 2, "changed archive query"));
  await act(async () => lists()[1]!.answer(problem(503, "unavailable")));
  expect(button("common.order_by.last_created")?.getAttribute("aria-pressed")).toBe("true");
  expect(toasts.filter((t) => t.type === "error")).toHaveLength(1);
  await click("common.retry");
  await act(async () => until(() => lists().length === 3, "archive GET retry"));
  await act(async () => lists()[2]!.answer(json(200, issuePageOf([archived]))));
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  expect(lists()[2]!.query).toMatchObject({ archived: "true", order_by: "-created_at" });
});

it("retains an archive continuation and retries only its refused next page", async () => {
  localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "ARCHIVED",
        workspaceSlug: "ws",
        viewId: "A",
        userId: "u-alice",
        filters: { display_filters: { group_by: null } },
      },
    ])
  );
  await mount();
  await act(async () => until(() => lists().length === 1, "archive first page"));
  expect(lists()[0]!.query.group_by).toBeUndefined();
  await act(async () => lists()[0]!.answer(json(200, { ...issuePageOf([archived], "archive-next"), total: 2 })));
  await click("common.load_more ↓");
  await act(async () => until(() => lists().length === 2, "archive next page"));
  await act(async () => lists()[1]!.answer(problem(503, "unavailable")));
  expect(document.querySelector("#issue-old")).not.toBeNull();
  await click("common.retry");
  await act(async () => until(() => lists().length === 3, "archive next retry"));
  await act(async () =>
    lists()[2]!.answer(json(200, { ...issuePageOf([{ ...archived, id: "next", sort_order: 2000 }]), total: 2 }))
  );
  expect(lists()[2]!.query).toMatchObject({ archived: "true", cursor: "archive-next" });
  expect(document.querySelector("#issue-next")).not.toBeNull();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
