/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useTranslation } from "@nerve/i18n";
// ui
import { Button } from "@nerve/propel/button";
import { DeleteOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useUser } from "@/hooks/store/user";
// lib
import { errorMessageKey } from "@/lib/error-messages";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function DeactivateAccountModal(props: Props) {
  const { isOpen, onClose } = props;
  // hooks
  const { t } = useTranslation();
  const { deactivateAccount } = useUser();

  // states
  const [isDeactivating, setIsDeactivating] = useState(false);
  // nerve's reason for refusing the deactivation (an i18n key), which the dialog says until it closes or confirms again
  const [refusal, setRefusal] = useState<string | undefined>(undefined);

  const handleClose = () => {
    setRefusal(undefined);
    onClose();
  };

  const handleDeleteAccount = async () => {
    setIsDeactivating(true);
    setRefusal(undefined);

    await deactivateAccount()
      .then((endedHere) => {
        // The deactivation ended the tab's session: the sign-in page takes over (AuthenticationWrapper), and shows
        // this. It ended none when another tab had moved this one to another account meanwhile, whose page this is
        // then, and hears nothing of it (M3 design 7.1).
        if (!endedHere) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("account_deactivated"),
        });
        handleClose();
        return;
      })
      // nerve's reason, such as the only admin's of a workspace or a project (W9). The dialog is a page's of the
      // session the deactivation was sent in: another tab's sign-in of another account unmounts it, the wrapper
      // waiting for that account (authentication-wrapper.tsx), so a refusal answered after says nothing there.
      .catch((error: unknown) => setRefusal(errorMessageKey(error)))
      .finally(() => setIsDeactivating(false));
  };

  // While the deactivation is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is said
  // in the dialog that sent it, and no dialog opened again offers Confirm while the request is out.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isDeactivating ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="">
          <div className="flex items-start gap-x-4">
            <div className="mt-3 grid place-items-center rounded-full bg-danger-subtle p-2 sm:mt-3 sm:p-2 md:mt-0 md:p-4 lg:mt-0 lg:p-4">
              <DeleteOutline
                className="h-4 w-4 text-danger-primary sm:h-4 sm:w-4 md:h-6 md:w-6 lg:h-6 lg:w-6"
                aria-hidden="true"
              />
            </div>
            <div>
              <h3 className="my-4 text-20 leading-6 font-medium text-primary">{t("deactivate_your_account")}</h3>
              <p className="mt-6 list-disc pr-4 text-14 font-regular text-secondary">
                {t("deactivate_your_account_description")}
              </p>
              {refusal && (
                <p role="alert" className="mt-4 pr-4 text-14 font-medium text-danger-primary">
                  {t(refusal)}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="mb-2 flex items-center justify-end gap-2 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isDeactivating}>
          {t("cancel")}
        </Button>
        {/* Disabled while the request is out: a second click would send the deactivation again */}
        <Button variant="error-fill" size="lg" onClick={handleDeleteAccount} loading={isDeactivating}>
          {isDeactivating ? t("deactivating") : t("confirm")}
        </Button>
      </div>
    </ModalCore>
  );
}
