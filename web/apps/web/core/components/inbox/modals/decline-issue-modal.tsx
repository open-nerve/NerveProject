/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import React, { useState } from "react";
// types
import { useTranslation } from "@nerve/i18n";

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

export function DeclineIssueModal(props: Props) {
  const { isOpen, onClose, data, onSubmit } = props;
  // states
  const [isDeclining, setIsDeclining] = useState(false);
  // store hooks
  const { getProjectById } = useProject();
  const { t } = useTranslation();
  // derived values
  const projectDetails = data.project_id ? getProjectById(data?.project_id) : undefined;
  const identifier = `${projectDetails?.identifier ?? ""}-${data.sequence_id ?? ""}`;

  const handleClose = () => {
    setIsDeclining(false);
    onClose();
  };

  const handleDecline = async () => {
    setIsDeclining(true);
    // nerve's refusal is said by onSubmit, the modal open to try again or cancel
    await onSubmit().catch(() => undefined);
    setIsDeclining(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDecline}
      isSubmitting={isDeclining}
      isOpen={isOpen}
      title={t("inbox_issue.modals.decline.title")}
      content={t("inbox_issue.modals.decline.content", { identifier })}
      primaryButtonText={{
        loading: t("declining"),
        default: t("decline"),
      }}
      secondaryButtonText={t("common.cancel")}
    />
  );
}
