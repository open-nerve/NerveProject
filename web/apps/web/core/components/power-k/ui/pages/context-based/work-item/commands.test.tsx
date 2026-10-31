// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext, useMemo } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueDetail, ProjectRole, WorkspaceRole } from "@nerve/api-client";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { usePowerKWorkItemContextBasedCommands } from "./commands";

const route = createContext({ workspaceSlug: "ws", workItem: "A-1" });
const auth = vi.hoisted(() => ({ state: { loginId: "commands-login" } }));
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
let commands: TPowerKCommandConfig[], issue: IssueDetail;
const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");
let clipboard: ReturnType<typeof heldChange<void>>, write: ReturnType<typeof vi.fn<(text: string) => Promise<void>>>;
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
  return (
    <>
      {commands
        .filter((command) => command.isVisible(ctx))
        .map((command) => (
          <button key={command.id} disabled={!command.isEnabled(ctx)}>
            {command.id}
          </button>
        ))}
    </>
  );
});
function RouteView({ workItem, workspaceSlug }: { workItem: string; workspaceSlug: string }) {
  const value = useMemo(() => ({ workspaceSlug, workItem }), [workspaceSlug, workItem]);
  return (
    <route.Provider value={value}>
      <Consumer />
    </route.Provider>
  );
}
const render = (workItem = "A-1", workspaceSlug = "ws") =>
  act(async () => root.render(<RouteView workItem={workItem} workspaceSlug={workspaceSlug} />));
const command = (id: string) => commands.find((item) => item.id === id)!;
const invoke = (item: TPowerKCommandConfig, value?: unknown) =>
  item.type === "action" ? item.action(ctx) : item.onSelect(value, ctx);
const start = (id: string, value?: unknown) =>
  act(async () => {
    void invoke(command(id), value);
  });
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const tick = () => act(async () => vi.advanceTimersByTimeAsync(0));
async function setup(
  projectRole: ProjectRole = 15,
  workspaceRole: WorkspaceRole = 15,
  author = true,
  archived = false,
  returned = true
) {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: workspaceRole }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: projectRole, guest_view_all_features: false },
  ]);
  issue = {
    ...issueDetailOf("x"),
    created_by_id: author ? "u-actor" : "other",
    archived_at: archived ? "2026-10-30T00:00:00Z" : null,
  };
  // OwnOnly is enforced by the API. This fixture only supplies an actually returned entity.
  if (returned) tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  await render();
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  auth.state.loginId = "commands-login";
  tab = await issueStoreTab();
  runInAction(() => {
    tab.app.user.data = accountOf("actor");
  });
  clipboard = heldChange<void>();
  write = vi.fn(() => clipboard.sent);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: write } });
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  clipboard.answer();
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  if (clipboardDescriptor) Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
  else Reflect.deleteProperty(navigator, "clipboard");
  vi.restoreAllMocks();
  vi.useRealTimers();
});

it.each([
  ["change_work_item_state", "state-next", { state_id: "state-next" }],
  ["change_work_item_priority", "high", { priority: "high" }],
  ["change_work_item_assignees", "member-next", { assignee_ids: ["member-next"] }],
  ["assign_work_item_to_me", undefined, { assignee_ids: ["u-actor"] }],
  ["add_work_item_labels", { id: "label-next" }, { label_ids: ["label-next"] }],
])("commandWritesOnlyNativeFieldsInCurrentScope: %s waits for confirmation", async (id, value, body) => {
  await setup();
  await start(id, value);
  await act(async () => until(() => patches().length === 1, "native command"));
  expect(patches()[0]).toMatchObject({ path: "/api/v0/issues/x", body });
  expect(patches()[0]!.body).toEqual(body);
  expect(tab.app.issue.issues.getIssueById("x")).toEqual(issue);
  await act(async () => patches()[0]!.answer(json(200, { ...issue, ...body, updated_at: "2026-10-30T01:00:00Z" })));
  expect(tab.app.issue.issues.getIssueById("x")).toMatchObject(body);
});

it.each([
  ["change_work_item_assignees", "first", "second", "assignee_ids"],
  ["add_work_item_labels", { id: "first" }, { id: "second" }, "label_ids"],
])("%s derives each queued toggle from the preceding confirmed value", async (id, first, second, field) => {
  await setup();
  await start(id, first);
  await act(async () => until(() => patches().length === 1, "first toggle"));
  await start(id, second);
  expect(patches()).toHaveLength(1);
  await act(async () =>
    patches()[0]!.answer(json(200, { ...issue, [field]: ["first"], updated_at: "2026-10-30T01:00:00Z" }))
  );
  await act(async () => until(() => patches().length === 2, "queued toggle"));
  expect(patches()[1]!.body).toEqual({ [field]: ["first", "second"] });
  await act(async () =>
    patches()[1]!.answer(json(200, { ...issue, [field]: ["first", "second"], updated_at: "2026-10-30T02:00:00Z" }))
  );
  expect(tab.app.issue.issues.getIssueById("x")).toMatchObject({ [field]: ["first", "second"] });
});

