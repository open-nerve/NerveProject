/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useMemo } from "react";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

/** Local UI lifetime only. Permission facts and their ABA watermark still belong to IssueRootStore. */
export function useDetailScope(
  workspaceSlug: string,
  project: string | undefined,
  issue: string,
  member = false
): () => () => boolean {
  const { rootIssueStore: root } = useIssueDetail();
  const owner = useMemo(
    () => ({ active: false, epoch: 0, root, workspaceSlug, project, issue, member }),
    [root, workspaceSlug, project, issue, member]
  );
  useLayoutEffect(() => {
    owner.active = true;
    owner.epoch += 1;
    return () => {
      owner.active = false;
    };
  }, [owner]);
  return useCallback(() => {
    if (!owner.active || !root.active) return () => false;
    const epoch = owner.epoch;
    const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    const projects = project ? [project] : [];
    const identity = root.permissionReads.identity(workspace?.id, projects, member);
    // A successful delete invalidates reads itself; that is not a change of the UI's permission or host.
    return () =>
      owner.active &&
      owner.epoch === epoch &&
      root.active &&
      !!workspace &&
      root.permissionReads.identity(workspace.id, projects, member) === identity &&
      (!project ||
        root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, project) !== undefined);
  }, [owner, root, workspaceSlug, project, member]);
}
