// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect, useState, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects, stateOf } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page";
import IdEntry from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "detail-route" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "detail-route" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("next-themes", () => ({ useTheme: () => ({ resolvedTheme: "light" }) }));
vi.mock("@/components/core/page-title", () => ({ PageHead: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: ({ header }: { header: ReactNode }) => <HeaderBoundary>{header}</HeaderBoundary>,
}));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/hooks/use-navigation-preferences", () => ({
  useProjectNavigationPreferences: () => ({ preferences: { navigationMode: "SIDEBAR" } }),
}));
// Keep the real header actions and delete dialog. Only the independent body leaf uses
// a stateful popover; real body owners are covered by detail-host.test.tsx.
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));
vi.mock("@/components/browse/workItem-detail", () => ({
  WorkItemDetailRoot: ({ issueId }: { issueId: string }) => (
    <main data-issue={issueId}>
      <FiltersDropdown title="Body actions">
        <span>Body {issueId}</span>
      </FiltersDropdown>
    </main>
  ),
}));
vi.mock("@/layouts/auth-layout/project-wrapper", () => ({
  ProjectAuthWrapper: ({ children }: { children: ReactNode }) => children,
}));

// A route-ownership probe above the issue-specific actions: the entire header subtree
// must retire, including local state that does not disappear during a body reload.
function HeaderBoundary({ children }: { children: ReactNode }) {
  const [note, setNote] = useState("");
  return (
    <header>
      <input aria-label="Header local state" value={note} onInput={(event) => setNote(event.currentTarget.value)} />
      {children}
    </header>
  );
}

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter>;
let holdIdentifier = false;
let refusal: number | undefined;
const source = (id: string) => ({ ...issueDetailOf(id), sequence_id: id === "one" ? 1 : 2 });
const params = (values: Readonly<Record<string, string | undefined>>) => ({
  workspaceSlug: values.workspaceSlug ?? "",
  workItem: values.workItem ?? "",
  projectId: values.projectId ?? "",
  issueId: values.issueId ?? "",
});
function RouteLayout() {
  const values = useParams();
  useEffect(() => tab.app.router.setQuery(values), [values]);
  return <Layout params={params(values)} />;
}
function RoutePage() {
  return <Page params={params(useParams())} />;
}
function RouteIdEntry() {
  return <IdEntry params={params(useParams())} />;
}
const trigger = (name: "Header" | "Body") =>
  document.querySelector<HTMLButtonElement>(
    name === "Header" ? 'header button[aria-label^="common.options:"]' : 'button[aria-label="Body actions"]'
  );
