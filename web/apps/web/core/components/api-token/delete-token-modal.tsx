/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// ui
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useApiTokens } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  tokenId: string;
};

export function DeleteApiTokenModal(props: Props) {
  const { isOpen, onClose, tokenId } = props;
  // store hooks
  const { revokeToken } = useApiTokens();
  // states
  const [deleteLoading, setDeleteLoading] = useState<boolean>(false);
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast("workspace_settings.settings.api_tokens.delete.error.title");

  const handleClose = () => {
    onClose();
    setDeleteLoading(false);
  };

  const handleDeletion = async () => {
    setDeleteLoading(true);
    try {
      await revokeToken(tokenId);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("workspace_settings.settings.api_tokens.delete.success.title"),
        message: t("workspace_settings.settings.api_tokens.delete.success.message"),
      });
      handleClose();
    } catch (error) {
      toastRefusal(error);
      setDeleteLoading(false);
    }
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={deleteLoading}
      isOpen={isOpen}
      title={t("workspace_settings.settings.api_tokens.delete.title")}
      content={<>{t("workspace_settings.settings.api_tokens.delete.description")} </>}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
    />
  );
}
