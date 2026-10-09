/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useUser } from "@/hooks/store/user";

/**
 * Signs the caller out (the user store's signOut, M2 design 7.1): resolves true once signed out. When nerve's logout
 * fails the session stays, and a toast says so: resolves false then. Never rejects. The one place a page signs out.
 */
export function useSignOut(): () => Promise<boolean> {
  const { signOut } = useUser();
  const { t } = useTranslation();
  return () =>
    signOut().then(
      () => true,
      () => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("auth.sign_out.toast.error.title"),
          message: t("auth.sign_out.toast.error.message"),
        });
        return false;
      }
    );
}
