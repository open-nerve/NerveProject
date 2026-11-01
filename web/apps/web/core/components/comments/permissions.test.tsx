// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { runInAction } from "mobx";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { accountOf, userOf } from "@/store/member/fake-users";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { issueCommentOf, issueDetailOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { useWorkItemCommentOperations } from "../issues/issue-detail/issue-activity/helper";
import { CommentCard } from "./card/root";
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({ t: (key: string, args?: { name: string }) => (args ? `${key}: ${args.name}` : key) }),
}));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => binding.tab!.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => userOf("Ann") }) }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => binding.tab!.app.user,
  useUserPermissions: () => binding.tab!.app.user.permission,
}));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
// The actual menu, permission store, comment commands and HTTP client remain in this consumer test.
vi.mock("@/components/editor/lite-text", () => ({ LiteTextEditor: () => <div>Comment body</div> }));
let root: Root, host: HTMLDivElement;
const comment = issueCommentOf("comment");
function Card({ disabled = false }: { disabled?: boolean }) {
  const operations = useWorkItemCommentOperations("ws", "A", "x");
  return (
    <CommentCard
      workspaceSlug="ws"
      projectId="A"
      comment={comment}
      activityOperations={operations}
      ends={undefined}
      showCopyLinkOption
      disabled={disabled}
    />
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  binding.tab = await issueStoreTab();
  vi.spyOn(binding.tab.app.issue.issueDetail.comment, "refreshAfterChange").mockResolvedValue();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  binding.tab!.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function setup(projectRole: ProjectRole, workspaceRole: WorkspaceRole, author = false) {
  const tab = binding.tab!;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: workspaceRole }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: projectRole },
  ]);
  runInAction(() => {
    tab.app.user.data = { ...accountOf(author ? "actor" : "reader"), id: author ? comment.actor_id : "reader" };
  });
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  read.call.answer(json(200, { data: [comment], next_cursor: null }));
  await read.promise;
  tab.nerve.calls.length = 0;
}
const render = (disabled = false) =>
  act(async () =>
    root.render(
      <MemoryRouter>
        <Card disabled={disabled} />
      </MemoryRouter>
    )
  );
const items = () => [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')];
async function open() {
  const trigger = host.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]')!;
  await act(async () => {
    trigger.focus();
    trigger.click();
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  return trigger;
}
it.each([
  { project: 20, workspace: 15 },
  { project: 5, workspace: 20 },
] as const)(
  "adminDeletesButCannotEditOthersComment with project $project and workspace $workspace",
  async ({ project, workspace }) => {
    await setup(project, workspace);
    await render();
    const trigger = await open();
    expect(items().map((item) => item.textContent)).toEqual(["common.actions.copy_link", "common.actions.delete"]);
    expect(trigger.getAttribute("aria-label")).toBe("aria_labels.comment_options: Ann");
    expect(trigger.querySelector("button")).toBeNull();
    const remove = items().find((item) => item.textContent === "common.actions.delete")!;
    await act(async () => remove.click());
    await act(async () =>
      until(() => binding.tab!.nerve.to("/api/v0/issue-comments/comment").length === 1, "admin delete")
    );
    const call = binding.tab!.nerve.to("/api/v0/issue-comments/comment")[0]!;
    expect(call.method).toBe("DELETE");
    expect(trigger.disabled).toBe(true);
    await act(async () => trigger.click());
    expect(items()).toHaveLength(0);
    expect(binding.tab!.nerve.calls).toHaveLength(1);
    await act(async () => call.answer(noContent()));
    expect(binding.tab!.app.issue.issueDetail.comment.getCommentById("x", "comment")).toBeUndefined();
    expect(binding.tab!.nerve.calls).toHaveLength(1);
  }
);
it.each([15, 5] as const)("a non-author with project role %s has only a copy action", async (role) => {
  await setup(role, 15);
  await render();
  await open();
  expect(items().map((item) => item.textContent)).toEqual(["common.actions.copy_link"]);
  expect(binding.tab!.nerve.calls).toHaveLength(0);
});
it("guestCommentsOwnVisibleIssue keeps author-only edit and delete independent of work item edit permission", async () => {
  await setup(5, 5, true);
  await render();
  await open();
  expect(items().map((item) => item.textContent)).toEqual([
    "common.actions.edit",
    "common.actions.copy_link",
    "common.actions.delete",
  ]);
  await act(async () => items()[0]!.click());
  expect(host.querySelector("form")).not.toBeNull();
  expect(binding.tab!.nerve.calls).toHaveLength(0);
});
it("an archived comment has no edit, delete or create reaction trigger", async () => {
  await setup(20, 20, true);
  await render(true);
  expect(host.querySelector("button")).toBeNull();
  expect(binding.tab!.nerve.calls).toHaveLength(0);
});
