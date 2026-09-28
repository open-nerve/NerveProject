/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { IWorkspaceMemberInvitation } from "@nerve/types";
import { Checkbox } from "@makeplane/propel/components/checkbox";
import { Spinner } from "@nerve/ui";
import { truncateText } from "@nerve/utils";
// constants
import { WorkspaceLogo } from "@/components/workspace/logo";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserSettings } from "@/hooks/store/user";
// services
import { WorkspaceService } from "@/services/workspace.service";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  invitations: IWorkspaceMemberInvitation[];
  handleNextStep: () => Promise<void>;
  handleCurrentViewChange: () => void;
};
const workspaceService = new WorkspaceService();

export function WorkspaceJoinInvitesStep(props: Props) {
  const { invitations, handleNextStep, handleCurrentViewChange } = props;
  const { t } = useTranslation();
  // states
  const [isJoiningWorkspaces, setIsJoiningWorkspaces] = useState(false);
  const [invitationsRespond, setInvitationsRespond] = useState<string[]>([]);
  // store hooks
  const { fetchWorkspaces } = useWorkspace();
  const { fetchCurrentUserSettings } = useUserSettings();

  // handle invitation
  const handleInvitation = (workspace_invitation: IWorkspaceMemberInvitation, action: "accepted" | "withdraw") => {
    if (action === "accepted") {
      setInvitationsRespond((prevData) => [...prevData, workspace_invitation.id]);
    } else if (action === "withdraw") {
      setInvitationsRespond((prevData) => prevData.filter((item: string) => item !== workspace_invitation.id));
    }
  };

  // submit invitations
  const submitInvitations = async () => {
    const invitation = invitations?.find((item) => item.id === invitationsRespond[0]);

    if (invitationsRespond.length <= 0 && !invitation?.role) return;

    setIsJoiningWorkspaces(true);

    try {
      await workspaceService.joinWorkspaces({ invitations: invitationsRespond });
      await fetchWorkspaces();
      await fetchCurrentUserSettings();
      await handleNextStep();
    } catch (error: any) {
      console.error(error);
      setIsJoiningWorkspaces(false);
    }
  };

  return invitations && invitations.length > 0 ? (
    <div className="flex flex-col gap-10">
      <CommonOnboardingHeader
        title={t("onboarding.workspace.join_title")}
        description={t("onboarding.workspace.description")}
      />
      <div className="flex flex-col gap-3">
        {invitations &&
          invitations.length > 0 &&
          invitations.map((invitation) => {
            const isSelected = invitationsRespond.includes(invitation.id);
            const invitedWorkspace = invitation.workspace;
            const checkboxId = `invitation-${invitation.id}`;
            // The row is the checkbox's label: a click anywhere on it, or Space on the checkbox, picks the
            // invitation (a label takes phrasing content only, so the row holds spans)
            return (
              <label
                key={invitation.id}
                htmlFor={checkboxId}
                className="flex cursor-pointer items-center gap-2 rounded-lg border border-subtle px-3 py-2 hover:bg-surface-2"
              >
                <span className="block flex-shrink-0">
                  <WorkspaceLogo
                    logo={invitedWorkspace?.logo_url}
                    name={invitedWorkspace?.name}
                    classNames="size-8 flex-shrink-0 rounded-lg"
                  />
                </span>
                <span className="block min-w-0 flex-1">
                  <span className="block text-13 font-medium">{truncateText(invitedWorkspace?.name, 30)}</span>
                  <span className="block text-11 text-secondary">{t(ROLE_DETAILS[invitation.role].i18n_title)}</span>
                </span>
                <span className="flex-shrink-0">
                  <Checkbox
                    id={checkboxId}
                    checked={isSelected}
                    onCheckedChange={(checked) => handleInvitation(invitation, checked ? "accepted" : "withdraw")}
                  />
                </span>
              </label>
            );
          })}
      </div>
      <div className="flex flex-col gap-4">
        <Button
          variant="primary"
          size="xl"
          className="w-full"
          onClick={submitInvitations}
          disabled={isJoiningWorkspaces || !invitationsRespond.length}
        >
          {isJoiningWorkspaces ? <Spinner height="20px" width="20px" /> : t("continue")}
        </Button>
        <Button
          variant="ghost"
          size="xl"
          className="w-full"
          onClick={handleCurrentViewChange}
          disabled={isJoiningWorkspaces}
        >
          {t("onboarding.workspace.create_new")}
        </Button>
      </div>
    </div>
  ) : (
    <div>{t("onboarding.workspace.no_invitations")}</div>
  );
}
