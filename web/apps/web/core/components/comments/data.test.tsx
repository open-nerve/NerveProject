// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { seedAccount } from "@/store/user/fake-account";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { accountOf } from "@/store/member/fake-users";
import { issueCommentOf, issueDetailOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { useWorkItemCommentOperations } from "../issues/issue-detail/issue-activity/helper";
import { CommentCreate } from "./comment-create";
import { CommentCardEditForm } from "./card/edit-form";
import { IssueActivityCommentRoot } from "../issues/issue-detail/issue-activity/activity-comment-root";
import { E_SORT_ORDER } from "@nerve/constants";
import { MemoryRouter } from "react-router";
import { CommentCard } from "./card/root";
const binding = vi.hoisted(() => ({
  tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined,
  ready: true,
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { status: "signed-in", loginId: "one" }, start: async () => {}, subscribe: () => () => {} },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => binding.tab!.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => binding.tab!.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ workspace: binding.tab!.app.memberRoot.workspace, getUserDetails: () => undefined }),
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => binding.tab!.app.user,
  useUserPermissions: () => binding.tab!.app.user.permission,
}));
vi.mock("@/hooks/store/use-editor-asset", () => ({
  useEditorAsset: () => ({ uploadEditorAsset: vi.fn(), duplicateEditorAsset: vi.fn() }),
}));
vi.mock("@/components/editor/lite-text", async () => ({
  LiteTextEditor: (await import("./fake-comment-editor")).commentEditor(() => binding.ready),
}));
let root: Root;
let container: HTMLElement;
const finished = vi.fn();
const edited = vi.fn();
const comment = { ...issueCommentOf("c-z"), actor_id: "u-actor" };
function Host({ id = "x", edit = false }: { id?: string; edit?: boolean }) {
  const operations = useWorkItemCommentOperations("ws", "A", id);
  return edit ? (
    <CommentCardEditForm
      key={id}
      activityOperations={operations}
      comment={id === "x" ? comment : { ...comment, id: "other-comment", issue_id: id }}
      isEditing
      projectId="A"
      workspaceId="id-ws"
      workspaceSlug="ws"
      readOnlyEditorRef={null}
      setIsEditing={edited}
    />
  ) : (
    <CommentCreate
      workspaceSlug="ws"
      projectId="A"
      entityId={id}
      activityOperations={operations}
      onSubmitCallback={finished}
    />
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  binding.ready = true;
  binding.tab = await issueStoreTab();
  vi.spyOn(binding.tab.app.issue.issueDetail.comment, "refreshAfterChange").mockResolvedValue();
  await seedAccount(binding.tab!.nerve, binding.tab!.app.user, accountOf("actor"));
  for (const id of ["x", "other"])
    binding.tab.app.issue.issues.acceptRead(issueDetailOf(id), binding.tab.app.issue.captureRead());
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  toasts.length = 0;
  finished.mockReset();
  edited.mockReset();
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  binding.tab!.app.dispose();
  vi.useRealTimers();
});
const render = (id = "x", edit = false) =>
  act(async () =>
    root.render(
      <StoreContext.Provider value={binding.tab!.app}>
        <Host id={id} edit={edit} />
      </StoreContext.Provider>
    )
  );
