/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useLayoutEffect, useState } from "react";
import { StoreContext } from "@/lib/store-context";
import { useContentUpload, useContentView } from "@/hooks/use-content-assets";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import type { ContentView } from "@/store/asset/content-view";
import type { CommentPreparations } from "@/store/asset/comment-preparations";
export function useCommentView(
  workspaceSlug: string,
  projectID: string | undefined,
  issueID: string,
  commentID?: string
) {
  const detail = useIssueDetail();
  const issue = detail.issue.getIssueDetailById(issueID);
  const workspace = detail.rootIssueStore.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const valid =
    !!issue &&
    !!workspace &&
    issue.workspace_id === workspace.id &&
    issue.project_id === projectID &&
    (!commentID || !!detail.comment.getCommentById(issueID, commentID));
  return useContentView(
    valid
      ? commentID
        ? { kind: "comment", workspaceID: workspace.id, projectID: issue.project_id, issueID, commentID }
        : { kind: "new_comment", workspaceID: workspace.id, projectID: issue.project_id, issueID }
      : undefined
  );
}
export function useCommentPreparations(
  view: ContentView | undefined,
  ids: readonly string[],
  changed: (ids: string[]) => void
) {
  const root = useContext(StoreContext);
  const upload = useContentUpload(view, "attachment");
  const [held, setHeld] = useState<{ view: ContentView; files: CommentPreparations }>();
  useLayoutEffect(() => {
    if (!view?.current) return;
    const files = root.asset.commentPreparations(view, ids, changed);
    setHeld({ view, files });
    void files.load().catch(() => undefined);
    return () => files.dispose();
    // IDs seed a new view only; current-view selection is owned by the resource.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root, view, changed]);
  const files = view?.current && held?.view === view ? held.files : undefined;
  const transferIncomplete = !!upload && !["idle", "cancelled", "ready"].includes(upload.phase);
  return { files, upload, ready: !!files?.ready && !transferIncomplete };
}
