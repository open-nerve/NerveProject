/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Label } from "@nerve/api-client";
// ui
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useLabel } from "@/hooks/store/use-label";
// lib
import { errorMessageKey } from "@/lib/error-messages";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  data: Label | null;
};

export const DeleteLabelModal = observer(function DeleteLabelModal(props: Props) {
  const { isOpen, onClose, data } = props;
  // store hooks
  const { deleteLabel } = useLabel();
  const { t } = useTranslation();
  // states
  const [isDeleteLoading, setIsDeleteLoading] = useState(false);

  const handleClose = () => {
    onClose();
    setIsDeleteLoading(false);
  };

  const handleDeletion = async () => {
    if (!data) return;

    setIsDeleteLoading(true);

    await deleteLabel(data.id)
      .then(() => handleClose())
      .catch((error: unknown) => {
        setIsDeleteLoading(false);
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
      });
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
