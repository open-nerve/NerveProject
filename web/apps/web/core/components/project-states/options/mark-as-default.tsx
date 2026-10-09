/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { TStateOperationsCallbacks } from "@nerve/types";
import { cn } from "@nerve/utils";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";

type TStateMarksAsDefault = {
  stateId: string;
  isDefault: boolean;
  markStateAsDefaultCallback: TStateOperationsCallbacks["markStateAsDefault"];
};

export const StateMarksAsDefault = observer(function StateMarksAsDefault(props: TStateMarksAsDefault) {
  const { stateId, isDefault, markStateAsDefaultCallback } = props;
  const toastRefusal = useRefusalToast();
  // states
  const [isLoading, setIsLoading] = useState(false);

  const handleMarkAsDefault = async () => {
    if (!stateId || isDefault) return;
    setIsLoading(true);

    try {
      await markStateAsDefaultCallback(stateId);
    } catch (error) {
      toastRefusal(error);
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
