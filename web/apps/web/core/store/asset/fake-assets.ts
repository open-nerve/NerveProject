/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { runInAction } from "mobx";
import { expect, vi } from "vitest";
import type { Asset, Attachment, Draft, IssueComment, IssueDetail, NewAsset, PreparedAsset } from "@nerve/api-client";
import { answered, json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { draftOf, issueCommentOf, issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import type { ContentHost } from "./content-view";

export const imageFile = (name = "picture.png", bytes = 4, type = "image/png") =>
  new File([new Uint8Array(bytes)], name, { type });
export const avatarTarget = { purpose: "USER_AVATAR", scope: { kind: "user" } } as const;

function preparedAsset(id: string, input: NewAsset): PreparedAsset {
  return {
    asset: {
      ...input,
      id,
      state: "pending",
      is_bound: false,
      width: null,
      height: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
    upload: {
      method: "PUT",
      url: `/static/${id}?token=private`,
      headers: [{ name: "Content-Type", value: input.content_type }],
      expires_at: new Date(Date.now() + 60_000).toISOString(),
    },
  };
}

/** Real root, API service and PUT; only the two network edges wait for answers chosen by the test. */
export async function assetTab() {
  const tab = await issueStoreTab();
  const workspace = { ...tab.workspace, role: 20 as const };
  const project = { ...tab.project, member_role: 20 as const };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspace]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, [project]);
  runInAction(() => {
    tab.app.instance.config = {
      product: "Nerve",
      version: "test",
      commit: "test",
      api_version: "v0",
      signup_enabled: true,
      workspace_creation_enabled: true,
      file_size_limit: 10,
    };
  });
  tab.nerve.calls.length = 0;
  const assets = new Map<string, Asset>();
  let number = 0;
  const answer = (call: Call) => {
    if (call.path === "/api/v0/assets") {
      const prepared = preparedAsset(`asset-${++number}`, call.body as NewAsset);
      assets.set(prepared.asset.id, prepared.asset);
      call.answer(json(201, prepared));
    } else if (call.path.endsWith("/complete")) {
      const id = call.path.split("/").at(-2)!;
      call.answer(json(200, { ...assets.get(id), state: "uploaded", width: 2, height: 2 }));
    }
  };
  tab.nerve.respond = answer;
  const puts: {
    file: BodyInit | null | undefined;
    signal: AbortSignal | null | undefined;
    answer: (response: Response) => void;
    fail: () => void;
  }[] = [];
  vi.stubGlobal(
    "fetch",
    (_url: RequestInfo | URL, init?: RequestInit) =>
      new Promise<Response>((resolve, reject) => {
        puts.push({
          file: init?.body,
          signal: init?.signal,
          answer: resolve,
          fail: () => reject(new TypeError("network failed")),
        });
      })
  );
  const sent = async (result: { settled: boolean }, count = 1) => {
    await until(() => puts.length >= count || result.settled, "raw PUT or rejected operation");
    expect(puts).toHaveLength(count);
    return puts[count - 1]!;
  };
  return { ...tab, workspace, project, puts, answer, sent, assets };
}

/** Content owners enter the real canonical stores through their generated API clients. */
export async function contentAssetTab() {
  const tab = await assetTab();
  const { app, nerve, workspace, project } = tab;
  await seedAccount(nerve, app.user, accountOf("actor"));
  const issue = issueDetailOf("issue-1", project.id);
  const comment = { ...issueCommentOf("comment-1", issue.id), actor_id: app.issue.currentUserId! };
  const draft = { ...draftOf("draft-1", workspace.id), project_id: null };
  const readIssue = async (row: IssueDetail = issue) => {
    const result = await answered(
      nerve,
      () => app.issue.issues.refreshAuthoritative(row.id),
      ["GET", `/api/v0/issues/${row.id}`],
      row,
      "content issue"
    );
    expect(result.error).toBeUndefined();
  };
  const readComments = async (rows: IssueComment[] = [comment]) => {
    const result = await answered(
      nerve,
      () => app.issue.issueDetail.comment.fetchComments(workspace.slug, project.id, issue.id),
      ["GET", `/api/v0/issues/${issue.id}/comments`],
      { data: rows, next_cursor: null },
      "content comments"
    );
    expect(result.error).toBeUndefined();
  };
  const readDrafts = async (rows: Draft[] = [draft]) => {
    const result = await answered(
      nerve,
      () => app.issue.workspaceDraftIssues.fetchIssues(workspace.slug),
      ["GET", `/api/v0/workspaces/${workspace.slug}/drafts`],
      { data: rows, total: rows.length, next_cursor: null },
      "content drafts"
    );
    expect(result.error).toBeUndefined();
  };
  await readIssue();
  await readComments();
  await readDrafts();
  nerve.calls.length = 0;
  const scope = { workspaceID: workspace.id, projectID: project.id, issueID: issue.id };
  const hosts = {
    issue: { kind: "issue", ...scope },
    new_comment: { kind: "new_comment", ...scope },
    comment: { kind: "comment", ...scope, commentID: comment.id },
    draft: { kind: "draft", workspaceID: workspace.id, projectID: null, draftID: draft.id },
  } satisfies Record<"issue" | "new_comment" | "comment" | "draft", ContentHost>;
  return { ...tab, issue, comment, draft, hosts, readIssue, readComments, readDrafts };
}

export const attachmentOf = (id: string): Attachment => ({
  id,
  name: `${id}.txt`,
  content_type: "text/plain",
  size: 4,
  created_at: "2026-11-01T00:00:00Z",
  created_by_id: "u-actor",
});
export function contentAssetOf(id: string, host: ContentHost, use: "attachment" | "description"): Asset {
  const purpose =
    host.kind === "draft"
      ? use === "attachment"
        ? "DRAFT_ISSUE_ATTACHMENT"
        : "DRAFT_ISSUE_DESCRIPTION"
      : host.kind === "issue"
        ? use === "attachment"
          ? "ISSUE_ATTACHMENT"
          : "ISSUE_DESCRIPTION"
        : use === "attachment"
          ? "COMMENT_ATTACHMENT"
          : "COMMENT_DESCRIPTION";
  return {
    id,
    purpose,
    scope: {
      kind: host.kind,
      id: host.kind === "draft" ? host.draftID : host.kind === "comment" ? host.commentID : host.issueID,
    },
    name: `${id}.png`,
    size: 4,
    content_type: "image/png",
    state: "uploaded",
    is_bound: true,
    width: 2,
    height: 2,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  };
}
