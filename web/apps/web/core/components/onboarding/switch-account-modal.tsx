/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { ArrowRightLeft } from "lucide-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
// hooks
import { useUser } from "@/hooks/store/user";
import { useSignOut } from "@/hooks/use-sign-out";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function SwitchAccountModal(props: Props) {
  const { isOpen, onClose } = props;
  const { t } = useTranslation();
  // states
  const [switchingAccount, setSwitchingAccount] = useState(false);
  // store hooks
  const { data: userData } = useUser();
  const signOut = useSignOut();

  const handleClose = () => {
    setSwitchingAccount(false);
    onClose();
  };

  const handleSwitchAccount = async () => {
    setSwitchingAccount(true);
    if (await signOut()) handleClose();
    setSwitchingAccount(false);
  };

  // While the sign-out is out the dialog cannot be dismissed (Escape, the backdrop): it closes once nerve has signed
  // the session out, and stays, the switch offered again, when it has not.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={switchingAccount ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
      className="overflow-hidden"
    >
      <div className="p-6 pb-1">
        <div className="flex gap-x-4">
          <div className="flex items-start">
            <div className="grid place-items-center rounded-full bg-accent-primary/20 p-4">
              <ArrowRightLeft className="h-5 w-5 text-accent-primary" aria-hidden="true" />
            </div>
          </div>
          <div className="flex flex-col gap-y-6 py-3">
            <ModalTitle className="text-20 leading-6 font-medium text-primary">
              {t("onboarding.switch_account.title")}
            </ModalTitle>
            {userData?.email && (
              <div className="text-14 font-regular text-secondary">
                <Translated
                  i18nKey="onboarding.switch_account.description"
                  values={{ email: <span className="text-accent-primary">{userData.email}</span> }}
                />
              </div>
            )}
          </div>
        </div>
      </div>
      <div className="mb-2 flex items-center justify-end gap-3 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleSwitchAccount} disabled={switchingAccount}>
          {switchingAccount ? t("onboarding.switch_account.switching") : t("onboarding.switch_account.switch")}
        </Button>
      </div>
    </ModalCore>
  );
}