const headerInput = () => document.querySelector<HTMLInputElement>('input[aria-label="Header local state"]')!;
const lookupReads = () => tab.nerve.calls.filter((call) => call.path.startsWith("/api/v0/workspaces/ws/issues/"));
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 20 }]);
  holdIdentifier = false;
  refusal = undefined;
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/workspaces/ws/projects") return;
    if (call.path.startsWith("/api/v0/workspaces/ws/issues/")) {
      if (!holdIdentifier)
        call.answer(
          refusal ? problem(refusal, "issue.not_found") : json(200, source(call.path.endsWith("-1") ? "one" : "two"))
        );
    } else if (/^\/api\/v0\/issues\/[^/]+$/.test(call.path)) {
      call.answer(refusal ? problem(refusal, "issue.not_found") : json(200, source(call.path.split("/").at(-1)!)));
    } else if (call.path.endsWith("/states")) {
      call.answer(json(200, { data: [stateOf(tab.project, "Todo", "unstarted", 1, { id: "state" })] }));
    } else call.answer(json(200, { data: [], next_cursor: null }));
  };
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
});
async function mount(path = "/ws/browse/A-1") {
  router = createMemoryRouter(
    [
      {
        path: "/:workspaceSlug/browse/:workItem",
        Component: RouteLayout,
        children: [{ index: true, Component: RoutePage }],
      },
      { path: "/:workspaceSlug/projects/:projectId/issues/:issueId", Component: RouteIdEntry },
    ],
    { initialEntries: [path] }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
}
async function ready(id = "one") {
  await act(async () =>
    until(() => document.querySelector("main")?.getAttribute("data-issue") === id, "current route body")
  );
  expect(trigger("Header")).not.toBeNull();
}

it("keepsNativeIdRedirect with a replaced history entry and the proven project identifier", async () => {
  await mount("/ws/projects/A/issues/one");
  await ready();
  expect(router.state.location.pathname).toBe("/ws/browse/A-1");
  expect(router.state.historyAction).toBe("REPLACE");
  expect(tab.nerve.calls[0]?.path).toBe("/api/v0/issues/one");
  expect(lookupReads().map((call) => call.path)).toEqual(["/api/v0/workspaces/ws/issues/A-1"]);
});
it.each([401, 403, 404])("does not redirect a refused native id (%s)", async (status) => {
  refusal = status;
  await mount("/ws/projects/A/issues/one");
  await act(async () => until(() => document.querySelector('[role="alert"]') !== null, "visible refusal"));
  expect(router.state.location.pathname).toBe("/ws/projects/A/issues/one");
  expect(lookupReads()).toHaveLength(0);
});
it.each(["Header", "Body"] as const)(
  "retiresHeaderAndBodyTogether: %s state does not survive A to B to A",
  async (name) => {
    await act(async () => {
      expect(
        (await settle(tab.app.issue.issueDetail.fetchIssueWithIdentifier("ws", "A-2"), "warm target")).error
      ).toBeUndefined();
    });
    await mount();
    await ready();
    await act(async () => {
      trigger(name)!.click();
      await vi.advanceTimersByTimeAsync(30);
    });
    expect(trigger(name)!.getAttribute("aria-expanded")).toBe("true");
    const panel = trigger(name)!.getAttribute("aria-controls")!;
    await act(async () => router.navigate("/ws/browse/A-2"));
    await ready("two");
    expect(document.getElementById(panel)).toBeNull();
    expect(trigger(name)!.getAttribute("aria-expanded")).toBe("false");
    await act(async () => router.navigate("/ws/browse/A-1"));
    await ready();
    expect(trigger(name)!.getAttribute("aria-expanded")).toBe("false");
  }
);
it("withholds cached header commands during the current read and after its refusal", async () => {
  await act(async () => {
    const result = await settle(tab.app.issue.issueDetail.fetchIssueWithIdentifier("ws", "A-1"), "cached item");
    expect(result.error).toBeUndefined();
  });
  tab.nerve.calls.length = 0;
  holdIdentifier = true;
  await mount();
  await act(async () => until(() => lookupReads().length === 1, "held lookup"));
  expect(trigger("Header")).toBeNull();
  expect(trigger("Body")).toBeNull();
  await act(async () => {
    lookupReads()[0]!.answer(problem(404, "issue.not_found"));
    await vi.advanceTimersByTimeAsync(30);
  });
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(trigger("Header")).toBeNull();
  expect(trigger("Body")).toBeNull();
});
it("ends header state on permission ABA without restoring the old menu", async () => {
  await mount();
  await ready();
  await act(async () => {
    trigger("Header")!.click();
    await vi.advanceTimersByTimeAsync(30);
  });
  const panel = trigger("Header")!.getAttribute("aria-controls")!;
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  });
  await ready();
  expect(document.getElementById(panel)).toBeNull();
  expect(trigger("Header")!.getAttribute("aria-expanded")).toBe("false");
});

it("ends the real header delete dialog when a warm target replaces the route", async () => {
  await act(async () => {
    expect(
      (await settle(tab.app.issue.issueDetail.fetchIssueWithIdentifier("ws", "A-2"), "warm target")).error
    ).toBeUndefined();
  });
  await mount();
  await ready();
  await act(async () => {
    trigger("Header")!.click();
    await vi.advanceTimersByTimeAsync(30);
  });
  const remove = [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find((item) =>
    item.textContent?.includes("common.actions.delete")
  );
  expect(remove).toBeDefined();
  await act(async () => {
    remove!.click();
    await vi.advanceTimersByTimeAsync(30);
  });
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  await act(async () => router.navigate("/ws/browse/A-2"));
  await ready("two");
  expect(document.querySelector('[role="dialog"]')).toBeNull();
});

it("retires all header-local state at the common route boundary", async () => {
  await mount();
  await ready();
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(headerInput(), "previous item");
    headerInput().dispatchEvent(new Event("input", { bubbles: true }));
  });
  expect(headerInput().value).toBe("previous item");
  await act(async () => router.navigate("/ws/browse/A-2"));
  await ready("two");
  expect(headerInput().value).toBe("");
  await act(async () => router.navigate("/ws/browse/A-1"));
  await ready();
  expect(headerInput().value).toBe("");
});
