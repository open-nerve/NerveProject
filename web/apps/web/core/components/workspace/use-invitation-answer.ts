/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useOpenWorkspace } from "@/hooks/use-open-workspace";
// lib
import { ApiError } from "@/lib/api-error";
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * What the invitation page does with nerve's word on an answer: it reads the invitation again, or it says the
 * invitation is another address's (decision 1).
 */
type AnswerFollowUps = { reread: () => void; mismatched: () => void };

/**
 * The answers the invitation page gives to the invitation of invitationId, with its link's token (M3 design 7.4, W5):
 * accepted, the page opens the workspace, its membership's or the caller's own already (M3 design 3.8); declined, it
 * reads the invitation again, which says so. nerve's refusal: another address's invitation, which the page then says;
 * else its reason, and the invitation read again, as nerve has it now (answered meanwhile, or deleted). The page
 * follows an answer only in the session it was sent in (M3 design 7.1): once another tab has moved this one to
 * another account, the page is that account's, and does nothing of the answer.
 */
export function useInvitationAnswer({ reread, mismatched }: AnswerFollowUps) {
  const { acceptInvitation, declineInvitation } = useWorkspace();
  const openWorkspace = useOpenWorkspace();
  const { t } = useTranslation();
  const refused = (error: unknown) => {
    if (error instanceof ApiError && error.problem?.code === "workspace.invitation_email_mismatch") {
      mismatched();
      return;
    }
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    reread();
  };

  return {
    accept: (invitationId: string, token: string) =>
      followInSession(() => acceptInvitation(invitationId, token), {
        done: (workspace) => void openWorkspace(workspace),
        failed: refused,
      }),
    decline: (invitationId: string, token: string) =>
      followInSession(() => declineInvitation(invitationId, token), { done: () => reread(), failed: refused }),
  };
}
