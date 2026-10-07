/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { WorkspaceMember } from "@nerve/api-client";
import { Table } from "@nerve/ui";
// components
import { MembersLayoutLoader } from "@/components/ui/loader/layouts/members-layout-loader";
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser } from "@/hooks/store/user";
import { useNavigate } from "react-router";
// components
import { useMemberColumns } from "@/components/workspace/settings/useMemberColumns";

type Props = {
  memberDetails: (WorkspaceMember | null)[];
};

export const WorkspaceMembersListItem = observer(function WorkspaceMembersListItem(props: Props) {
  const { memberDetails } = props;
  const { columns, workspaceSlug, removeMemberModal, setRemoveMemberModal } = useMemberColumns();
  // router
  const navigate = useNavigate();
  // store hooks
  const { data: currentUser } = useUser();
  const {
    workspace: { removeMemberFromWorkspace },
  } = useMember();
  const { currentWorkspace, leaveWorkspace } = useWorkspace();
  const { t } = useTranslation();
  // derived values

  const handleLeaveWorkspace = async () => {
    if (!currentWorkspace || !currentUser) return;

    try {
      await leaveWorkspace(currentWorkspace);
      // the root lands the caller where his workspaces, as they are now, say (M3 design 3.14)
      navigate("/");
    } catch (err: unknown) {
      const error = err as { error?: string };
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: error?.error || t("something_went_wrong_please_try_again"),
      });
    }
  };

  const handleRemoveMember = async (memberId: string) => {
    if (!workspaceSlug || !memberId) return;

    try {
      await removeMemberFromWorkspace(workspaceSlug, memberId);
    } catch (err: unknown) {
      const error = err as { error?: string };
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: error?.error || t("something_went_wrong_please_try_again"),
      });
    }
  };

  const handleRemove = async (memberId: string) => {
    if (memberId === currentUser?.id) await handleLeaveWorkspace();
    else await handleRemoveMember(memberId);
  };

  // is the member current logged in user
  // const isCurrentUser = memberDetails?.member.id === currentUser?.id;
  // is the current logged in user admin
  // role change access-
  // 1. user cannot change their own role
  // 2. only admin or member can change role
  // 3. user cannot change role of higher role

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
