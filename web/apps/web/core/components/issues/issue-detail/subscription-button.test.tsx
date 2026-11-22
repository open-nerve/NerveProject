// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, Project, ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueOf } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { loadProject, loadProjects } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { seedAccount } from "@/store/user/fake-account";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { IssueDetailHost } from "../detail-host";
import { IssueView } from "../peek-overview/view";
import { IssueDetailQuickActions } from "./issue-detail-quick-actions";
import { IssueSubscriptionButton } from "./subscription-button";
import { useIssueOperations } from "./use-issue-operations";

// Subscribe and unsubscribe in a work item's detail header and its peek's (P4 spec 2.6, R13, R14): shown to whoever
// has a role in its project, a guest too, while its detail is held and neither it nor its project is archived; saying
// which by is_subscribed as nerve last read it; out of use and saying Loading while the change is out, never showing it
// before nerve's read of the work item; a toast says nerve made it, or why not, only in the session it was sent in;
// named by its text at every width. The stores and HTTP are real; the peek's content beside its header is left out.

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, session: { loginId: "one" } }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: bound.session }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
  useNavigate: () => () => {},
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("./issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("../issue-detail-widgets/issue-detail-widget-collapsibles", () => ({
  IssueDetailWidgetCollapsibles: () => null,
}));
vi.mock("../peek-overview/issue-detail", () => ({ PeekOverviewIssueDetails: () => null }));
vi.mock("../peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  bound.session.loginId = "one";
  toasts.length = 0;
  await seedAccount(tab.nerve, tab.app.user, accountOf("actor"));
});
afterEach(async () => tab.close());

const detail = (fields: Partial<IssueDetail> = {}): IssueDetail => ({ ...issueDetailOf("x"), ...fields });
const ARCHIVED = "2026-10-29T00:00:00Z";
/**
 * The caller's roles in ws and in A, A as fields say, and the work item x as held. An archived A is read alone, as a
 * page reads a project its list of projects leaves out (useProjectRead).
 */
async function held(
  workspace: WorkspaceRole,
  role: ProjectRole | null,
  issue: Issue | IssueDetail = detail(),
  fields: Partial<Project> = {}
) {
  const project = { ...tab.project, member_role: role, ...fields };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: workspace }]);
  const archived = project.archived_at !== null;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, archived ? [] : [project]);
  if (archived) await loadProject(tab.nerve, tab.app.projectRoot.project, project);
  tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
}
const render = () =>
  tab.render(
    <MemoryRouter>
      <IssueSubscriptionButton workspaceSlug="ws" projectId="A" issueId="x" />
    </MemoryRouter>
  );
const buttonSaying = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((button) => button.textContent === text);
const subscribeButtons = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((button) =>
    ["common.actions.subscribe", "common.actions.unsubscribe", "common.loading"].includes(button.textContent ?? "")
  );
/** The request to path nerve receives after the `before` ones it received so far. */
async function sent(path: string, before = 0): Promise<Call> {
  await act(async () => until(() => tab.nerve.to(path).length > before, path));
  return tab.nerve.to(path)[before]!;
}
const click = (button: HTMLButtonElement | undefined) => act(async () => button?.click());
const settled = () => act(async () => vi.advanceTimersByTimeAsync(50));

