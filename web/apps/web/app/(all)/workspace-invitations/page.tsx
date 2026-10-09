/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { ROLE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { BoxesOutline, CloseOutline, LogOutOutline, TickOutline, UserOutline } from "@makeplane/propel/icons";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { EmptySpace, EmptySpaceItem } from "@/components/ui/empty-space";
import { invitationView } from "@/components/workspace/invitation-view";
import { useInvitationAnswer } from "@/components/workspace/use-invitation-answer";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
import { useInvitationPreview } from "@/hooks/use-invitation-preview";
// lib
import { invitationAuthPath } from "@/lib/invitation-link";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

/**
 * The page an invitation's link opens (M3 design 7.4, W5): what invitationView decides, from the link, its invitation
 * as the link shows it, and the caller.
 */
function WorkspaceInvitationPage() {
  // query params: the invitation's link
  const [searchParams] = useSearchParams();
  const link = { invitationId: searchParams.get("invitation_id"), token: searchParams.get("token") };
  // store hooks
  const { data: currentUser, signOut } = useUser();
  const { t } = useTranslation();
  // whether nerve answered the caller's answer that the invitation is another address's (decision 1)
  const [mismatched, setMismatched] = useState(false);

  const preview = useInvitationPreview(link.invitationId, link.token);
  const view = invitationView({
    link,
    preview: { data: preview.data, error: preview.error },
    signedIn: currentUser !== undefined,
    mismatched,
  });

  // the caller's answers, followed only in the session they were sent in (use-invitation-answer.ts)
  const { accept, decline } = useInvitationAnswer({
    reread: () => void preview.mutate(),
    mismatched: () => setMismatched(true),
  });

  const home = currentUser ? (
    <EmptySpaceItem Icon={BoxesOutline} title={t("workspace_invitation.home")} href="/" />
  ) : (
    <EmptySpaceItem Icon={UserOutline} title={t("workspace_invitation.sign_in")} href="/" />
  );

  const content = () => {
    switch (view.kind) {
      case "loading":
        return <LogoSpinner />;
      case "invalid":
        return (
          <EmptySpace
            title={t("workspace_invitation.invalid.title")}
            description={t("workspace_invitation.invalid.description")}
          >
            {home}
          </EmptySpace>
        );
      case "unavailable":
        return (
          <EmptySpace title={t("errors.unreachable")} description="">
            <EmptySpaceItem Icon={BoxesOutline} title={t("common.retry")} action={() => void preview.mutate()} />
          </EmptySpace>
        );
      case "declined":
        return (
          <EmptySpace
            title={t("workspace_invitation.declined.title")}
            description={t("workspace_invitation.declined.description", { workspace: view.invitation.workspace_name })}
          >
            {home}
          </EmptySpace>
        );
      case "sign-in":
        return (
          <EmptySpace
            title={t("workspace_invitation.invited", {
              workspace: view.invitation.workspace_name,
              role: ROLE[view.invitation.role],
            })}
            description={t("workspace_invitation.addressed")}
          >
            <EmptySpaceItem
              Icon={UserOutline}
              title={t("workspace_invitation.sign_in_to_accept")}
              href={invitationAuthPath("/", { id: view.invitation.id, token: view.token })}
            />
            <EmptySpaceItem
              Icon={UserOutline}
              title={t("workspace_invitation.sign_up_to_accept")}
              href={invitationAuthPath("/sign-up", { id: view.invitation.id, token: view.token })}
            />
          </EmptySpace>
        );
      case "mismatch":
        return (
          <EmptySpace
            title={t("errors.workspace_invitation_email_mismatch")}
            description={t("workspace_invitation.mismatch")}
          >
            <EmptySpaceItem Icon={LogOutOutline} title={t("sign_out")} action={() => void signOut()} />
          </EmptySpace>
        );
      case "answer": {
        const { invitation, token } = view;
        return (
          <EmptySpace
            title={t("workspace_invitation.invited", {
              workspace: invitation.workspace_name,
              role: ROLE[invitation.role],
            })}
            description={t("workspace_invitation.description")}
          >
            <EmptySpaceItem
              Icon={TickOutline}
              title={t("workspace_invitation.accept")}
              action={() => void accept(invitation.id, token)}
            />
            <EmptySpaceItem
              Icon={CloseOutline}
              title={t("workspace_invitation.ignore")}
              action={() => void decline(invitation.id, token)}
            />
          </EmptySpace>
        );
      }
    }
  };

  return (
    <AuthenticationWrapper pageType={EPageTypes.PUBLIC}>
      <div className="flex h-full w-full flex-col items-center justify-center px-3">{content()}</div>
    </AuthenticationWrapper>
  );
}

export default observer(WorkspaceInvitationPage);
