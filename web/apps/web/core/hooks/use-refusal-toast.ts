/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// lib
import { errorMessageKey } from "@/lib/error-messages";

/**
 * Why a change failed (M2 design 7.3), as a toast says it: nerve's refusal by its problem's code, or why nerve could not
 * be reached (errorMessageKey), in the page's language. The message of every toast that says such a failure, a
 * promise toast's (setPromiseToast) too (the keyword rule refusal-toast).
 */
export function useRefusalMessage(): (error: unknown) => string {
  const { t } = useTranslation();
  return (error) => t(errorMessageKey(error));
}

/**
 * Says in a toast why a change failed, useRefusalMessage's, under the title of titleKey. Every setToast that says such
 * a failure is this one (the keyword rule refusal-toast). A page that shows some refusals under its fields decides that
 * itself, and calls this for the others.
 */
export function useRefusalToast(titleKey = "toast.error"): (error: unknown) => void {
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  return (error) => {
    setToast({ type: TOAST_TYPE.ERROR, title: t(titleKey), message: refusalMessage(error) });
  };
}
