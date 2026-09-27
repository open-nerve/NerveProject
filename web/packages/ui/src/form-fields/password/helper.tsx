/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { E_PASSWORD_STRENGTH } from "@nerve/constants";

/** The strengths the indicator shows: the password is not empty and not yet valid. */
export type TShownPasswordStrength = E_PASSWORD_STRENGTH.LENGTH_NOT_VALID | E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID;

type TActiveFragments = 1 | 2;

interface StrengthInfo {
  textColor: string;
  activeFragments: TActiveFragments;
}

/**
 * Get the message's color and the number of active fragments of a shown strength
 */
export const getStrengthInfo = (strength: TShownPasswordStrength): StrengthInfo => {
  switch (strength) {
    case E_PASSWORD_STRENGTH.LENGTH_NOT_VALID:
      return {
        textColor: "text-danger-primary",
        activeFragments: 1,
      };
    case E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID:
      return {
        textColor: "text-orange-500",
        activeFragments: 2,
      };
  }
};

/**
 * Get fragment color based on position and active state
 */
export const getFragmentColor = (fragmentIndex: number, activeFragments: TActiveFragments): string => {
  if (fragmentIndex >= activeFragments) {
    return "bg-layer-1";
  }

  switch (activeFragments) {
    case 1:
      return "bg-danger-primary";
    case 2:
      return "bg-orange-500";
  }
};
