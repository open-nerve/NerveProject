/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// local imports
import { useStateChanges } from "../use-state-changes";

type TStateMarksAsDefault = {
  stateId: string;
  isDefault: boolean;
};

export const StateMarksAsDefault = observer(function StateMarksAsDefault(props: TStateMarksAsDefault) {
  const { stateId, isDefault } = props;
  const { markDefault } = useStateChanges();
  const { t } = useTranslation();
  // states
  const [isLoading, setIsLoading] = useState(false);

  // busy until the page has followed nerve's answer; a refusal is said in a toast
  const handleMarkAsDefault = async () => {
    if (isDefault) return;
    setIsLoading(true);
    await markDefault(stateId);
    setIsLoading(false);
  };

  return (
    <button
      type="button"
      className={cn(
        "text-11 whitespace-nowrap transition-colors",
        isDefault ? "text-tertiary" : "text-secondary hover:text-primary"
      )}
      disabled={isDefault || isLoading}
      onClick={handleMarkAsDefault}
    >
      {t(
        isLoading
          ? "project_settings.states.default.marking"
          : isDefault
            ? "project_settings.states.default.is_default"
            : "project_settings.states.default.mark"
      )}
    </button>
  );
});
