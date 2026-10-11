/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { Label } from "@nerve/api-client";
// ui
import { AlertModalCore } from "@nerve/ui";
// local imports
import { useLabelChanges } from "./use-label-changes";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  data: Label | null;
};

export const DeleteLabelModal = observer(function DeleteLabelModal(props: Props) {
  const { isOpen, onClose, data } = props;
  const { remove } = useLabelChanges();
  // states
  const [isDeleteLoading, setIsDeleteLoading] = useState(false);

  const handleClose = () => {
    onClose();
    setIsDeleteLoading(false);
  };

  // Busy, and the dialog not to be dismissed, until the page has followed nerve's answer: it closes once nerve has
  // deleted the label, and stays, its buttons enabled again, when nerve refuses, the reason in a toast.
  const handleDeletion = async () => {
    if (!data) return;
    setIsDeleteLoading(true);
    await remove(data.id, handleClose);
    setIsDeleteLoading(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={isDeleteLoading}
      isOpen={isOpen}
      title="Delete Label"
      content={
        <>
          Are you sure you want to delete <span className="font-medium text-primary">{data?.name}</span>? This will
          remove the label from all the work item and from any views where the label is being filtered upon.
        </>
      }
    />
  );
});
