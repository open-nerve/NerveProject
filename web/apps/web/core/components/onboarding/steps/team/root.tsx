/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { AddOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Workspace, WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Spinner } from "@nerve/ui";
// components
import { InvitationFields } from "@/components/workspace/invite-modal/fields";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspaceInvitationActions } from "@/hooks/use-workspace-invitation";
// local components
import { CommonOnboardingHeader } from "../common";
import { InvitationLinks } from "./links";

type Props = {
  /** The workspace the onboarding created, which the step invites to (M3 design 7.4). */
  workspace: Workspace;
  /** Ends the onboarding. */
  onDone: () => void;
};

/**
 * The invitation step (M3 design 7.4): the members page's invitation form, for the workspace the onboarding created;
 * once nerve has the invitations, their links, which the caller gives the people invited (v0 sends no email).
 */
export const InviteTeamStep = observer(function InviteTeamStep(props: Props) {
  const { workspace, onDone } = props;
  const { t } = useTranslation();
  // states
  const [sent, setSent] = useState<WorkspaceInvitation[]>();
  // store hooks
  const {
    workspace: { inviteMembersToWorkspace },
  } = useMember();
  const { control, fields, formState, remove, onFormSubmit, appendField } = useWorkspaceInvitationActions({
    invite: (data) => inviteMembersToWorkspace(workspace.slug, data),
    onSent: setSent,
  });

  if (sent) return <InvitationLinks invitations={sent} onDone={onDone} />;
  return (
    <form
      className="flex flex-col gap-10"
      onSubmit={onFormSubmit}
      onKeyDown={(e) => {
        if (e.code === "Enter") e.preventDefault();
      }}
    >
      <CommonOnboardingHeader title={t("onboarding.invite.title")} description={t("onboarding.invite.description")} />
      <div className="w-full text-13">
        <InvitationFields fields={fields} control={control} formState={formState} remove={remove} />
        <button
          type="button"
          className="flex items-center gap-1.5 bg-transparent text-13 font-medium text-accent-primary outline-accent-strong"
          onClick={appendField}
        >
          <AddOutline className="h-4 w-4" />
          {t("onboarding.invite.add_another")}
        </button>
      </div>
      <div className="mx-auto flex w-full flex-col items-center justify-center gap-4 px-8 sm:px-2">
        <Button variant="primary" type="submit" size="xl" className="w-full" disabled={formState.isSubmitting}>
          {formState.isSubmitting ? <Spinner height="20px" width="20px" /> : t("continue")}
        </Button>
        <Button variant="ghost" size="xl" className="w-full" onClick={onDone}>
          {t("onboarding.invite.later")}
        </Button>
      </div>
    </form>
  );
});
