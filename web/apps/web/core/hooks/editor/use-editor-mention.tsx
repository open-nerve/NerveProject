/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { Avatar } from "@makeplane/propel/components/avatar";
import type { TMentionSection } from "@nerve/editor";
import { getFileURL } from "@nerve/utils";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IMemberRootStore } from "@/store/member";

type Args = { workspaceSlug: string; projectId?: string; editable: boolean; members: IMemberRootStore };

/** Suggestions read native directories; the editor's captured callback follows the current host. */
export function useEditorMention({ workspaceSlug, projectId, editable, members }: Args) {
  const { rootIssueStore: root } = useIssueDetail();
  const scope = useDetailScope(workspaceSlug, projectId, "editor-mentions", !projectId);
  const owner = useMemo(() => ({ active: false, request: 0, scope, editable, members }), [scope, editable, members]);
  useLayoutEffect(() => {
    owner.active = true;
    return () => {
      owner.active = false;
    };
  }, [owner]);
  const fetchCurrent = useCallback(
    async (query: string): Promise<TMentionSection[]> => {
      if (!owner.editable) return [];
      const request = ++owner.request;
      const inScope = owner.scope();
      const current = () => owner.active && request === owner.request && inScope();
      const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
      if (!current() || !workspace || (!projectId && workspace.role !== 15 && workspace.role !== 20))
        throw new SessionChangedError();
      const project = projectId ? root.rootStore.projectRoot.project.getProjectById(projectId) : undefined;
      if (projectId && project?.workspace_id !== workspace.id) throw new SessionChangedError();
      try {
        const [workspaceMembers, projectMembers] = await Promise.all([
          owner.members.workspace.fetchWorkspaceMembers(workspace),
          projectId ? owner.members.project.fetchProjectMembers(projectId) : undefined,
        ]);
        if (!current()) throw new SessionChangedError();
        if (!workspaceMembers || (projectId && !projectMembers))
          throw new Error("Member directory read was superseded");
        const normalized = query.toLocaleLowerCase();
        const users = Object.values(workspaceMembers)
          .filter((row) => row.is_active && (!projectId || !!projectMembers?.[row.member.id]))
          .map((row) => row.member)
          .filter((user) => user.display_name.toLocaleLowerCase().includes(normalized))
          .slice(0, 5);
        if (!users.length) return [];
        return [
          {
            key: "users",
            title: "Users",
            items: users.map((user) => ({
              id: user.id,
              entity_identifier: user.id,
              entity_name: "user_mention",
              title: user.display_name,
              icon: (
                <Avatar
                  alt={user.display_name}
                  fallback={user.display_name[0]?.toUpperCase()}
                  src={getFileURL(user.avatar_url)}
                  size="xs"
                />
              ),
            })),
          },
        ];
      } catch (error) {
        if (!current()) throw new SessionChangedError();
        throw error;
      }
    },
    [owner, root, workspaceSlug, projectId]
  );
  // TipTap retains the initial extension options. Preserve its callback identity without retaining an old scope.
  const current = useRef(fetchCurrent);
  useLayoutEffect(() => {
    current.current = fetchCurrent;
  }, [fetchCurrent]);
  return { fetchMentions: useCallback((query: string) => current.current(query), []) };
}
