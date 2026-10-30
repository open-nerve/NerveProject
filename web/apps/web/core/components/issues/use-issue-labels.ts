/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
/** Project directories remain their owner's facts. Workspace presentation only derives from already held labels. */
export function useIssueLabels(workspaceSlug: string | undefined, projectId: string | null | undefined) {
  const { rootIssueStore: root } = useIssueDetail();
  const { workspaceRoot, projectRoot, label } = root.rootStore;
  const workspace = workspaceSlug ? workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  if (!root.active || !workspace) return undefined;
  if (projectId)
    return projectRoot.project.getProjectById(projectId)?.workspace_id === workspace.id
      ? label.getProjectLabels(projectId)
      : undefined;
  return Object.values(label.labelMap).filter((item) => item.workspace_id === workspace.id);
}
