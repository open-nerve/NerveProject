/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TickCircleOutline } from "@makeplane/propel/icons";
import React from "react";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import type { TPasswordCriterionKey } from "@nerve/utils";
import { cn, getPasswordStrength, getPasswordCriteria } from "@nerve/utils";
import type { TShownPasswordStrength } from "./helper";
import { getStrengthInfo, getFragmentColor } from "./helper";

/** The indicator's texts, which the caller translates: a message per shown strength, a label per rule. */
export type TPasswordStrengthLabels = {
  strength: Record<TShownPasswordStrength, string>;
  criteria: Record<TPasswordCriterionKey, string>;
};

interface PasswordStrengthIndicatorProps {
  password: string;
  labels: TPasswordStrengthLabels;
}

/** The strength and the rules of a password while it is not valid: nothing while it is empty, or once it is valid. */
export function PasswordStrengthIndicator({ password, labels }: PasswordStrengthIndicatorProps) {
  const strength = getPasswordStrength(password);
  if (strength === E_PASSWORD_STRENGTH.EMPTY || strength === E_PASSWORD_STRENGTH.STRENGTH_VALID) {
    return null;
  }
  const criteria = getPasswordCriteria(password);
  const strengthInfo = getStrengthInfo(strength);

  return (
    <div className={cn("space-y-3")}>
      {/* Strength Indicator */}
      <div className="space-y-2">
        <div className="flex w-full gap-1 transition-all duration-300 ease-linear">
          {[0, 1, 2].map((fragmentIndex) => (
            <div
              key={fragmentIndex}
              className={cn(
                "h-1 flex-1 rounded-xs transition-all duration-300 ease-in-out",
                getFragmentColor(fragmentIndex, strengthInfo.activeFragments)
              )}
            />
          ))}
        </div>

        {/* Strength Message */}
        <p className={cn("!text-13 font-medium", strengthInfo.textColor)}>{labels.strength[strength]}</p>
      </div>

      {/* Criteria list */}
      <div className="flex flex-wrap gap-2">
        {criteria.map((criterion) => (
          <div key={criterion.key} className="flex items-center gap-1.5">
            <div className="flex items-center justify-center p-0.5">
              <TickCircleOutline
                className={cn("h-3 w-3 flex-shrink-0", {
                  "text-success-primary": criterion.isValid,
                  "text-primary": !criterion.isValid,
                })}
              />
            </div>
            <span
              className={cn("!text-11", {
                "text-success-primary": criterion.isValid,
                "text-primary": !criterion.isValid,
              })}
            >
              {labels.criteria[criterion.key]}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
