/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useMember } from "@/hooks/store/use-member";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * The changes the members page makes to a workspace's invitations (M3 design 7.5), each to the invitation of
 * invitationId in the workspace of workspaceSlug, as the store's changes take them: a pending invitation's role, and
 * the deletion of an invitation, pending or declined. Each shows nerve's refusal as its reason; the page follows each
 * only in the session it was sent in (M3 design 7.1): once another tab has moved this one to another account, the
 * page is that account's, and says nothing of the change.
 */
export function useInvitationChanges() {
  const {
    workspace: { updateMemberInvitation, deleteMemberInvitation },
  } = useMember();
  const { t } = useTranslation();
  const failed = (error: unknown) =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });

  return {
    /** Gives the pending invitation the role: its number, as nerve's invitation update takes it. */
    changeRole: (workspaceSlug: string, invitationId: string, role: WorkspaceRole) =>
      followInSession(() => updateMemberInvitation(workspaceSlug, invitationId, { role }), { failed }),
    /** Deletes the invitation, and says so. */
    remove: (workspaceSlug: string, invitationId: string) =>
      followInSession(() => deleteMemberInvitation(workspaceSlug, invitationId), {
        done: () =>
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitation removed successfully." }),
        failed,
      }),
  };
}
