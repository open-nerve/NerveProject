/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import React, { useState } from "react";
import { observer } from "mobx-react";
// types
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

// ui
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useProject } from "@/hooks/store/use-project";

type Props = {
  data: Partial<IssueDetail>;
  isOpen: boolean;
  onClose: () => void;
  onSubmit: () => Promise<void>;
};

export const DeleteInboxIssueModal = observer(function DeleteInboxIssueModal({
  isOpen,
  onClose,
  onSubmit,
  data,
}: Props) {
  // states
  const [isDeleting, setIsDeleting] = useState(false);
  // store hooks
  const { getProjectById } = useProject();
  const { t } = useTranslation();
  // derived values
  const projectDetails = data.project_id ? getProjectById(data?.project_id) : undefined;

  const handleClose = () => {
    setIsDeleting(false);
    onClose();
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    await onSubmit()
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: `${t("success")}`,
          message: `${t("inbox_issue.modals.delete.success")}`,
        });
        return;
      })
      // a refusal is said by onSubmit, through the refusal toast
      .catch(() => undefined)
      .finally(() => handleClose());
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDelete}
      isSubmitting={isDeleting}
      isOpen={isOpen}
      title={t("inbox_issue.modals.delete.title")}
      // TODO: Need to translate the confirmation message
      content={
        <>
          Are you sure you want to delete work item{" "}
          <span className="font-medium break-words text-primary">
            {projectDetails?.identifier}-{data?.sequence_id}
          </span>
          {""}? The work item will only be deleted from the intake and this action cannot be undone.
        </>
      }
    />
  );
});
