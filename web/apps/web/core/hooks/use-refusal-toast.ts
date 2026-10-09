/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// lib
import { errorMessageKey } from "@/lib/error-messages";

/**
 * Says in a toast why a change failed (M2 design 7.3): nerve's refusal by its problem's code, or why nerve could not be
 * reached (errorMessageKey), under the title of titleKey. Every setToast that says a failure by errorMessageKey is this
 * one (the keyword rule refusal-toast); the theme switcher's promise toast says its failure itself (setPromiseToast,
 * P9 spec §5). A page that shows some refusals under its fields decides that itself, and calls this for the others.
 */
export function useRefusalToast(titleKey = "toast.error"): (error: unknown) => void {
  const { t } = useTranslation();
  return (error) => {
    setToast({ type: TOAST_TYPE.ERROR, title: t(titleKey), message: t(errorMessageKey(error)) });
  };
}
