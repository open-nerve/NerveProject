/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { cn } from "@nerve/utils";
// lib
import { errorMessageKey } from "@/lib/error-messages";

type TStateMarksAsDefault = {
  stateId: string;
  isDefault: boolean;
  markStateAsDefaultCallback: TStateOperationsCallbacks["markStateAsDefault"];
};

export const StateMarksAsDefault = observer(function StateMarksAsDefault(props: TStateMarksAsDefault) {
  const { stateId, isDefault, markStateAsDefaultCallback } = props;
  const { t } = useTranslation();
  // states
  const [isLoading, setIsLoading] = useState(false);

  const handleMarkAsDefault = async () => {
    if (!stateId || isDefault) return;
    setIsLoading(true);

    try {
      await markStateAsDefaultCallback(stateId);
    } catch (error) {
      setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <button
      className={cn(
        "text-11 whitespace-nowrap transition-colors",
        isDefault ? "text-tertiary" : "text-secondary hover:text-primary"
      )}
      disabled={isDefault || isLoading}
      onClick={handleMarkAsDefault}
    >
      {isLoading ? "Marking as default" : isDefault ? `Default` : `Mark as default`}
    </button>
  );
});
