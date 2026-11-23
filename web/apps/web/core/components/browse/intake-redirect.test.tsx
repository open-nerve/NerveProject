// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// P5b spec 2.4, R14: the browse route sends a work item in its project's intake (pending, snoozed, declined, a
// duplicate) to the intake page that selects it, in the tab that lists it, once the visit's own read has answered:
// in place of its entry, so that Back does not come back to it, its header and its detail never mounted for it, and
// whether the project's intake is on or off. One accepted, or never submitted to an intake, shows its detail as
// before. The memory router's shape is detail-route.test.tsx's.
import { act, useEffect, type ComponentProps, type PropsWithChildren } from "react";
import { createMemoryRouter, RouterProvider, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IntakeStatus, IssueDetail } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import type { WorkItemDetailRoot } from "@/components/browse/workItem-detail";
import type { IssueDetailQuickActions } from "@/components/issues/issue-detail/issue-detail-quick-actions";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page";
import IdEntry from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
// Every mount of the route's header actions and of its body, by the work item each mounted for.
const mounts = vi.hoisted(() => ({ seen: [] as string[] }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "redirect" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "redirect" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("next-themes", () => ({ useTheme: () => ({ resolvedTheme: "light" }) }));
vi.mock("@/components/core/page-title", () => ({ PageHead: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/hooks/use-navigation-preferences", () => ({
  useProjectNavigationPreferences: () => ({ preferences: { navigationMode: "SIDEBAR" } }),
}));
vi.mock("@/components/issues/issue-detail/issue-detail-quick-actions", () => ({
  IssueDetailQuickActions: function MockActions({ issueId }: ComponentProps<typeof IssueDetailQuickActions>) {
    useEffect(() => {
      mounts.seen.push(`header ${issueId}`);
    }, [issueId]);
    return null;
  },
}));
vi.mock("@/components/browse/workItem-detail", () => ({
  WorkItemDetailRoot: function MockBody({ issueId }: ComponentProps<typeof WorkItemDetailRoot>) {
    useEffect(() => {
      mounts.seen.push(`body ${issueId}`);
    }, [issueId]);
    return <main data-issue={issueId} />;
  },
}));
vi.mock("@/layouts/auth-layout/project-wrapper", () => ({
  ProjectAuthWrapper: ({ children }: PropsWithChildren) => children,
}));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter>;
/** nerve's answer for the work item one, A-1, to its lookup and its read alike. */
let answer: IssueDetail;
let holdLookup = false;
const one = (intake: IntakeStatus | null, snoozedTill: string | null = null): IssueDetail => ({
  ...issueDetailOf("one"),
  intake: intake === null ? null : { status: intake, snoozed_till: snoozedTill, duplicate_to_id: null },
});
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
const lookups = () => tab.nerve.calls.filter((call) => call.path === "/api/v0/workspaces/ws/issues/A-1");
const where = () => `${router.state.location.pathname}${router.state.location.search}`;
const intakeOf = (shown: "open" | "closed") => `/ws/projects/A/intake?currentTab=${shown}&inboxIssueId=one`;

/** The project A, its intake on (intake_view) unless off. */
async function given(intakeView = true) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: 20, intake_view: intakeView },
  ]);
  tab.nerve.calls.length = 0;
}
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  await given();
  holdLookup = false;
  mounts.seen.length = 0;
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (call.path === "/api/v0/workspaces/ws/projects") return;
    if (call.path === "/api/v0/workspaces/ws/issues/A-1") {
      if (!holdLookup) call.answer(json(200, answer));
    } else if (call.path === "/api/v0/issues/one") call.answer(json(200, answer));
    else call.answer(json(200, { data: [], next_cursor: null }));
  };
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
});
/** Opens path, an entry after another page's. */
async function visit(path: string) {
  router = createMemoryRouter(
    [
      {
        path: "/:workspaceSlug/browse/:workItem",
        Component: RouteLayout,
        children: [{ index: true, Component: RoutePage }],
      },
      { path: "/:workspaceSlug/projects/:projectId/issues/:issueId", Component: RouteIdEntry },
      { path: "/:workspaceSlug/projects/:projectId/intake", element: <div>intake page</div> },
      { path: "*", element: <div>elsewhere</div> },
    ],
    { initialEntries: ["/before", path], initialIndex: 1 }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
}
/** Waits for the visit's read to have been judged: the intake page reached, or the detail shown. */
const settled = () =>
  act(async () =>
    until(
      () =>
        router.state.location.pathname === "/ws/projects/A/intake" ||
        document.querySelector("main")?.getAttribute("data-issue") === "one",
      "the intake page or the detail"
    )
  );
/** Back, from the page the route is at. */
const back = () =>
  act(async () => {
    await router.navigate(-1);
    await vi.advanceTimersByTimeAsync(30);
  });

it.each([
  { status: "pending", tab: "open" },
  { status: "snoozed", till: "2099-01-01T00:00:00Z", tab: "open" },
  // stored as snoozed, its time come: still the open tab's
  { status: "snoozed", till: "2020-01-01T00:00:00Z", tab: "open" },
  { status: "declined", tab: "closed" },
  { status: "duplicate", tab: "closed" },
] as const)(
  "sends a work item $status in its project's intake to the $tab tab that selects it, in place of its entry, neither its header nor its detail mounted",
  async (cell) => {
    answer = one(cell.status, "till" in cell ? cell.till : null);
    await visit("/ws/browse/A-1");
    await settled();
    expect(where()).toBe(intakeOf(cell.tab));
    expect(router.state.historyAction).toBe("REPLACE");
    expect(lookups()).toHaveLength(1);
    expect(mounts.seen).toEqual([]);
    await back();
    expect(where()).toBe("/before");
  }
);

it("sends it while the project's intake is off, to the page that says so", async () => {
  await given(false);
  answer = one("pending");
  await visit("/ws/browse/A-1");
  await settled();
  expect(where()).toBe(intakeOf("open"));
  expect(mounts.seen).toEqual([]);
});

it.each([
  { what: "accepted from its project's intake", status: "accepted" },
  { what: "never submitted to an intake", status: null },
] as const)("shows the detail of a work item $what, where it is", async ({ status }) => {
  answer = one(status);
  await visit("/ws/browse/A-1");
  await settled();
  expect(where()).toBe("/ws/browse/A-1");
  expect(mounts.seen.toSorted()).toEqual(["body one", "header one"]);
});

it("judges by the visit's own read, not by an answer an earlier visit left", async () => {
  answer = one("pending");
  await visit("/ws/browse/A-1");
  await settled();
  expect(where()).toBe(intakeOf("open"));
  // accepted since: SWR still holds the earlier visit's answer, in the intake, for the route
  answer = { ...one("accepted"), updated_at: "2026-10-29T00:00:00Z" };
  holdLookup = true;
  await act(async () => {
    await router.navigate("/ws/browse/A-1");
    await vi.advanceTimersByTimeAsync(30);
  });
  await act(async () => until(() => lookups().length === 2, "the visit's own lookup"));
  expect(where()).toBe("/ws/browse/A-1");
  expect(mounts.seen).toEqual([]);
  await act(async () => lookups()[1]!.answer(json(200, answer)));
  await settled();
  expect(where()).toBe("/ws/browse/A-1");
});

it("sends a work item reached by its id, its two entries replaced", async () => {
  answer = one("declined");
  await visit("/ws/projects/A/issues/one");
  await settled();
  expect(where()).toBe(intakeOf("closed"));
  expect(router.state.historyAction).toBe("REPLACE");
  expect(mounts.seen).toEqual([]);
  await back();
  expect(where()).toBe("/before");
});