const fill = (value: string) =>
  act(async () => {
    const input = container.querySelector("textarea")!;
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
const enter = () =>
  act(async () => {
    container.querySelector("textarea")!.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
  });
it.each(["create", "edit"] as const)(
  "%s waits for confirmation, keeps refused text, and finishes once after retry",
  async (kind) => {
    const tab = binding.tab!;
    if (kind === "edit") {
      const read = await requestedIssue(
        tab.nerve,
        () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"),
        ["GET", "/api/v0/issues/x/comments"]
      );
      read.call.answer(json(200, { data: [comment], next_cursor: null }));
      await read.promise;
    }
    await render("x", kind === "edit");
    await fill("<p>keep my text</p>");
    await enter();
    const path = kind === "create" ? "/api/v0/issues/x/comments" : "/api/v0/issue-comments/c-z";
    const method = kind === "create" ? "POST" : "PATCH";
    await act(async () => until(() => tab.nerve.to(path).some((call) => call.method === method), "comment submission"));
    expect(edited).not.toHaveBeenCalled();
    expect(finished).not.toHaveBeenCalled();
    const sent = tab.nerve.to(path).filter((call) => call.method === method);
    expect(sent).toHaveLength(1);
    await enter();
    await act(async () =>
      container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
    expect(tab.nerve.to(path).filter((call) => call.method === method)).toHaveLength(1);
    expect(container.querySelector("textarea")?.disabled).toBe(true);
    expect(container.querySelector("textarea")?.closest("[inert]")).not.toBeNull();
    await act(async () => sent[0]!.answer(problem(403, "forbidden")));
    expect(container.querySelector("textarea")?.value).toBe("<p>keep my text</p>");
    expect(edited).not.toHaveBeenCalled();
    expect(finished).not.toHaveBeenCalled();
    expect(toasts.at(-1)?.type).toBe("error");
    await enter();
    await act(async () =>
      until(() => tab.nerve.to(path).filter((call) => call.method === method).length === 2, "comment retry")
    );
    await act(async () =>
      tab.nerve
        .to(path)
        .filter((call) => call.method === method)[1]!
        .answer(
          json(kind === "create" ? 201 : 200, {
            ...comment,
            comment_html: "<p>confirmed</p>",
            updated_at: "2026-10-30T01:00:00Z",
          })
        )
    );
    if (kind === "create") {
      expect(container.querySelector("textarea")?.value).toBe("");
      expect(finished).toHaveBeenCalledExactlyOnceWith(comment.id);
    } else expect(edited).toHaveBeenCalledExactlyOnceWith(false);
  }
);
it("refusalKeepsInputAndOldSuccessDoesNotClearNewHost after a confirmed create", async () => {
  const tab = binding.tab!;
  await render();
  await fill("<p>old input</p>");
  await enter();
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/comments").length === 1, "old create"));
  await render("other");
  await fill("<p>new input</p>");
  await act(async () => tab.nerve.to("/api/v0/issues/x/comments")[0]!.answer(json(201, comment)));
  expect(container.querySelector("textarea")?.value).toBe("<p>new input</p>");
  expect(finished).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
it("the activity view exposes a comment read refusal and retries only the collection", async () => {
  const tab = binding.tab!;
  const sent = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  function ActivityPanel() {
    const operations = useWorkItemCommentOperations("ws", "A", "x");
    return (
      <IssueActivityCommentRoot
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        isIntakeIssue={false}
        activityOperations={operations}
        selectedFilters={[]}
        sortOrder={E_SORT_ORDER.ASC}
      />
    );
  }
  await act(async () => root.render(<ActivityPanel />));
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  await act(async () => container.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/comments").length === 2, "comment read retry"));
  await act(async () =>
    tab.nerve.to("/api/v0/issues/x/comments")[1]!.answer(json(200, { data: [], next_cursor: null }))
  );
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(tab.nerve.calls.map((call) => call.method)).toEqual(["GET", "GET"]);
});
it("renders the native comment with nullable edited time and the member-directory fallback", async () => {
  function Card() {
    const operations = useWorkItemCommentOperations("ws", "A", "x");
    return (
      <CommentCard
        workspaceSlug="ws"
        projectId="A"
        comment={comment}
        activityOperations={operations}
        ends={undefined}
        showCopyLinkOption={false}
        disabled
      />
    );
  }
  await act(async () =>
    root.render(
      <StoreContext.Provider value={binding.tab!.app}>
        <MemoryRouter>
          <Card />
        </MemoryRouter>
      </StoreContext.Provider>
    )
  );
  expect(container.textContent).toContain("common.deactivated_user");
  expect(container.textContent).not.toContain("edited");
  expect(container.querySelector("textarea")?.value).toBe(comment.comment_html);
  // The fragment target and the editor each need a unique DOM identity.
  expect(container.querySelectorAll(`[id="${comment.id}"]`)).toHaveLength(1);
  expect(container.querySelectorAll(`[id="comment-${comment.id}"]`)).toHaveLength(1);
  expect(container.querySelector(`[id="comment-${comment.id}"]`)?.contains(container.querySelector("textarea"))).toBe(
    true
  );
});

for (const edit of [false, true]) {
  it(`imeAndEmptyDoNotSend for ${edit ? "edit" : "create"}, including the editor callback and form submit`, async () => {
    if (edit) {
      const read = await requestedIssue(
        binding.tab!.nerve,
        () => binding.tab!.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"),
        ["GET", "/api/v0/issues/x/comments"]
      );
      read.call.answer(json(200, { data: [comment], next_cursor: null }));
      await read.promise;
      binding.tab!.nerve.calls.length = 0;
    }
    await render("x", edit);
    await fill("<p>中文</p>");
    const input = container.querySelector("textarea")!;
    await act(async () => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", isComposing: true, bubbles: true, cancelable: true })
      );
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", keyCode: 229, bubbles: true, cancelable: true })
      );
    });
    expect(binding.tab!.nerve.calls).toHaveLength(0);
    binding.ready = false;
    await enter();
    await act(async () =>
      container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
    expect(binding.tab!.nerve.calls).toHaveLength(0);
    binding.ready = true;
    await fill("<p><br></p>");
    await enter();
    await act(async () =>
      container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
    expect(binding.tab!.nerve.calls).toHaveLength(0);
  });
  it(`a ${edit ? "edit" : "create"} field refusal is visible and described before focus, retaining the input`, async () => {
    if (edit) {
      const read = await requestedIssue(
        binding.tab!.nerve,
        () => binding.tab!.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"),
        ["GET", "/api/v0/issues/x/comments"]
      );
      read.call.answer(json(200, { data: [comment], next_cursor: null }));
      await read.promise;
    }
    await render("x", edit);
    await fill("<p>kept</p>");
    await enter();
    const path = edit ? "/api/v0/issue-comments/c-z" : "/api/v0/issues/x/comments";
    const method = edit ? "PATCH" : "POST";
    await act(async () =>
      until(() => binding.tab!.nerve.to(path).some((call) => call.method === method), "field refusal request")
    );
    await act(async () =>
      binding
        .tab!.nerve.to(path)
        .find((call) => call.method === method)!
        .answer(
          json(422, { status: 422, code: "validation_failed", errors: [{ field: "comment_html", code: "required" }] })
        )
    );
    expect(container.querySelector("textarea")?.value).toBe("<p>kept</p>");
    const invalid = container.querySelector<HTMLElement>('[aria-invalid="true"]');
    expect(invalid).not.toBeNull();
    expect(document.getElementById(invalid!.getAttribute("aria-describedby")!)?.textContent).toBe(
      "errors.field.required"
    );
    expect(document.activeElement).toBe(invalid);
    expect(toasts).toHaveLength(0);
  });
}

it("an old edit confirmation neither closes the replacement form nor changes its input or feedback", async () => {
  const tab = binding.tab!;
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  read.call.answer(json(200, { data: [comment], next_cursor: null }));
  await read.promise;
  await render("x", true);
  await fill("<p>Old edit</p>");
  await enter();
  await act(async () => until(() => tab.nerve.to("/api/v0/issue-comments/c-z").length === 1, "old edit"));
  await render("other", true);
  await fill("<p>Replacement edit</p>");
  await act(async () =>
    tab.nerve
      .to("/api/v0/issue-comments/c-z")[0]!
      .answer(json(200, { ...comment, comment_html: "<p>Old confirmed edit</p>" }))
  );
  expect(container.querySelector("textarea")?.value).toBe("<p>Replacement edit</p>");
  expect(edited).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
