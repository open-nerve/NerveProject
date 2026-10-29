// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig, type Cache } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { IssueParentSiblings } from "./issue-detail/parent/siblings";
import { IssuePeekOverview } from "./peek-overview/root";
import type { IssueView } from "./peek-overview/view";
import { DetailResourceError } from "./issue-detail-widgets/resource-error";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: tab.app.issue.projectIssues }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => undefined }));
vi.mock("./issue-detail/parent/sibling-item", () => ({
  IssueParentSiblingItem: ({ issueId }: { issueId: string }) => <span>{issueId}</span>,
}));
// Keep the request-owning component, actual SWR and native stores. Its large presentation tree is independent.
vi.mock("./peek-overview/view", () => ({
  IssueView: (props: ComponentProps<typeof IssueView>) => (
    <DetailResourceError {...props} error={props.error} retry={props.retry!} />
  ),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let cache: Cache<unknown>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
  cache = new Map();
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const render = (view: ReactNode) =>
  act(async () =>
    renderer.render(<SWRConfig value={{ provider: () => cache, shouldRetryOnError: false }}>{view}</SWRConfig>)
  );
const siblings = () => (
  <IssueParentSiblings workspaceSlug="ws" currentIssue={issueOf("child")} parentIssue={issueOf("parent")} />
);
const variants = [
  { name: "siblings", path: "/api/v0/issues/parent/sub-issues", view: siblings },
  {
    name: "peek",
    path: "/api/v0/issues/x",
    view: () => {
      tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "x" });
      return <IssuePeekOverview />;
    },
  },
];
it.each(variants)("$name exposes a current refusal and retries the original read", async ({ path, view }) => {
  await render(view());
  await act(async () => {
    await until(() => tab.nerve.to(path).length === 1, "initial read");
    tab.nerve.to(path)[0]!.answer(problem(403, "forbidden"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.forbidden");
  expect(host.textContent).not.toContain("No sibling work items");
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await until(() => tab.nerve.to(path).length === 2, "retry");
  });
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it("siblings re-reads after permission ABA within SWR's deduping interval", async () => {
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/issues/parent") call.answer(json(200, issueDetailOf("parent")));
    else if (call.path.endsWith("/sub-issues") || call.path.endsWith("/states")) call.answer(json(200, { data: [] }));
  };
  await render(siblings());
  await act(async () => {
    await until(() => tab.nerve.to("/api/v0/issues/parent").length === 1, "complete parent read");
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(host.textContent).toContain("No sibling work items");
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    await vi.advanceTimersByTimeAsync(20);
  });
  // Guest remains a legitimate reader: both 20→5 and 5→20 must start a read under their new proof.
  expect(tab.nerve.to("/api/v0/issues/parent/sub-issues")).toHaveLength(3);
});

it("requests a readable archived peek for a legitimate guest", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  tab.nerve.calls.length = 0;
  tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "x", isArchived: true });
  await render(<IssuePeekOverview />);
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual([["GET", "/api/v0/issues/x"]]);
});
