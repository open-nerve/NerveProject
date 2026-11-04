/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { observer } from "mobx-react";
import { MemoryRouter } from "react-router";
import type { Asset } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import { contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { useWorkItemCommentOperations } from "../issues/issue-detail/issue-activity/helper";
import { IssueDetailHost } from "../issues/detail-host";
import { IssueAttachments } from "../issues/issue-detail-widgets/attachments/root";
import { CommentCreate } from "./comment-create";
import { CommentCard } from "./card/root";
export async function commentFilesTab() {
  const tab = await pickerDOM(await contentAssetTab());
  const control = { core: (call: Call) => call.answer(json(200, tab.issue)), comments: () => [tab.comment] };
  const CommentPanel = observer(function CommentPanel() {
    const issue = tab.app.issue.issues.getIssueDetailById(tab.issue.id);
    const operations = useWorkItemCommentOperations("ws", "A", tab.issue.id);
    const comment = tab.app.issue.issueDetail.comment.getCommentById(tab.issue.id, tab.comment.id);
    if (!issue)
      return (
        <button onClick={() => void tab.app.issue.issues.refreshAuthoritative(tab.issue.id).catch(() => undefined)}>
          read owner
        </button>
      );
    return (
      <>
        <CommentCreate workspaceSlug="ws" projectId="A" entityId={issue.id} activityOperations={operations} />
        <CommentCard
          workspaceSlug="ws"
          projectId="A"
          comment={comment}
          activityOperations={operations}
          ends={undefined}
          showCopyLinkOption={false}
        />
        <IssueAttachments issueId={issue.id} disabled={false} />
      </>
    );
  });
  tab.nerve.respond = (call) => {
    if (call.path === `/api/v0/issues/${tab.issue.id}` && call.method === "GET") control.core(call);
    else if (call.method === "GET" && call.path === `/api/v0/issues/${tab.issue.id}/comments`)
      call.answer(json(200, { data: control.comments(), next_cursor: null }));
    else if (call.method === "GET" && call.path.startsWith("/api/v0/assets/")) {
      const asset = tab.assets.get(call.path.split("/").at(-1)!);
      call.answer(asset ? json(200, asset) : problem(404, "asset.not_found"));
    } else if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
    else if (call.path.endsWith("/complete")) {
      const id = call.path.split("/").at(-2)!;
      const asset: Asset = {
        ...tab.assets.get(id)!,
        state: "uploaded",
        is_bound: tab.assets.get(id)?.scope.kind !== "new_comment",
        width: null,
        height: null,
      };
      tab.assets.set(id, asset);
      call.answer(json(200, asset));
    } else if (call.path === "/api/v0/assets") tab.answer(call);
  };
  const render = (copies = 1) =>
    act(async () =>
      tab.renderer.render(
        <StoreContext.Provider value={tab.app}>
          <MemoryRouter>
            {Array.from({ length: copies }, (_, index) => (
              <div key={index} data-comment-host={index}>
                <IssueDetailHost workspaceSlug="ws" projectId="A" issueId={tab.issue.id}>
                  <CommentPanel />
                </IssueDetailHost>
              </div>
            ))}
          </MemoryRouter>
        </StoreContext.Provider>
      )
    );
  const text = () => document.querySelector<HTMLTextAreaElement>(`textarea[id^="add_comment_${tab.issue.id}"]`);
  const fill = (value: string, input = text()!) =>
    act(async () => {
      Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  const enter = (input = text()!) =>
    act(async () =>
      input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
    );
  const posts = () => tab.nerve.to(`/api/v0/issues/${tab.issue.id}/comments`).filter((call) => call.method === "POST");
  const upload = async () => {
    await tab.select([new File(["body"], "comment.txt", { type: "text/plain" })]);
    await tab.click("content_files.upload");
    await act(async () => until(() => tab.puts.length === 1, "comment raw PUT"));
    await act(async () => tab.puts[0]!.answer(new Response(null, { status: 200 })));
    await waitForPickerDOM(
      () => !!document.querySelector('button[aria-label="content_files.remove comment.txt"]'),
      "selected attachment"
    );
  };
  return { ...tab, control, render, text, fill, enter, posts, upload };
}
