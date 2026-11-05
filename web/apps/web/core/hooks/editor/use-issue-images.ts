/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useContentView } from "@/hooks/use-content-assets";
import { useEditorAssets } from "./use-editor-assets";
/** Only a fully read canonical issue can give an editor a content lifetime. */
export function useIssueImages(workspaceSlug: string, projectID: string, issueID: string, readOnly = false) {
  const issue = useIssueDetail().issue.getIssueDetailById(issueID);
  const workspace = useWorkspace().getWorkspaceBySlug(workspaceSlug);
  const view = useContentView(
    issue && workspace && issue.workspace_id === workspace.id && issue.project_id === projectID
      ? { kind: "issue", workspaceID: workspace.id, projectID, issueID }
      : undefined
  );
  const images = useEditorAssets(view, readOnly);
  return { view, images };
}
