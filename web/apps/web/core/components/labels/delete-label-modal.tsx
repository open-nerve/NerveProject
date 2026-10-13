/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// ui
import { AlertModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
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
  const { t } = useTranslation();
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
      title={t("project_settings.labels.delete.title")}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
      content={
        <Translated
          i18nKey="project_settings.labels.delete.body"
          values={{ name: <span className="font-medium text-primary">{data?.name}</span> }}
        />
      }
    />
  );
});
