/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { useCopyInvitationLink } from "@/hooks/use-copy-invitation-link";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  /** The invitations nerve made of the step's form. */
  invitations: WorkspaceInvitation[];
  /** Ends the onboarding. */
  onDone: () => void;
};

/** The links of the invitations the step sent, each to copy (M3 design 7.4: v0 sends no email). */
export function InvitationLinks({ invitations, onDone }: Props) {
  const { t } = useTranslation();
  const copyLink = useCopyInvitationLink();

  return (
    <div className="flex flex-col gap-10">
      <CommonOnboardingHeader
        title={t("onboarding.invite.links.title")}
        description={t("onboarding.invite.links.description")}
      />
      <ul className="flex flex-col gap-3">
        {invitations.map((invitation) => (
          <li key={invitation.id} className="flex items-center justify-between gap-4 text-13">
            <span className="truncate text-secondary">{invitation.email}</span>
            <Button variant="secondary" size="sm" onClick={() => void copyLink(invitation)}>
              {t("common.actions.copy_link")}
            </Button>
          </li>
        ))}
      </ul>
      <Button variant="primary" size="xl" className="w-full" onClick={onDone}>
        {t("continue")}
      </Button>
    </div>
  );
}
