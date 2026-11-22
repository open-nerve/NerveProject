// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext, useMemo } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, Project, ProjectRole, WorkspaceRole } from "@nerve/api-client";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import type { Call } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { loadProject, loadProjects } from "@/store/project/fake-projects";
import { seedAccount } from "@/store/user/fake-account";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { usePowerKWorkItemContextBasedCommands } from "./commands";

// The work item's command that subscribes the caller to it or unsubscribes him (P4 spec 2.6, R13, R14), by Shift+S:
// offered where the detail's button is (a role in its project, a guest's too; its detail held; it and its project not
// archived), titled by is_subscribed; out of use while the change is pending; nerve's answer said as the button says
// it, only in the session it was sent in; a command kept from another work item sends nothing. The stores and HTTP
// are real (issueStoreTab).

const route = createContext({ workspaceSlug: "ws", workItem: "A-1" });
const auth = vi.hoisted(() => ({ state: { loginId: "subscription-login" } }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: auth, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => useContext(route) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => tab.app.user }));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => tab.app.commandPalette }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>, host: HTMLDivElement, root: Root;
let commands: TPowerKCommandConfig[];
const ctx: TPowerKContext = {
  params: { workspaceSlug: "ws", workItem: "A-1", projectId: "A" },
  activeCommand: null,
  activeContext: "work-item",
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
};
const Consumer = observer(function Consumer() {
  commands = usePowerKWorkItemContextBasedCommands();
  return null;
});
function RouteView({ workItem }: { workItem: string }) {
  const value = useMemo(() => ({ workspaceSlug: "ws", workItem }), [workItem]);
  return (
    <route.Provider value={value}>
      <Consumer />
    </route.Provider>
  );
}
const render = (workItem = "A-1") => act(async () => root.render(<RouteView workItem={workItem} />));
const command = () => commands.find((item) => item.id === "subscribe_work_item")!;
const run = (item: TPowerKCommandConfig) =>
  act(async () => {
    if (item.type === "action") void item.action(ctx);
  });
const detail = (fields: Partial<IssueDetail> = {}): IssueDetail => ({ ...issueDetailOf("x"), ...fields });
const ARCHIVED = "2026-10-29T00:00:00Z";
/**
 * The caller's roles in ws and in A, A as fields say, the work item A-1 held as issue, and the palette on it. An
 * archived A is read alone, as a page reads a project its list of projects leaves out.
 */
async function setup(
  workspace: WorkspaceRole = 5,
  role: ProjectRole = 5,
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
  await render();
}
async function sent(path: string, before = 0): Promise<Call> {
  await act(async () => until(() => tab.nerve.to(path).length > before, path));
  return tab.nerve.to(path)[before]!;
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  auth.state.loginId = "subscription-login";
  tab = await issueStoreTab();
  await seedAccount(tab.nerve, tab.app.user, accountOf("actor"));
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

it("is titled by is_subscribed, run by Shift+S, an action that closes the palette, its keys no other command's", async () => {
  await setup();
  expect(command()).toMatchObject({
    i18n_title: "power_k.contextual_actions.work_item.subscribe",
    type: "action",
    modifierShortcut: "shift+s",
    closeOnSelect: true,
  });
  expect(command().shortcut).toBeUndefined();
  const keys = commands.flatMap((item) => [item.shortcut, item.keySequence, item.modifierShortcut]).filter(Boolean);
  expect(new Set(keys).size).toBe(keys.length);
  await act(async () => tab.app.issue.issues.acceptRead(detail({ is_subscribed: true }), tab.app.issue.captureRead()));
  expect(command().i18n_title).toBe("power_k.contextual_actions.work_item.unsubscribe");
});

it.each([
  ["a guest", 5, 5, detail(), {}, true],
  ["a member, the work item archived", 15, 15, detail({ archived_at: ARCHIVED }), {}, false],
  ["a member, the project archived", 15, 15, detail(), { archived_at: ARCHIVED }, false],
  ["a member holding a list row, not the detail", 15, 15, issueOf("x"), {}, false],
] as const)("is offered to %s as the button is", async (_, workspace, role, issue, fields, offered) => {
  await setup(workspace, role, issue, fields);
  expect(command().isVisible(ctx)).toBe(offered);
  expect(command().isEnabled(ctx)).toBe(offered);
});

it("subscribes him as the button does, out of use until nerve has read the work item again, then says so", async () => {
  await setup();
  await run(command());
  const post = await sent("/api/v0/issues/x/subscribe");
  expect(command().isEnabled(ctx)).toBe(false);
  post.answer(noContent());
  const read = await sent("/api/v0/issues/x");
  expect(command().isEnabled(ctx)).toBe(false);
  read.answer(json(200, detail({ is_subscribed: true })));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(command().isEnabled(ctx)).toBe(true);
  expect(command().i18n_title).toBe("power_k.contextual_actions.work_item.unsubscribe");
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "issue.subscription.actions.subscribed" },
  ]);
  await run(command());
  (await sent("/api/v0/issues/x/unsubscribe")).answer(noContent());
  (await sent("/api/v0/issues/x", 1)).answer(json(200, detail()));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(toasts.at(-1)?.message).toBe("issue.subscription.actions.unsubscribed");
});

it("says why nerve refused, never that it made it", async () => {
  await setup();
  await run(command());
  (await sent("/api/v0/issues/x/subscribe")).answer(problem(403, "forbidden"));
  (await sent("/api/v0/issues/x")).answer(json(200, detail()));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
});

it.each([
  ["made it", () => noContent()],
  ["refused", () => problem(503, "server_busy")],
])("says nothing once the tab has moved to another account before nerve %s (followInSession)", async (_, reply) => {
  await setup();
  await run(command());
  const post = await sent("/api/v0/issues/x/subscribe");
  auth.state.loginId = "later-session";
  post.answer(reply());
  (await sent("/api/v0/issues/x")).answer(json(200, detail()));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(toasts).toEqual([]);
});

it("sends nothing from a command kept from before the palette moved to another work item and back", async () => {
  await setup();
  const kept = command();
  const other = { ...issueDetailOf("y"), sequence_id: 2 };
  await act(async () => tab.app.issue.issues.acceptRead(other, tab.app.issue.captureRead()));
  await render("A-2");
  await render();
  await run(kept);
  await act(async () => vi.advanceTimersByTimeAsync(1_000));
  expect(tab.nerve.calls).toEqual([]);
});
