// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, useImperativeHandle, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
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
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => binding.tab!.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => binding.tab!.app.user }));
vi.mock("@/hooks/store/use-editor-asset", () => ({
  useEditorAsset: () => ({ uploadEditorAsset: vi.fn(), duplicateEditorAsset: vi.fn() }),
}));
vi.mock("@/components/editor/lite-text", () => ({
  LiteTextEditor: ({
    initialValue,
    onChange,
    isSubmitting,
    ref,
  }: {
    initialValue: string;
    onChange: (json: unknown, html: string) => void;
    isSubmitting?: boolean;
    ref?: React.Ref<{
      clearEditor: () => void;
      setEditorValue: (html: string) => void;
      isEditorReadyToDiscard: () => boolean;
    }>;
  }) => {
    const [text, setText] = useState(initialValue);
    useImperativeHandle(ref, () => ({
      clearEditor: () => setText(""),
      setEditorValue: setText,
      isEditorReadyToDiscard: () => true,
    }));
    return (
      <textarea
        data-editor
        value={text}
        disabled={isSubmitting}
        onChange={(event) => {
          setText(event.target.value);
          onChange(undefined, event.target.value);
        }}
      />
    );
  },
}));
let root: Root;
let container: HTMLElement;
const finished = vi.fn();
const edited = vi.fn();
const comment = issueCommentOf("c-z");
function Host({ id = "x", edit = false }: { id?: string; edit?: boolean }) {
  const operations = useWorkItemCommentOperations("ws", "A", id);
  return edit ? (
    <CommentCardEditForm
      activityOperations={operations}
      comment={comment}
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
  binding.tab = await issueStoreTab();
  runInAction(() => {
    binding.tab!.app.user.data = accountOf("actor");
  });
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
const render = (id = "x", edit = false) => act(async () => root.render(<Host id={id} edit={edit} />));
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
it("an old host's successful create does not clear new input or follow up in the new issue", async () => {
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
        entityId="x"
        comment={comment}
        activityOperations={operations}
        ends={undefined}
        showCopyLinkOption={false}
        enableReplies={false}
        disabled
      />
    );
  }
  await act(async () =>
    root.render(
      <MemoryRouter>
        <Card />
      </MemoryRouter>
    )
  );
  expect(container.textContent).toContain("common.deactivated_user");
  expect(container.textContent).not.toContain("edited");
  expect(container.querySelector("textarea")?.value).toBe(comment.comment_html);
});
