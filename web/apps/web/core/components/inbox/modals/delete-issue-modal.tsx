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
  const identifier = `${projectDetails?.identifier ?? ""}-${data.sequence_id ?? ""}`;

  const handleClose = () => {
    setIsDeleting(false);
    onClose();
  };

  // Once nerve has deleted the work item, the selection leaves it and the modal goes with it; a refusal, said by
  // onSubmit through the refusal toast, leaves the modal open, the focus in it, to try again or cancel, but for one
  // saying the work item is gone from the intake, on which the selection leaves it, the modal with it.
  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onSubmit();
    } catch {
      setIsDeleting(false);
      return;
    }
    setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("inbox_issue.modals.delete.success") });
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDelete}
      isSubmitting={isDeleting}
      isOpen={isOpen}
      title={t("inbox_issue.modals.delete.title")}
      // the pending and snoozed work items the page offers to delete are deleted, their intake rows with them
      content={t("inbox_issue.modals.delete.content", { identifier })}
      primaryButtonText={{ loading: t("deleting"), default: t("delete") }}
      secondaryButtonText={t("common.cancel")}
    />
  );
});
