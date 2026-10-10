/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { ProjectRole } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * The changes of the memberships of the project of projectId, in the workspace of workspaceSlug, that its pages make
 * (M3 design 7.6): a member's role, a member's removal, and the caller's leaving. Each shows nerve's refusal as its
 * reason; the page follows each only in the session it was sent in (M3 design 7.1): once another tab has moved this
 * one to another account, the page is that account's, and says nothing of the change. A dialog that sent the removal
 * or the leaving gives done, its closing, which follows nerve's having made it alone: one nerve refuses leaves it open.
 */
export function useProjectMembershipChanges(workspaceSlug: string, projectId: string) {
  const navigate = useNavigate();
  const { getProjectById, leaveProject } = useProject();
  const {
    project: { updateMemberRole, removeMemberFromProject },
  } = useMember();
  const failed = useRefusalToast();

  return {
    /** Gives the member of userId the role: its number, as nerve's ProjectMemberUpdate takes it. */
    changeRole: (userId: string, role: ProjectRole) =>
      followInSession(() => updateMemberRole(projectId, userId, role), { failed }),
    /** Ends the membership of the member of userId; once nerve has, done. */
    remove: (userId: string, done?: () => void) =>
      followInSession(() => removeMemberFromProject(projectId, userId), { done, failed }),
    /**
     * Ends the caller's own membership; once nerve has, and only then, done, and the workspace's projects show (M1-P4:
     * a leaving nerve refuses, its only admin's, leaves him where he was, its member still). Settles once they show.
     */
    leave: async (done?: () => void) => {
      const project = getProjectById(projectId);
      if (!project) return;
      await followInSession(() => leaveProject(project), {
        done: () => {
          done?.();
          return navigate(`/${workspaceSlug}/projects`);
        },
        failed,
      });
    },
  };
}
