/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
// nerve imports
import type { WorkspaceMember } from "@nerve/api-client";
import { Table } from "@nerve/ui";
// components
import { MembersLayoutLoader } from "@/components/ui/loader/layouts/members-layout-loader";
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useUser } from "@/hooks/store/user";
// components
import { useMemberColumns } from "@/components/workspace/settings/useMemberColumns";
// local imports
import { useMembershipChanges } from "./use-membership-changes";

type Props = {
  /** The workspace of the page's address, whose members these are. */
  workspaceSlug: string;
  memberDetails: (WorkspaceMember | null)[];
};

export const WorkspaceMembersListItem = observer(function WorkspaceMembersListItem(props: Props) {
  const { workspaceSlug, memberDetails } = props;
  const { columns, removeMemberModal, setRemoveMemberModal } = useMemberColumns(workspaceSlug);
  // store hooks
  const { data: currentUser } = useUser();
  const { leave, remove } = useMembershipChanges(workspaceSlug);

  // the caller's own row offers him to leave; another's, to remove that member (an admin's)
  const handleRemove = (memberId: string) => (memberId === currentUser?.id ? leave() : remove(memberId));

  if (isEmpty(columns)) return <MembersLayoutLoader />;

  return (
    <div className="grid border-t border-subtle">
      {removeMemberModal && (
        <ConfirmWorkspaceMemberRemove
          isOpen={removeMemberModal.member.id.length > 0}
          onClose={() => setRemoveMemberModal(null)}
          userDetails={{
            id: removeMemberModal.member.id,
            display_name: removeMemberModal.member.display_name || "",
          }}
          onSubmit={() => handleRemove(removeMemberModal.member.id)}
        />
      )}
      <Table<WorkspaceMember>
        columns={columns ?? []}
        data={memberDetails.filter((member): member is WorkspaceMember => member !== null)}
        keyExtractor={(rowData) => rowData?.member.id ?? ""}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-10 text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </div>
  );
});
