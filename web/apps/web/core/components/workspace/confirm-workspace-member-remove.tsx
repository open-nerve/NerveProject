/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { WarningTriangleOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
// hooks
import { useUser } from "@/hooks/store/user";

export type Props = {
  isOpen: boolean;
  onClose: () => void;
  /** Makes the removal or the leaving; settles once the page has followed it: its follow-up of success closes this. */
  onSubmit: () => Promise<void>;
  userDetails: {
    id: string;
    display_name: string;
  };
};

export const ConfirmWorkspaceMemberRemove = observer(function ConfirmWorkspaceMemberRemove(props: Props) {
  const { isOpen, onClose, onSubmit, userDetails } = props;
  // states
  const [isRemoving, setIsRemoving] = useState(false);
  // store hooks
  const { data: currentUser } = useUser();
  const { t } = useTranslation();

  const handleClose = () => {
    onClose();
    setIsRemoving(false);
  };

  const handleDeletion = async () => {
    setIsRemoving(true);
    await onSubmit();
    setIsRemoving(false);
  };

  // While the removal or the leaving is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): it closes
  // once nerve has made the change, and no dialog opened again offers it while the request is out. One nerve refuses
  // leaves it open, its buttons enabled again, the reason in the refusal's toast (A-M2).
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isRemoving ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="sm:flex sm:items-start">
          <div className="mx-auto flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-danger-subtle sm:mx-0 sm:h-10 sm:w-10">
            <WarningTriangleOutline className="h-6 w-6 text-danger-primary" aria-hidden="true" />
          </div>
          <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
            <ModalTitle className="text-h5-medium leading-6 text-primary">
              {currentUser?.id === userDetails.id
                ? t("workspace_settings.settings.members.leave_title")
                : t("workspace_settings.settings.members.remove_title", { name: userDetails.display_name })}
            </ModalTitle>
            <div className="mt-2">
              {currentUser?.id === userDetails.id ? (
                <p className="text-body-xs-regular text-secondary">
                  {t("workspace_settings.settings.members.leave_confirmation")}
                </p>
              ) : (
                <p className="text-body-xs-regular text-secondary">
                  <Translated
                    i18nKey="workspace_settings.settings.members.remove_confirmation"
                    values={{ name: <span className="font-bold">{userDetails.display_name}</span> }}
                  />
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="flex justify-end gap-2 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isRemoving}>
          {t("cancel")}
        </Button>
        <Button variant="error-fill" size="lg" onClick={handleDeletion} loading={isRemoving}>
          {currentUser?.id === userDetails.id
            ? isRemoving
              ? t("leaving")
              : t("leave")
            : isRemoving
              ? t("removing")
              : t("remove")}
        </Button>
      </div>
    </ModalCore>
  );
});