it.each([
  [15, 15, true, false, true, true, true],
  [15, 15, false, false, true, true, false],
  [20, 15, false, false, true, true, true],
  [5, 5, true, false, true, false, false],
  [5, 5, false, false, false, false, false],
  [5, 20, true, false, true, true, true],
  [5, 20, false, false, false, false, false],
  [15, 15, true, true, true, false, true],
] as const)(
  "ownOnlyAdminAndArchiveUseReturnedGrant: project=%s workspace=%s author=%s archived=%s",
  async (pr, wr, author, archived, visible, edit, remove) => {
    await setup(pr, wr, author, archived, visible);
    expect(command("copy_work_item_title").isVisible(ctx)).toBe(visible);
    expect(command("change_work_item_state").isEnabled(ctx)).toBe(edit);
    expect(command("delete_work_item").isEnabled(ctx)).toBe(remove);
    await start("delete_work_item");
    expect(tab.app.commandPalette.isDeleteIssueModalOpen).toBe(remove);
    expect(tab.nerve.calls).toHaveLength(0);
    if (!edit) {
      await start("change_work_item_state", "forbidden");
      await tick();
      expect(patches()).toHaveLength(0);
    }
  }
);

it("oldContextCannotMutateNewIssue after A to B to A, or copy its title from a retained action", async () => {
  await setup();
  const oldState = command("change_work_item_state"),
    oldCopy = command("copy_work_item_title"),
    oldDelete = command("delete_work_item");
  const other = { ...issueDetailOf("y"), sequence_id: 2 };
  await act(async () => tab.app.issue.issues.acceptRead(other, tab.app.issue.captureRead()));
  await render("A-2");
  await render();
  await act(async () => {
    void invoke(oldState, "stale");
    void invoke(oldCopy);
    void invoke(oldDelete);
  });
  await tick();
  expect(patches()).toHaveLength(0);
  expect(write).not.toHaveBeenCalled();
  expect(tab.app.commandPalette.isDeleteIssueModalOpen).toBe(false);
});
it("an old workspace cannot use a same-number mapping from another workspace before router effects run", async () => {
  await setup();
  await render("A-1", "missing-workspace");
  expect(command("copy_work_item_title").isVisible(ctx)).toBe(false);
  await start("change_work_item_state", "stale");
  await tick();
  expect(patches()).toHaveLength(0);
});
it.each([403, 409, 422])(
  "current native refusal reports its code without projecting requested state (%s)",
  async (status) => {
    await setup();
    await start("change_work_item_state", "new");
    await act(async () => until(() => patches().length === 1, "refused command"));
    const code = status === 409 ? "issue.conflict" : status === 422 ? "validation_failed" : "forbidden";
    await act(async () => patches()[0]!.answer(problem(status, code)));
    expect(tab.app.issue.issues.getIssueById("x")?.state_id).toBe("state");
    expect(toasts).toHaveLength(1);
    expect(toasts[0]!.message).not.toBe("Work item could not be updated. Please try again.");
    if (status === 403) expect(toasts[0]!.message).toBe("errors.forbidden");
  }
);
it("a retired command refusal cannot notify another route or a later session", async () => {
  await setup();
  await start("change_work_item_state", "new");
  await act(async () => until(() => patches().length === 1, "old command"));
  await render("A-2");
  auth.state.loginId = "later-session";
  await act(async () => patches()[0]!.answer(problem(403, "forbidden")));
  expect(toasts).toHaveLength(0);
});
it.each([
  ["copy_work_item_id", "A-1"],
  ["copy_work_item_title", "x"],
  ["copy_work_item_url", `${window.location.origin}/ws/browse/A-1`],
])("%s waits for the clipboard and has one hook-owned refusal", async (id, expected) => {
  await setup();
  await start(id);
  expect(write).toHaveBeenCalledExactlyOnceWith(expected);
  expect(toasts).toHaveLength(0);
  await act(async () => clipboard.refuse(new DOMException("denied", "NotAllowedError")));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});

it("current commands are visible on the first committed render without another store notification", async () => {
  await setup();
  expect([...host.querySelectorAll("button")].map((button) => button.textContent)).toContain("change_work_item_state");
});
it.each(["route", "archived"])("a queued toggle is dropped when its %s retires before execution", async (reason) => {
  await setup();
  await start("change_work_item_assignees", "first");
  await act(async () => until(() => patches().length === 1, "first pending toggle"));
  await start("change_work_item_assignees", "second");
  if (reason === "route") await render("A-2");
  await act(async () =>
    patches()[0]!.answer(
      json(200, {
        ...issue,
        assignee_ids: ["first"],
        updated_at: "2026-10-30T01:00:00Z",
        archived_at: reason === "archived" ? "2026-10-30T01:00:00Z" : null,
      })
    )
  );
  await tick();
  expect(patches()).toHaveLength(1);
  expect(toasts).toHaveLength(0);
});
