/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef, useState } from "react";
import { observer } from "mobx-react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { expect, vi } from "vitest";
import type {
  Asset,
  IssueComment,
  IssueCommentCreate,
  IssueCommentUpdate,
  IssueDetail,
  IssueUpdate,
  NewAsset,
} from "@nerve/api-client";
import type { EditorImages, EditorRefApi } from "@nerve/editor";
import { CommentCreate } from "@/components/comments/comment-create";
import { CommentCardDisplay } from "@/components/comments/card/display";
import { useWorkItemCommentOperations } from "./issue-detail/issue-activity/helper";
import { IssueDetailHost } from "./detail-host";
import { DescriptionVersionsModal } from "@/components/core/description-versions/modal";
import { useIssueImages } from "@/hooks/editor/use-issue-images";
import { StoreContext } from "@/lib/store-context";
import { contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { IssueDetailRoot } from "./issue-detail/root";
import { IssuePeekOverview } from "./peek-overview/root";
export async function bodyImagesTab() {
  const tab = await pickerDOM(await contentAssetTab());
  let issue: IssueDetail = { ...tab.issue, description_html: '<p data-id="base">Original body</p>' };
  let comments: IssueComment[] = [{ ...tab.comment, comment_html: '<p data-id="comment-base">Original comment</p>' }];
  tab.nerve.respond = undefined;
  await tab.readIssue(issue);
  await tab.readComments(comments);
  tab.nerve.calls.length = 0;
  const control = {
    historyHTML: "",
    historyImages: undefined as EditorImages | undefined,
    core: (call: Call) => call.answer(json(200, issue)),
    patch: (call: Call) => {
      issue = { ...issue, ...(call.body as IssueUpdate), updated_at: new Date().toISOString() } as IssueDetail;
      call.answer(json(200, issue));
    },
    asset: (call: Call, row: Asset | undefined) => call.answer(row ? json(200, row) : problem(404, "asset.not_found")),
    comment: (call: Call) => {
      const html = (call.body as IssueCommentCreate | IssueCommentUpdate).comment_html;
      const row =
        call.method === "POST"
          ? { ...comments[0]!, id: "comment-2", comment_html: html }
          : { ...comments[0]!, comment_html: html };
      comments = call.method === "POST" ? [...comments, row] : [row, ...comments.slice(1)];
      for (const [key, asset] of tab.assets) {
        if (
          asset.scope.kind === "new_comment" &&
          (html.includes(key) || (call.body as IssueCommentCreate).attachment_ids?.includes(key))
        )
          tab.assets.set(key, { ...asset, scope: { kind: "comment", id: row.id }, is_bound: true });
      }
      call.answer(json(call.method === "POST" ? 201 : 200, row));
    },
  };
  let sequence = 0;
  let grants = 0;
  const id = () => `11111111-1111-4111-8111-${String(++sequence).padStart(12, "0")}`;
  tab.nerve.respond = (call) => {
    if (call.path === `/api/v0/issues/${issue.id}`) {
      if (call.method === "GET") control.core(call);
      else if (call.method === "PATCH") control.patch(call);
    } else if (call.path === `/api/v0/issues/${issue.id}/comments` && call.method === "GET") {
      call.answer(json(200, { data: comments, next_cursor: null }));
    } else if (
      (call.path.startsWith(`/api/v0/issues/${issue.id}/comments`) ||
        call.path.startsWith("/api/v0/issue-comments/")) &&
      ["POST", "PATCH"].includes(call.method)
    ) {
      control.comment(call);
    } else if (call.path === "/api/v0/assets" && call.method === "POST") {
      const row: Asset = {
        ...(call.body as NewAsset),
        id: id(),
        state: "pending",
        is_bound: false,
        width: null,
        height: null,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      tab.assets.set(row.id, row);
      call.answer(
        json(201, {
          asset: row,
          upload: {
            method: "PUT",
            url: `/static/${row.id}?upload=1`,
            headers: [{ name: "Content-Type", value: row.content_type }],
            expires_at: new Date(Date.now() + 60_000).toISOString(),
          },
        })
      );
    } else if (call.path.endsWith("/complete")) {
      const currentID = call.path.split("/").at(-2)!;
      const row = {
        ...tab.assets.get(currentID)!,
        state: "uploaded" as const,
        is_bound: tab.assets.get(currentID)!.purpose.endsWith("ATTACHMENT"),
        width: 2,
        height: 2,
      };
      tab.assets.set(currentID, row);
      call.answer(json(200, row));
    } else if (call.path.endsWith("/content")) {
      const currentID = call.path.split("/").at(-2)!;
      call.answer(
        json(200, {
          url: `/static/${currentID}?grant=${++grants}`,
          expires_at: new Date(Date.now() + 60_000).toISOString(),
        })
      );
    } else if (call.path.startsWith("/api/v0/assets/") && call.method === "GET") {
      control.asset(call, tab.assets.get(call.path.split("/").at(-1)!));
    } else if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
  };
  type Kind = "detail" | "peek" | "new-comment" | "edit-comment" | "read-comment" | "history";
  const History = observer(function History() {
    const { images } = useIssueImages("ws", "A", issue.id, true);
    control.historyImages = images;
    return (
      <DescriptionVersionsModal
        activeVersionDescription={control.historyHTML}
        activeVersionDetails={{ id: "version-1", ownerId: tab.comment.actor_id, lastSavedAt: issue.updated_at }}
        images={images}
        handleClose={() => {}}
        handleNavigation={() => {}}
        handleRestore={async () => {}}
        isNextDisabled
        isPrevDisabled
        isRestoreDisabled
        isOpen
        workspaceSlug="ws"
        projectId="A"
      />
    );
  });
  const Comments = observer(function Comments({ kind }: { kind: Kind }) {
    const ops = useWorkItemCommentOperations("ws", "A", issue.id);
    const [editing, setEditing] = useState(kind === "edit-comment");
    const ref = useRef<EditorRefApi>(null);
    const canonical = tab.app.issue.issues.getIssueDetailById(issue.id);
    const comment = tab.app.issue.issueDetail.comment.getCommentById(issue.id, comments[0]!.id);
    if (!canonical || !comment) return null;
    return kind === "new-comment" ? (
      <CommentCreate workspaceSlug="ws" projectId="A" entityId={issue.id} activityOperations={ops} />
    ) : (
      <CommentCardDisplay
        workspaceSlug="ws"
        workspaceId={tab.workspace.id}
        projectId="A"
        comment={comment}
        disabled={false}
        activityOperations={ops}
        isEditing={editing}
        setIsEditing={setEditing}
        readOnlyEditorRef={ref}
      />
    );
  });
  const render = async (kind: Kind = "detail") => {
    if (kind === "peek")
      tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: issue.id });
    await act(async () =>
      tab.renderer.render(
        <StoreContext.Provider value={tab.app}>
          <MemoryRouter>
            <SWRConfig value={{ provider: () => new Map() }}>
              {kind === "detail" ? (
                <IssueDetailRoot workspaceSlug="ws" projectId="A" issueId={issue.id} />
              ) : kind === "peek" ? (
                <IssuePeekOverview hostScope="body-images" />
              ) : kind === "history" ? (
                <History />
              ) : (
                <IssueDetailHost workspaceSlug="ws" projectId="A" issueId={issue.id}>
                  <Comments kind={kind} />
                </IssueDetailHost>
              )}
            </SWRConfig>
          </MemoryRouter>
        </StoreContext.Provider>
      )
    );
    await waitForPickerDOM(
      () => (kind === "read-comment" || kind === "history" ? !!document.querySelector(".tiptap") : !!editor()),
      "real issue editor"
    );
  };
  const patches = () => tab.nerve.to(`/api/v0/issues/${issue.id}`).filter((call) => call.method === "PATCH");
  /** Another writer changed the stored body; the owner reads it like any later canonical update. */
  const setBody = async (html: string) => {
    issue = { ...issue, description_html: html };
    await act(async () => {
      await tab.app.issue.issues.refreshAuthoritative(issue.id);
    });
  };
  return {
    ...tab,
    control,
    render,
    editor,
    paste,
    debounce,
    patches,
    setBody,
    titleField,
    typeTitle,
    removeImage,
    moveImageBefore,
    canonical: () => issue,
  };
}
const editor = () => document.querySelector<HTMLElement>('.tiptap[contenteditable="true"]');
const paste = async (html?: string, files: File[] = []) => {
  const field = editor();
  expect(field).not.toBeNull();
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: {
      files,
      items: files.map((file) => ({
        kind: "file",
        type: file.type,
        getAsFile: () => file,
        webkitGetAsEntry: () => ({ isDirectory: false }),
      })),
      getData: (type: string) => (type === "text/nerve-editor-html" ? (html ?? "") : ""),
    },
  });
  await act(async () => field!.dispatchEvent(event));
  return event;
};
const debounce = () => act(async () => vi.advanceTimersByTimeAsync(1600));
const titleField = () => document.querySelector<HTMLTextAreaElement>('textarea[aria-label="issue.title.label"]');
const typeTitle = async (value: string) => {
  const field = titleField();
  expect(field).not.toBeNull();
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(field, value);
    field!.dispatchEvent(new Event("input", { bubbles: true }));
  });
};
// Same-editor edits a user makes on images: deleting a selected one, or dragging one above another.
const mounted = () => (editor() as (HTMLElement & { editor?: MountedEditor }) | null)?.editor;
const imageNode = (assetID: string) => {
  let found: { node: DocNode; pos: number } | undefined;
  mounted()!.state.doc.descendants((node, pos) => {
    if (node.attrs.src === assetID) found = { node, pos };
  });
  expect(found, `image ${assetID}`).toBeDefined();
  return found!;
};
const removeImage = async (assetID: string) => {
  const view = mounted()!;
  const { node, pos } = imageNode(assetID);
  await act(async () => view.view.dispatch(view.state.tr.delete(pos, pos + node.nodeSize)));
};
const moveImageBefore = async (assetID: string, beforeID: string) => {
  const view = mounted()!;
  const moved = imageNode(assetID);
  const target = imageNode(beforeID);
  expect(target.pos).toBeLessThan(moved.pos);
  await act(async () =>
    view.view.dispatch(view.state.tr.delete(moved.pos, moved.pos + moved.node.nodeSize).insert(target.pos, moved.node))
  );
};
type DocNode = { attrs: Record<string, unknown>; nodeSize: number };
type DocTransaction = {
  delete(from: number, to: number): DocTransaction;
  insert(pos: number, node: DocNode): DocTransaction;
};
/** The structural part of the mounted Tiptap editor these image edits use. */
type MountedEditor = {
  state: { doc: { descendants(visit: (node: DocNode, pos: number) => void): void }; tr: DocTransaction };
  view: { dispatch(tr: DocTransaction): void };
};
