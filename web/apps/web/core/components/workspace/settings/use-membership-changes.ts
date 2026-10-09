/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { WorkspaceRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * The changes the members page makes to the memberships of the workspace of workspaceSlug (M3 design 7.5): a member's
 * role, a member's removal, and the caller's leaving. Each shows nerve's refusal as its reason; the page follows each
 * only in the session it was sent in (M3 design 7.1): once another tab has moved this one to another account, the
 * page is that account's, and says nothing of the change.
 */
export function useMembershipChanges(workspaceSlug: string) {
  const navigate = useNavigate();
  const { getWorkspaceBySlug, leaveWorkspace } = useWorkspace();
  const {
    workspace: { updateMember, removeMemberFromWorkspace },
  } = useMember();
  const { t } = useTranslation();
  const failed = (error: unknown) =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });

  return {
    /** Gives the member of userId the role: its number, as nerve's WorkspaceMemberUpdate takes it. */
    changeRole: (userId: string, role: WorkspaceRole) =>
      followInSession(() => updateMember(workspaceSlug, userId, { role }), { failed }),
    /** Ends the membership of the member of userId. */
    remove: (userId: string) => followInSession(() => removeMemberFromWorkspace(workspaceSlug, userId), { failed }),
    /**
     * Ends the caller's own membership; then the root lands him where his workspaces, as they are now, say (M3 design
     * 3.14).
     */
    leave: async () => {
      const workspace = getWorkspaceBySlug(workspaceSlug);
      if (!workspace) return;
      await followInSession(() => leaveWorkspace(workspace), { done: () => void navigate("/"), failed });
    },
  };
}
