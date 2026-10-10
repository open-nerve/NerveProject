/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Table } from "@nerve/ui";
// hooks
import { useUser } from "@/hooks/store/user";
// components
import { useProjectColumns } from "@/components/projects/settings/useProjectColumns";
// store
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
// local imports
import { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";
import { useProjectMembershipChanges } from "./settings/use-project-membership-changes";

type Props = {
  memberDetails: (IProjectMemberDetails | null)[];
  projectId: string;
  workspaceSlug: string;
};

export const ProjectMemberListItem = observer(function ProjectMemberListItem(props: Props) {
  const { memberDetails, projectId, workspaceSlug } = props;
  // store hooks
  const { data: currentUser } = useUser();
  const { remove, leave } = useProjectMembershipChanges(workspaceSlug, projectId);
  // helper hooks
  const { columns, removeMemberModal, setRemoveMemberModal } = useProjectColumns({
    projectId,
    workspaceSlug,
  });

  // the caller's own membership he leaves; another's he removes; the dialog that asked closes once nerve has done it
  const handleRemove = (memberId: string) => {
    const closeDialog = () => setRemoveMemberModal(null);
    return memberId === currentUser?.id ? leave(closeDialog) : remove(memberId, closeDialog);
  };

  if (!memberDetails) return null;
  return (
    <>
      {removeMemberModal && (
        <ConfirmProjectMemberRemove
          isOpen={removeMemberModal !== null}
          onClose={() => setRemoveMemberModal(null)}
          data={{ id: removeMemberModal.member.id, display_name: removeMemberModal.member.display_name || "" }}
          onSubmit={() => handleRemove(removeMemberModal.member.id)}
        />
      )}
      <Table
        columns={columns}
        data={memberDetails.filter((member): member is IProjectMemberDetails => member !== null)}
        keyExtractor={(rowData) => rowData?.member.id ?? ""}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-[40px] text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </>
  );
});
