/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { TPasswordStrengthLabels } from "@nerve/ui";

/** The password strength indicator's texts in the page's language, for each page that shows it. */
export const usePasswordStrengthLabels = (): TPasswordStrengthLabels => {
  const { t } = useTranslation();
  return {
    strength: {
      [E_PASSWORD_STRENGTH.LENGTH_NOT_VALID]: t("auth.common.password.strength.length_not_valid"),
      [E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID]: t("auth.common.password.strength.weak"),
    },
    criteria: {
      length: t("auth.common.password.strength.criteria.length"),
      uppercase: t("auth.common.password.strength.criteria.uppercase"),
      lowercase: t("auth.common.password.strength.criteria.lowercase"),
      number: t("auth.common.password.strength.criteria.number"),
      special: t("auth.common.password.strength.criteria.special"),
    },
  };
};
