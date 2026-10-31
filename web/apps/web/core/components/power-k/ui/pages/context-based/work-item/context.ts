/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useParams } from "react-router";
import { EUserPermissions } from "@nerve/constants";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";

/** UI lifetime and current canonical reads; permissions and the write queue remain on IssueRootStore. */
export function useWorkItemCommandContext() {
  const { workspaceSlug = "", workItem = "" } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const id = root.issues.getIssueIdByIdentifier(workItem);
  const issue = id ? root.issues.getIssueById(id) : undefined;
  const projectId = issue?.project_id;
  const identity = root.permissionReads.identity(workspace?.id, projectId ? [projectId] : [], true);
  const capture = useDetailScope(workspaceSlug, projectId, JSON.stringify([workItem, id, identity]), true);
  const read = () => {
    if (!workspace || !id) return undefined;
    const held = root.issues.getIssueById(id);
    if (!held || held.workspace_id !== workspace.id || held.project_id !== projectId) return undefined;
    const project = root.rootStore.projectRoot.project.getProjectById(held.project_id);
    if (!project || project.workspace_id !== workspace.id) return undefined;
    const user = root.rootStore.user;
    const role = user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, project.id);
    const member = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
    return {
      issue: held,
      project,
      workspaceSlug,
      canEdit: member && !held.archived_at && !project.archived_at,
      canDelete:
        member && !project.archived_at && (held.created_by_id === user.data?.id || role === EUserPermissions.ADMIN),
    };
  };
  return { root, read, capture };
}
export type WorkItemCommandContext = ReturnType<typeof useWorkItemCommandContext>;