describe("IssueSubscriptionButton", () => {
  it.each([
    ["an admin", 20, 20, detail(), {}, true],
    ["a member", 15, 15, detail(), {}, true],
    ["a guest who sees every work item", 5, 5, detail(), { guest_view_all_features: true }, true],
    ["a guest who sees his own", 5, 5, detail({ created_by_id: "u-actor" }), { guest_view_all_features: false }, true],
    ["one with no role in the project", 15, null, detail(), {}, false],
    ["a member holding a list row, not the detail", 15, 15, issueOf("x"), {}, false],
    ["a member, the work item archived", 15, 15, detail({ archived_at: ARCHIVED }), {}, false],
    ["a member, the project archived", 15, 15, detail(), { archived_at: ARCHIVED }, false],
  ] as const)("shows to %s", async (_, workspace, role, issue, fields, shown) => {
    await held(workspace, role, issue, fields);
    await render();
    expect(subscribeButtons()).toHaveLength(shown ? 1 : 0);
  });

  it("says Subscribe or Unsubscribe as is_subscribed says, its name its text at every width", async () => {
    await held(5, 5);
    await render();
    const [subscribe] = buttonSaying("common.actions.subscribe");
    expect(subscribe).toBeDefined();
    for (const part of [subscribe!, ...subscribe!.querySelectorAll("*")])
      expect(
        [...part.classList].filter((name) => name === "hidden" || name.endsWith(":hidden") || name === "sr-only")
      ).toEqual([]);
    await act(async () =>
      tab.app.issue.issues.acceptRead(detail({ is_subscribed: true }), tab.app.issue.captureRead())
    );
    expect(buttonSaying("common.actions.unsubscribe")).toHaveLength(1);
  });

  it("is out of use and says Loading while nerve subscribes him, then says what nerve read again", async () => {
    await held(5, 5);
    await render();
    await click(buttonSaying("common.actions.subscribe")[0]);
    const post = await sent("/api/v0/issues/x/subscribe");
    const [loading] = buttonSaying("common.loading");
    expect(loading?.disabled).toBe(true);
    await click(loading);
    post.answer(noContent());
    (await sent("/api/v0/issues/x")).answer(json(200, detail({ is_subscribed: true })));
    await settled();
    expect(tab.nerve.to("/api/v0/issues/x/subscribe")).toHaveLength(1);
    const [unsubscribe] = buttonSaying("common.actions.unsubscribe");
    expect(unsubscribe?.disabled).toBe(false);
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "issue.subscription.actions.subscribed" },
    ]);
    await click(unsubscribe);
    (await sent("/api/v0/issues/x/unsubscribe")).answer(noContent());
    (await sent("/api/v0/issues/x", 1)).answer(json(200, detail({ is_subscribed: false })));
    await settled();
    expect(buttonSaying("common.actions.subscribe")).toHaveLength(1);
    expect(toasts.at(-1)).toEqual({
      type: "success",
      title: "toast.success",
      message: "issue.subscription.actions.unsubscribed",
    });
  });

  it("says why nerve refused, never that it made it", async () => {
    await held(15, 15);
    await render();
    await click(buttonSaying("common.actions.subscribe")[0]);
    (await sent("/api/v0/issues/x/subscribe")).answer(problem(403, "forbidden"));
    (await sent("/api/v0/issues/x")).answer(json(200, detail()));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
    expect(buttonSaying("common.actions.subscribe")).toHaveLength(1);
  });

  it.each([
    ["made it", () => noContent()],
    ["refused", () => problem(503, "server_busy")],
  ])("says nothing once the tab has moved to another account before nerve %s (followInSession)", async (_, reply) => {
    await held(15, 15);
    await render();
    await click(buttonSaying("common.actions.subscribe")[0]);
    const post = await sent("/api/v0/issues/x/subscribe");
    bound.session.loginId = "two";
    post.answer(reply());
    (await sent("/api/v0/issues/x")).answer(json(200, detail()));
    await settled();
    expect(toasts).toEqual([]);
  });
});

/** The peek of x, embedded (the notifications page's) or not, its header with the button. */
function Peek({ embed }: { embed: boolean }) {
  const operations = useIssueOperations("ws", "A", "x");
  return (
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="x">
      <IssueView
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        is_archived={false}
        issueOperations={operations}
        embedIssue={embed}
      />
    </IssueDetailHost>
  );
}

describe("the headers that show IssueSubscriptionButton", () => {
  it("the detail's shows it once, before its copy link", async () => {
    await held(15, 15);
    await tab.render(
      <MemoryRouter>
        <IssueDetailQuickActions workspaceSlug="ws" projectId="A" issueId="x" />
      </MemoryRouter>
    );
    const buttons = [...document.querySelectorAll<HTMLButtonElement>("button")];
    const subscribe = buttons.findIndex((button) => button.textContent === "common.actions.subscribe");
    const copy = buttons.findIndex((button) => button.getAttribute("aria-label") === "common.actions.copy_link");
    expect(subscribeButtons()).toHaveLength(1);
    expect(subscribe).toBeGreaterThan(-1);
    expect(subscribe).toBeLessThan(copy);
  });

  it.each([false, true])("the peek's shows it once (embedded: %s)", async (embed) => {
    await held(15, 15);
    tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "x" });
    await tab.render(
      <MemoryRouter>
        <Peek embed={embed} />
      </MemoryRouter>
    );
    expect(subscribeButtons()).toHaveLength(1);
  });
});
