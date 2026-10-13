/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
// hooks
import { useUser } from "@/hooks/store/user";
// local imports
import { ConfirmWorkspaceMemberRemove } from "./confirm-workspace-member-remove";
import { useMembershipChanges } from "./settings/use-membership-changes";

type Props = {
  /** The workspace the caller leaves. */
  workspaceSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

/**
 * The workspace menu's leaving of the workspace of workspaceSlug, for every role (M3 closeout): the dialog and the
 * leaving the members page offers on the caller's own row. Closed once nerve has ended the membership, the root then
 * landing him; one nerve refuses, its only admin's, leaves it open, the reason in the refusal's toast.
 */
export const LeaveWorkspaceModal = observer(function LeaveWorkspaceModal(props: Props) {
  const { workspaceSlug, isOpen, onClose } = props;
  const { data: currentUser } = useUser();
  const { leave } = useMembershipChanges(workspaceSlug);

  if (!currentUser) return null;
  return (
    <ConfirmWorkspaceMemberRemove
      isOpen={isOpen}
      onClose={onClose}
      userDetails={{ id: currentUser.id, display_name: currentUser.display_name }}
      onSubmit={() => leave(onClose)}
    />
  );
});
