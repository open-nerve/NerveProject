/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import { Collapsible } from "@makeplane/propel/components/collapsible";
import { useTranslation } from "@nerve/i18n";
// components
import { CountChip } from "@/components/common/count-chip";
import { MembersSettingsLoader } from "@/components/ui/loader/settings/members";
// hooks
import { useMember } from "@/hooks/store/use-member";
// local imports
import { WorkspaceInvitationsListItem } from "./invitations-list-item";
import { WorkspaceMembersListItem } from "./members-list-item";
import { useMembersSettingsFetch } from "./use-members-settings-fetch";

export const WorkspaceMembersList = observer(function WorkspaceMembersList(props: {
  searchQuery: string;
  isAdmin: boolean;
}) {
  const { searchQuery, isAdmin } = props;
  const [showPendingInvites, setShowPendingInvites] = useState<boolean>(true);

  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const {
    workspace: {
      workspaceMemberIds,
      getFilteredWorkspaceMemberIds,
      getSearchedWorkspaceMemberIds,
      workspaceMemberInvitationIds,
      getSearchedWorkspaceInvitationIds,
      getWorkspaceMemberDetails,
    },
  } = useMember();
  const { t } = useTranslation();
  useMembersSettingsFetch(workspaceSlug);

  if (!workspaceMemberIds && !workspaceMemberInvitationIds) return <MembersSettingsLoader />;

  // derived values
  const filteredMemberIds = workspaceSlug ? getFilteredWorkspaceMemberIds(workspaceSlug) : [];
  const searchedMemberIds = searchQuery ? getSearchedWorkspaceMemberIds(searchQuery) : filteredMemberIds;
  const searchedInvitationsIds = getSearchedWorkspaceInvitationIds(searchQuery);
  const searchedMembers = searchedMemberIds?.map((memberId) => getWorkspaceMemberDetails(memberId)) ?? [];
  // the active members first, then those whose membership ended, each in the order the filters give
  const memberDetails = [
    ...searchedMembers.filter((member) => member?.is_active),
    ...searchedMembers.filter((member) => !member?.is_active),
  ];

  return (
    <>
      <div className="divide-y-[0.5px] divide-subtle overflow-scroll">
        {searchedMemberIds?.length !== 0 && <WorkspaceMembersListItem memberDetails={memberDetails} />}
        {searchedInvitationsIds?.length === 0 && searchedMemberIds?.length === 0 && (
          <h4 className="mt-16 text-center text-body-xs-regular text-placeholder">{t("no_matching_members")}</h4>
        )}
      </div>
      {isAdmin && searchedInvitationsIds && searchedInvitationsIds.length > 0 && (
        <div className="pt-4">
          <Collapsible
            open={showPendingInvites}
            onOpenChange={setShowPendingInvites}
            trigger={
              <span className="inline-flex items-center gap-2 py-2">
                <span className="text-h5-medium">{t("workspace_settings.settings.members.pending_invites")}</span>
                {searchedInvitationsIds && <CountChip count={searchedInvitationsIds.length} className="h-5" />}
              </span>
            }
          >
            <div className="ml-auto items-center gap-1.5 rounded-md bg-surface-1 py-1.5">
              {searchedInvitationsIds?.map((invitationId) => (
                <WorkspaceInvitationsListItem key={invitationId} invitationId={invitationId} />
              ))}
            </div>
          </Collapsible>
        </div>
      )}
    </>
  );
});
