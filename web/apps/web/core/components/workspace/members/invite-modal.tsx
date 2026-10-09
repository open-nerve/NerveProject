/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import type { WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { EModalWidth, EModalPosition, ModalCore } from "@nerve/ui";
// components
import { InvitationModalActions } from "@/components/workspace/invite-modal/actions";
import { InvitationFields } from "@/components/workspace/invite-modal/fields";
import { InvitationForm } from "@/components/workspace/invite-modal/form";
// hooks
import { useWorkspaceInvitationActions } from "@/hooks/use-workspace-invitation";

export type TSendWorkspaceInvitationModalProps = {
  isOpen: boolean;
  onClose: () => void;
  /** Sends the invitations of the form (the workspace's store, for the page's workspace). */
  invite: (data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
};

export const SendWorkspaceInvitationModal = observer(function SendWorkspaceInvitationModal(
  props: TSendWorkspaceInvitationModalProps
) {
  const { isOpen, onClose, invite } = props;
  // store hooks
  const { t } = useTranslation();
  // derived values
  const { control, fields, formState, remove, onFormSubmit, clear, appendField } = useWorkspaceInvitationActions({
    invite,
    onSent: onClose,
  });

  // the form empties once the modal has gone (its leave transition)
  const handleClose = () => {
    onClose();
    const timeout = setTimeout(() => {
      clear();
      clearTimeout(timeout);
    }, 350);
  };

  return (
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <InvitationForm
        title={t("workspace_settings.settings.members.modal.title")}
        description={t("workspace_settings.settings.members.modal.description")}
        onSubmit={onFormSubmit}
        actions={
          <InvitationModalActions
            isSubmitting={formState.isSubmitting}
            handleClose={handleClose}
            appendField={appendField}
          />
        }
        className="p-5"
      >
        <InvitationFields fields={fields} control={control} formState={formState} remove={remove} />
      </InvitationForm>
    </ModalCore>
  );
});
