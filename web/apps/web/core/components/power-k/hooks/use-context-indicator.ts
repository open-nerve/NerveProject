/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useModules } from "@/hooks/store/use-modules";
// local imports
import type { TPowerKContextType } from "../core/types";

type TArgs = {
  activeContext: TPowerKContextType | null;
};

export const useContextIndicator = (args: TArgs): string | null => {
  const { activeContext } = args;
  // navigation
  const { workItem: workItemIdentifier, cycleId, moduleId } = useParams();
  // store hooks
  const { getCycleById } = useCycles();
  const { getModuleById } = useModules();
  let indicator: string | undefined | null = null;

  switch (activeContext) {
    case "work-item": {
      indicator = workItemIdentifier ?? null;
      break;
    }
    case "cycle": {
      indicator = getCycleById(cycleId)?.name;
      break;
    }
    case "module": {
      indicator = getModuleById(moduleId)?.name;
      break;
    }
    default: {
      indicator = null;
    }
  }

  return indicator ?? null;
};
